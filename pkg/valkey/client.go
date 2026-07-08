package valkey

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/tecnickcom/gogen/pkg/encode"
	libvalkey "github.com/valkey-io/valkey-go"
)

// Sentinel errors returned by the package. They can be matched with errors.Is
// so callers can distinguish configuration problems from subscription state.
var (
	// ErrInvalidOptions is returned by New when no client is injected and the
	// server options carry a missing or malformed InitAddress.
	ErrInvalidOptions = errors.New("valkey: missing or invalid client options")

	// ErrNilEncodeFunc is returned by New when the message encode function is nil.
	ErrNilEncodeFunc = errors.New("valkey: nil message encode function")

	// ErrNilDecodeFunc is returned by New when the message decode function is nil.
	ErrNilDecodeFunc = errors.New("valkey: nil message decode function")

	// ErrInvalidChannelName is returned by New when a subscription channel
	// name configured via WithChannels is empty.
	ErrInvalidChannelName = errors.New("valkey: empty subscription channel name")

	// ErrKeyNotFound is returned by Get and GetData when the key does not
	// exist in the datastore. It signals a missing lookup target (e.g. maps
	// to HTTP 404) rather than a connection or protocol failure.
	ErrKeyNotFound = errors.New("valkey: key not found")

	// ErrNoSubscription is returned by Receive and ReceiveData when the client
	// was constructed without any subscription channel (see WithChannels).
	ErrNoSubscription = errors.New("valkey: no subscription channel configured")

	// ErrSubscriptionClosed is returned by Receive and ReceiveData after the
	// background subscription has ended cleanly (e.g. on Close).
	ErrSubscriptionClosed = errors.New("valkey: subscription closed")
)

// TEncodeFunc is the type of function used to replace the default message encoding function used by SendData() and SetData().
type TEncodeFunc func(ctx context.Context, data any) (string, error)

// TDecodeFunc is the type of function used to replace the default message decoding function used by ReceiveData() and GetData().
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

	// closeOnce guards Close so the underlying client is released at most once.
	closeOnce sync.Once

	// closed reports whether Close has been called. It is set before the
	// subscription is torn down so Receive can report a deliberate shutdown as
	// a clean closure regardless of the underlying termination error (the live
	// client races between context.Canceled and ErrClosing on Close).
	closed atomic.Bool
}

// New constructs a Valkey client wrapper with optional pinned Pub/Sub subscription and pluggable codecs.
//
// A Pub/Sub subscription is established only when at least one channel is
// configured via WithChannels: a single background goroutine then receives the
// published messages and hands them over, one per call, to Receive or
// ReceiveData.
//
// ctx does not bound the lifetime of the client or the subscription. It
// contributes only context values to the subscription and is unused entirely
// when no channels are configured. Canceling ctx after New returns does not
// stop the subscription — it runs until Close is called or the server
// disconnects. Consequently, when channels are configured, Close is the only
// way to stop the background goroutine and must be called.
func New(ctx context.Context, srvopt SrvOptions, opts ...Option) (*Client, error) {
	cfg, err := loadConfig(srvopt, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create a new valkey client: %w", err)
	}

	vkc := cfg.vkclient

	if vkc == nil {
		vkc, err = libvalkey.NewClient(cfg.srvOpts)
		if err != nil {
			return nil, fmt.Errorf("cannot create the valkey client: %w", err)
		}
	}

	c := &Client{
		vkclient:          vkc,
		messageEncodeFunc: cfg.messageEncodeFunc,
		messageDecodeFunc: cfg.messageDecodeFunc,
	}

	if len(cfg.channels) > 0 {
		c.subscribe(ctx, vkc.B().Subscribe().Channel(cfg.channels...).Build().Pin())
	}

	return c, nil
}

// Close stops the Pub/Sub subscription (when configured) and closes the
// underlying client after pending calls complete. It is safe to call Close
// multiple times; only the first call releases the underlying client.
//
// When channels are configured, Close must be called to stop the background
// subscription goroutine and release its resources: canceling the context
// passed to New does not.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		// Record the deliberate shutdown before tearing down the subscription
		// so a concurrent Receive observes it once subch closes.
		c.closed.Store(true)

		if c.subcancel != nil {
			c.subcancel()
		}

		c.vkclient.Close()
	})
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
		return fmt.Errorf("cannot set key %s: %w", key, err)
	}

	return nil
}

