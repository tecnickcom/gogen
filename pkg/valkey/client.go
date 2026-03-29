package valkey

import (
	"context"
	"fmt"
	"time"

	"github.com/tecnickcom/gogen/pkg/encode"
	libvalkey "github.com/valkey-io/valkey-go"
)

// TEncodeFunc is the type of function used to replace the default message encoding function used by SendData().
type TEncodeFunc func(ctx context.Context, data any) (string, error)

// TDecodeFunc is the type of function used to replace the default message decoding function used by ReceiveData().
type TDecodeFunc func(ctx context.Context, msg string, data any) error

// SrvOptions aliases valkey-go client options used when constructing a client.
type SrvOptions = libvalkey.ClientOption

// VKMessage aliases a valkey-go Pub/Sub message.
type VKMessage = libvalkey.PubSubMessage

// VKClient aliases the valkey-go client interface used by [Client].
type VKClient = libvalkey.Client

// VKPubSub aliases the valkey-go completed Pub/Sub command type used by [Client].
type VKPubSub = libvalkey.Completed

// Client wraps Valkey KV/PubSub operations with optional typed payload codecs.
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

// New constructs a Valkey client wrapper with pinned Pub/Sub subscription and pluggable codecs.
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

// Close closes the underlying client after pending calls complete.
func (c *Client) Close() {
	c.vkclient.Close()
}

// Set stores a raw string value for key with expiration.
func (c *Client) Set(ctx context.Context, key string, value string, exp time.Duration) error {
	err := c.vkclient.Do(ctx, c.vkclient.B().Set().Key(key).Value(value).Ex(exp).Build()).Error()
	if err != nil {
		return fmt.Errorf("cannot set key: %s %w", key, err)
	}

	return nil
}

// Get retrieves the raw string value for key.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.vkclient.Do(ctx, c.vkclient.B().Get().Key(key).Build()).ToString()
	if err != nil {
		return "", fmt.Errorf("cannot retrieve key %s: %w", key, err)
	}

	return value, nil
}

// Del deletes key from the datastore.
func (c *Client) Del(ctx context.Context, key string) error {
	err := c.vkclient.Do(ctx, c.vkclient.B().Del().Key(key).Build()).Error()
	if err != nil {
		return fmt.Errorf("cannot delete key: %s %w", key, err)
	}

	return nil
}

// Send publishes a raw string message to channel.
func (c *Client) Send(ctx context.Context, channel string, message string) error {
	err := c.vkclient.Do(ctx, c.vkclient.B().Publish().Channel(channel).Message(message).Build()).Error()
	if err != nil {
		return fmt.Errorf("cannot send message to %s channel: %w", channel, err)
	}

	return nil
}

// Receive returns the next raw Pub/Sub message as channel name and payload.
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

// MessageEncode encodes and serializes data into a string payload.
func MessageEncode(data any) (string, error) {
	return encode.Encode(data) //nolint:wrapcheck
}

// MessageDecode decodes a MessageEncode payload into data, which must be a pointer.
func MessageDecode(msg string, data any) error {
	return encode.Decode(msg, data) //nolint:wrapcheck
}

// DefaultMessageEncodeFunc provides default encoding used by SendData.
func DefaultMessageEncodeFunc(_ context.Context, data any) (string, error) {
	return MessageEncode(data)
}

// DefaultMessageDecodeFunc provides default decoding used by ReceiveData.
func DefaultMessageDecodeFunc(_ context.Context, msg string, data any) error {
	return MessageDecode(msg, data)
}

// SetData encodes data and stores it at key with expiration.
func (c *Client) SetData(ctx context.Context, key string, data any, exp time.Duration) error {
	value, err := c.messageEncodeFunc(ctx, data)
	if err != nil {
		return err
	}

	return c.Set(ctx, key, value, exp)
}

// GetData retrieves an encoded value from key and decodes it into data.
func (c *Client) GetData(ctx context.Context, key string, data any) error {
	value, err := c.Get(ctx, key)
	if err != nil {
		return err
	}

	return c.messageDecodeFunc(ctx, value, data)
}

// SendData encodes data and publishes it to channel.
func (c *Client) SendData(ctx context.Context, channel string, data any) error {
	message, err := c.messageEncodeFunc(ctx, data)
	if err != nil {
		return err
	}

	return c.Send(ctx, channel, message)
}

// ReceiveData receives an encoded message, decodes it into data, and returns the source channel.
func (c *Client) ReceiveData(ctx context.Context, data any) (string, error) {
	channel, value, err := c.Receive(ctx)
	if err != nil {
		return "", err
	}

	return channel, c.messageDecodeFunc(ctx, value, data)
}

// HealthCheck verifies Valkey connectivity with a PING command.
func (c *Client) HealthCheck(ctx context.Context) error {
	err := c.vkclient.Do(ctx, c.vkclient.B().Ping().Build()).Error()
	if err != nil {
		return fmt.Errorf("unable to connect to Valkey: %w", err)
	}

	return nil
}
