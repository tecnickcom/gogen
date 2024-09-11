package valkey

import (
	"context"
	"fmt"
	"time"

	"github.com/Vonage/gosrvlib/pkg/encode"
	libvalkey "github.com/valkey-io/valkey-go"
)

// TEncodeFunc is the type of function used to replace the default message encoding function used by SendData().
type TEncodeFunc func(ctx context.Context, data any) (string, error)

// TDecodeFunc is the type of function used to replace the default message decoding function used by ReceiveData().
type TDecodeFunc func(ctx context.Context, msg string, data any) error

// SrvOptions is an alias for the parent library client options.
type SrvOptions = libvalkey.ClientOption

// VKMessage is an alias for the parent library Message type.
type VKMessage = libvalkey.PubSubMessage

// VKClient represents the mockable functions in the parent Valkey Client.
type VKClient = libvalkey.Client

// VKPubSub represents the mockable functions in the parent Valkey PubSub.
type VKPubSub = libvalkey.Completed

// Client is a wrapper for the Valkey Client.
type Client struct {
	// vkclient is the upstream Client.
	vkclient VKClient

	// vkpubsub is the upstream PubSub completed command.
	vkpubsub VKPubSub

	// messageEncodeFunc is the function used by SendData()
	// to encode and serialize the input data to a string compatible with Valkey.
	messageEncodeFunc TEncodeFunc

	// messageDecodeFunc is the function used by ReceiveData()
	// to decode a message encoded with messageEncodeFunc to the provided data object.
	// The value underlying data must be a pointer to the correct type for the next data item received.
	messageDecodeFunc TDecodeFunc
}

// New creates a new instance of the Valkey client wrapper.
func New(ctx context.Context, srvopt SrvOptions, opts ...Option) (*Client, error) {
	cfg, err := loadConfig(ctx, srvopt, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create a new valkey client: %w", err)
	}

	vkc := cfg.vkclient

	if vkc == nil {
		vkc, err = libvalkey.NewClient(cfg.srvOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to create Valkey client: %w", err)
		}
	}

	return &Client{
		vkclient:          vkc,
		vkpubsub:          vkc.B().Subscribe().Channel(cfg.channels...).Build().Pin(),
		messageEncodeFunc: cfg.messageEncodeFunc,
		messageDecodeFunc: cfg.messageDecodeFunc,
	}, nil
}

// Close closes the client.
// All pending calls will be finished.
func (c *Client) Close() {
	c.vkclient.Close()
}

// Set a raw string value for the specified key with an expiration time.
func (c *Client) Set(ctx context.Context, key string, value string, exp time.Duration) error {
	err := c.vkclient.Do(ctx, c.vkclient.B().Set().Key(key).Value(value).Ex(exp).Build()).Error()
	if err != nil {
		return fmt.Errorf("cannot set key: %s %w", key, err)
	}

	return nil
}

// Get retrieves the raw string value of the specified key.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.vkclient.Do(ctx, c.vkclient.B().Get().Key(key).Build()).ToString()
	if err != nil {
		return "", fmt.Errorf("cannot retrieve key %s: %w", key, err)
	}

	return value, nil
}

// Del deletes the specified key from the datastore.
func (c *Client) Del(ctx context.Context, key string) error {
	err := c.vkclient.Do(ctx, c.vkclient.B().Del().Key(key).Build()).Error()
	if err != nil {
		return fmt.Errorf("cannot delete key: %s %w", key, err)
	}

	return nil
}

// Send publish a raw string value to the specified channel.
func (c *Client) Send(ctx context.Context, channel string, message string) error {
	err := c.vkclient.Do(ctx, c.vkclient.B().Publish().Channel(channel).Message(message).Build()).Error()
	if err != nil {
		return fmt.Errorf("cannot send message to %s channel: %w", channel, err)
	}

	return nil
}

// Receive receives a raw string message from a subscribed channel.
// Returns the channel name and the message value.
func (c *Client) Receive(ctx context.Context) (string, string, error) {
	data := VKMessage{}

	err := c.vkclient.Receive(ctx, c.vkpubsub, func(msg VKMessage) {
		data = msg
	})
	if err != nil {
		return "", "", fmt.Errorf("error receiving message: %w", err)
	}

	return data.Channel, data.Message, nil
}

// MessageEncode encodes and serialize the input data to a string.
func MessageEncode(data any) (string, error) {
	return encode.Encode(data) //nolint:wrapcheck
}

// MessageDecode decodes a message encoded with MessageEncode to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func MessageDecode(msg string, data any) error {
	return encode.Decode(msg, data) //nolint:wrapcheck
}

// DefaultMessageEncodeFunc is the default function to encode and serialize the input data for SendData().
func DefaultMessageEncodeFunc(_ context.Context, data any) (string, error) {
	return MessageEncode(data)
}

// DefaultMessageDecodeFunc is the default function to decode a message for ReceiveData().
// The value underlying data must be a pointer to the correct type for the next data item received.
func DefaultMessageDecodeFunc(_ context.Context, msg string, data any) error {
	return MessageDecode(msg, data)
}

// SetData sets an encoded value for the specified key with an expiration time.
// Zero expiration means the key has no expiration time.
func (c *Client) SetData(ctx context.Context, key string, data any, exp time.Duration) error {
	value, err := c.messageEncodeFunc(ctx, data)
	if err != nil {
		return err
	}

	return c.Set(ctx, key, value, exp)
}

// GetData retrieves an encoded value of the specified key and extract its content in the data parameter.
func (c *Client) GetData(ctx context.Context, key string, data any) error {
	value, err := c.Get(ctx, key)
	if err != nil {
		return err
	}

	return c.messageDecodeFunc(ctx, value, data)
}

// SendData publish an encoded value to the specified channel.
func (c *Client) SendData(ctx context.Context, channel string, data any) error {
	message, err := c.messageEncodeFunc(ctx, data)
	if err != nil {
		return err
	}

	return c.Send(ctx, channel, message)
}

// ReceiveData receives an encoded message from a subscribed channel,
// and extract its content in the data parameter.
// Returns the channel name in case of success.
func (c *Client) ReceiveData(ctx context.Context, data any) (string, error) {
	channel, value, err := c.Receive(ctx)
	if err != nil {
		return "", err
	}

	return channel, c.messageDecodeFunc(ctx, value, data)
}

// HealthCheck checks if the current data-store is alive.
func (c *Client) HealthCheck(ctx context.Context) error {
	err := c.vkclient.Do(ctx, c.vkclient.B().Ping().Build()).Error()
	if err != nil {
		return fmt.Errorf("unable to connect to Valkey: %w", err)
	}

	return nil
}
