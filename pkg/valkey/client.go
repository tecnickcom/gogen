package valkey

import (
	"context"
	"errors"
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
	// It is only set when at least one channel is configured via WithChannels.
	vkpubsub VKPubSub

	// subch delivers Pub/Sub messages from the background subscription
	// goroutine to Receive(). It is nil when no channels are configured and
	// it is closed when the subscription terminates.
	subch chan VKMessage

	// subcancel stops the background subscription goroutine.
	subcancel context.CancelFunc

	// suberr is the terminal error returned by the background subscription.
	// It is written once before subch is closed and must only be read after
	// observing subch closed.
	suberr error

	// messageEncodeFunc is the function used by SendData()
	// to encode and serialize the input data to a string compatible with Valkey.
	messageEncodeFunc TEncodeFunc

	// messageDecodeFunc is the function used by ReceiveData()
	// to decode a message encoded with messageEncodeFunc to the provided data object.
	// The value underlying data must be a pointer to the correct type for the next data item received.
	messageDecodeFunc TDecodeFunc
}

// New constructs a Valkey client wrapper with optional pinned Pub/Sub subscription and pluggable codecs.
//
// A Pub/Sub subscription is established only when at least one channel is
// configured via WithChannels: a single background goroutine then receives the
// published messages and hands them over, one per call, to Receive or
// ReceiveData. The subscription is bound to ctx and is terminated by Close.
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

	c := &Client{
		vkclient:          vkc,
		messageEncodeFunc: cfg.messageEncodeFunc,
		messageDecodeFunc: cfg.messageDecodeFunc,
	}

	if len(cfg.channels) > 0 {
		c.vkpubsub = vkc.B().Subscribe().Channel(cfg.channels...).Build().Pin()
		c.subscribe(ctx)
	}

	return c, nil
}

// Close stops the Pub/Sub subscription (when configured) and closes the
// underlying client after pending calls complete.
func (c *Client) Close() {
	if c.subcancel != nil {
		c.subcancel()
	}

	c.vkclient.Close()
}

// Set stores a raw string value for key with expiration.
//
// A non-positive exp stores the key without expiration (no TTL).
// Whole-second durations are sent as EX (seconds), while other durations use
// PX (milliseconds) for sub-second precision, with a minimum effective TTL of
// one millisecond.
func (c *Client) Set(ctx context.Context, key string, value string, exp time.Duration) error {
	base := c.vkclient.B().Set().Key(key).Value(value)

	var cmd libvalkey.Completed

	switch {
	case exp <= 0:
		cmd = base.Build()
	case exp < time.Millisecond:
		cmd = base.Px(time.Millisecond).Build()
	case exp%time.Second == 0:
		cmd = base.Ex(exp).Build()
	default:
		cmd = base.Px(exp).Build()
	}

	err := c.vkclient.Do(ctx, cmd).Error()
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
//
// Messages are delivered by the background subscription established at
// construction time; each call consumes exactly one message. It returns an
// error when the client was constructed without any subscription channel
// (see WithChannels), when ctx is canceled, or when the subscription has
// terminated.
func (c *Client) Receive(ctx context.Context) (string, string, error) {
	if c.subch == nil {
		return "", "", errors.New("no subscription channel configured")
	}

	select {
	case <-ctx.Done():
		return "", "", fmt.Errorf("context has been canceled: %w", ctx.Err())
	case msg, ok := <-c.subch:
		if ok {
			return msg.Channel, msg.Message, nil
		}
	}

	// The subscription terminated: c.suberr was written before subch closed.
	if c.suberr != nil {
		return "", "", fmt.Errorf("error receiving message: %w", c.suberr)
	}

	return "", "", errors.New("the subscription has been closed")
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

// subscribe starts the background goroutine that runs the blocking valkey-go
// Receive call once, pushing every published message into c.subch.
// The goroutine terminates (closing c.subch) when the subscription ends:
// on Close, on ctx cancellation, or on client-side disconnection.
func (c *Client) subscribe(ctx context.Context) {
	subctx, cancel := context.WithCancel(ctx)

	subch := make(chan VKMessage)

	c.subcancel = cancel
	c.subch = subch

	go func() {
		defer close(subch)

		c.suberr = c.vkclient.Receive(subctx, c.vkpubsub, func(msg VKMessage) {
			select {
			case subch <- msg:
			case <-subctx.Done():
			}
		})
	}()
}
