package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Vonage/gosrvlib/pkg/encode"
	libredis "github.com/redis/go-redis/v9"
)

// TEncodeFunc is the type of function used to replace the default message encoding function used by SendData().
type TEncodeFunc func(ctx context.Context, data any) (string, error)

// TDecodeFunc is the type of function used to replace the default message decoding function used by ReceiveData().
type TDecodeFunc func(ctx context.Context, msg string, data any) error

// SrvOptions is an alias for the parent library client options.
type SrvOptions = libredis.Options

// RMessage is an alias for the parent library Message type.
type RMessage = libredis.Message

// ChannelOption is an alias for the parent library ChannelOption.
type ChannelOption = libredis.ChannelOption

// RClient represents the mockable functions in the parent Redis Client.
type RClient interface {
	Close() error
	Del(ctx context.Context, keys ...string) *libredis.IntCmd
	Get(ctx context.Context, key string) *libredis.StringCmd
	Ping(ctx context.Context) *libredis.StatusCmd // this function is used by the HealthCheck
	Publish(ctx context.Context, channel string, message any) *libredis.IntCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *libredis.StatusCmd
	Subscribe(ctx context.Context, channels ...string) *libredis.PubSub
}

// RPubSub represents the mockable functions in the parent Redis PubSub.
type RPubSub interface {
	Channel(opts ...libredis.ChannelOption) <-chan *libredis.Message
	Close() error
}

// Client is a wrapper for the Redis Client.
type Client struct {
	// rclient is the upstream Client.
	rclient RClient

	// rpubsub is the upstream PubSub.
	rpubsub RPubSub

	// subch is a Go channel for concurrently receiving messages from the subscribed channels.
	subch <-chan *RMessage

	// messageEncodeFunc is the function used by SendData()
	// to encode and serialize the input data to a string compatible with Redis.
	messageEncodeFunc TEncodeFunc

	// messageDecodeFunc is the function used by ReceiveData()
	// to decode a message encoded with messageEncodeFunc to the provided data object.
	// The value underlying data must be a pointer to the correct type for the next data item received.
	messageDecodeFunc TDecodeFunc
}

// New creates a new instance of the Redis client wrapper.
func New(ctx context.Context, srvopt *SrvOptions, opts ...Option) (*Client, error) {
	cfg, err := loadConfig(ctx, srvopt, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create a new redis client: %w", err)
	}

	rclient := libredis.NewClient(cfg.srvOpts)
	rpubsub := rclient.Subscribe(ctx, cfg.subChannels...)
	subch := rpubsub.Channel(cfg.subChannelOpts...)

	return &Client{
		rclient:           rclient,
		rpubsub:           rpubsub,
		subch:             subch,
		messageEncodeFunc: cfg.messageEncodeFunc,
		messageDecodeFunc: cfg.messageDecodeFunc,
	}, nil
}

// Close closes the parent client, releasing any open resources.
func (c *Client) Close() error {
	err := c.rpubsub.Close()
	if err != nil {
		return fmt.Errorf("failed to close Redis PubSub: %w", err)
	}

	err = c.rclient.Close()
	if err != nil {
		return fmt.Errorf("failed to close Redis Client: %w", err)
	}

	return nil
}

// Set a raw value for the specified key with an expiration time.
// Zero expiration means the key has no expiration time.
func (c *Client) Set(ctx context.Context, key string, value any, exp time.Duration) error {
	err := c.rclient.Set(ctx, key, value, exp).Err()
	if err != nil {
		return fmt.Errorf("cannot set key %s: %w", key, err)
	}

	return nil
}

// Get retrieves the raw value of the specified key and extract its content in the value parameter.
func (c *Client) Get(ctx context.Context, key string, value any) error {
	err := c.rclient.Get(ctx, key).Scan(value)
	if err != nil {
		return fmt.Errorf("cannot retrieve key %s: %w", key, err)
	}

	return nil
}

// Del deletes the specified key from the datastore.
func (c *Client) Del(ctx context.Context, key string) error {
	err := c.rclient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("cannot delete key: %s %w", key, err)
	}

	return nil
}

// Send publish a raw value to the specified channel.
func (c *Client) Send(ctx context.Context, channel string, message any) error {
	err := c.rclient.Publish(ctx, channel, message).Err()
	if err != nil {
		return fmt.Errorf("cannot send message to %s channel: %w", channel, err)
	}

	return nil
}

// Receive receives a raw string message from a subscribed channel.
// Returns the channel name and the message value.
func (c *Client) Receive(ctx context.Context) (string, string, error) {
	select {
	case <-ctx.Done():
		return "", "", fmt.Errorf("context has been canceled: %w", ctx.Err())
	case msg, ok := <-c.subch:
		if ok && (msg != nil) {
			return msg.Channel, msg.Payload, nil
		}
	}

	return "", "", errors.New("the receiving channel is closed")
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
	var value string

	err := c.Get(ctx, key, &value)
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
	err := c.rclient.Ping(ctx).Err()
	if err != nil {
		return fmt.Errorf("unable to connect to Redis: %w", err)
	}

	return nil
}