// Get retrieves the raw string value for key.
//
// When the key does not exist, the returned error satisfies
// errors.Is(err, ErrKeyNotFound).
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	value, err := c.vkclient.Do(ctx, c.vkclient.B().Get().Key(key).Build()).ToString()
	if err != nil {
		if errors.Is(err, libvalkey.Nil) {
			return "", fmt.Errorf("cannot retrieve key %s: %w", key, ErrKeyNotFound)
		}

		return "", fmt.Errorf("cannot retrieve key %s: %w", key, err)
	}

	return value, nil
}

// Del deletes key from the datastore.
func (c *Client) Del(ctx context.Context, key string) error {
	err := c.vkclient.Do(ctx, c.vkclient.B().Del().Key(key).Build()).Error()
	if err != nil {
		return fmt.Errorf("cannot delete key %s: %w", key, err)
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
// construction time; each call consumes exactly one message. It returns
// ErrNoSubscription when the client was constructed without any subscription
// channel (see WithChannels), a wrapped ctx.Err() when ctx is canceled or its
// deadline expires, ErrSubscriptionClosed when the subscription has ended
// cleanly (e.g. on Close), and a wrapped subscription error when it terminated
// due to a failure. Match the sentinel errors with errors.Is.
func (c *Client) Receive(ctx context.Context) (string, string, error) {
	if c.subch == nil {
		return "", "", ErrNoSubscription
	}

	select {
	case <-ctx.Done():
		return "", "", fmt.Errorf("context terminated: %w", ctx.Err())
	case msg, ok := <-c.subch:
		if ok {
			return msg.Channel, msg.Message, nil
		}
	}

	// The subscription terminated: c.suberr was written before subch closed.
	// A deliberate Close is reported as a clean closure regardless of the
	// underlying cause: the live client races between returning context.Canceled
	// and ErrClosing on Close, so genuine failures (no Close) still surface as
	// errors while an intentional shutdown always reads as closed.
	if c.closed.Load() {
		return "", "", ErrSubscriptionClosed
	}

	if c.suberr != nil {
		return "", "", fmt.Errorf("error receiving message: %w", c.suberr)
	}

	return "", "", ErrSubscriptionClosed
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
		return fmt.Errorf("cannot encode data for key %s: %w", key, err)
	}

	return c.Set(ctx, key, value, exp)
}

// GetData retrieves an encoded value from key and decodes it into data.
//
// When the key does not exist, the returned error satisfies
// errors.Is(err, ErrKeyNotFound).
func (c *Client) GetData(ctx context.Context, key string, data any) error {
	value, err := c.Get(ctx, key)
	if err != nil {
		return err
	}

	err = c.messageDecodeFunc(ctx, value, data)
	if err != nil {
		return fmt.Errorf("cannot decode data for key %s: %w", key, err)
	}

	return nil
}

// SendData encodes data and publishes it to channel.
func (c *Client) SendData(ctx context.Context, channel string, data any) error {
	message, err := c.messageEncodeFunc(ctx, data)
	if err != nil {
		return fmt.Errorf("cannot encode data for %s channel: %w", channel, err)
	}

	return c.Send(ctx, channel, message)
}

// ReceiveData receives an encoded message, decodes it into data, and returns the
// source channel. On any error, including a decode failure, the returned channel
// name is empty.
func (c *Client) ReceiveData(ctx context.Context, data any) (string, error) {
	channel, value, err := c.Receive(ctx)
	if err != nil {
		return "", err
	}

	err = c.messageDecodeFunc(ctx, value, data)
	if err != nil {
		return "", fmt.Errorf("cannot decode data from %s channel: %w", channel, err)
	}

	return channel, nil
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
// Receive call once with the pinned sub command, pushing every published
// message into c.subch.
// The goroutine terminates (closing c.subch) when the subscription ends:
// on Close or on client-side disconnection. The subscription context retains
// the values of ctx but is not canceled when ctx is: only Close (via
// subcancel) stops it.
func (c *Client) subscribe(ctx context.Context, sub VKPubSub) {
	subctx, cancel := context.WithCancel(context.WithoutCancel(ctx))

	subch := make(chan VKMessage)

	c.subcancel = cancel
	c.subch = subch

	go func() {
		defer close(subch)

		c.suberr = c.vkclient.Receive(subctx, sub, func(msg VKMessage) {
			select {
			case subch <- msg:
			case <-subctx.Done():
			}
		})
	}()
}
