package redis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	libredis "github.com/redis/go-redis/v9"
	"github.com/tecnickcom/gogen/pkg/encode"
)

// Sentinel errors returned by the package. They can be matched with errors.Is
// so callers can distinguish configuration problems from lookup and
// subscription states.
var (
	// ErrInvalidOptions is returned by New when no client is injected and the
	// server options carry a missing or malformed address.
	ErrInvalidOptions = errors.New("redis: missing or invalid client options")

	// ErrNilEncodeFunc is returned by New when the message encode function is nil.
	ErrNilEncodeFunc = errors.New("redis: nil message encode function")

	// ErrNilDecodeFunc is returned by New when the message decode function is nil.
	ErrNilDecodeFunc = errors.New("redis: nil message decode function")

	// ErrInvalidChannelName is returned by New when a subscription channel
	// name configured via WithChannels is empty.
	ErrInvalidChannelName = errors.New("redis: empty subscription channel name")

	// ErrKeyNotFound is returned by Get and GetData when the key does not
	// exist in the datastore. It signals a missing lookup target (e.g. maps
	// to HTTP 404) rather than a connection or protocol failure.
	ErrKeyNotFound = errors.New("redis: key not found")

	// ErrNoSubscription is returned by Receive and ReceiveData when the client
	// was constructed without any subscription channel (see WithChannels).
	ErrNoSubscription = errors.New("redis: no subscription channel configured")

	// ErrSubscriptionClosed is returned by Receive and ReceiveData after the
	// subscription message channel has been closed (e.g. on Close).
	ErrSubscriptionClosed = errors.New("redis: subscription closed")
)

// TEncodeFunc is the type of function used to replace the default message encoding function used by SendData() and SetData().
type TEncodeFunc func(ctx context.Context, data any) (string, error)

// TDecodeFunc is the type of function used to replace the default message decoding function used by ReceiveData() and GetData().
type TDecodeFunc func(ctx context.Context, msg string, data any) error

// SrvOptions aliases go-redis client options used when constructing a client.
type SrvOptions = libredis.Options

// RMessage aliases a go-redis Pub/Sub message.
type RMessage = libredis.Message

// ChannelOption aliases go-redis Pub/Sub channel options.
type ChannelOption = libredis.ChannelOption

// RClient defines the go-redis client calls used by [Client].
type RClient interface {
	Close() error
	Del(ctx context.Context, keys ...string) *libredis.IntCmd
	Get(ctx context.Context, key string) *libredis.StringCmd

	// Ping is used by HealthCheck.
	Ping(ctx context.Context) *libredis.StatusCmd

	Publish(ctx context.Context, channel string, message any) *libredis.IntCmd
	Set(ctx context.Context, key string, value any, expiration time.Duration) *libredis.StatusCmd
	Subscribe(ctx context.Context, channels ...string) *libredis.PubSub
}

// RPubSub defines the go-redis Pub/Sub calls used by [Client].
type RPubSub interface {
	Channel(opts ...libredis.ChannelOption) <-chan *libredis.Message
	Close() error
}

// Client wraps Redis KV/PubSub operations with optional typed payload codecs.
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

	// closeOnce guards Close so the underlying resources are released at most once.
	closeOnce sync.Once

	// closeErr is the result of the first Close call, returned by every
	// subsequent call.
	closeErr error
}

// New constructs a Redis client wrapper with optional Pub/Sub subscriptions and pluggable message codecs.
//
// A Pub/Sub subscription is established only when at least one channel is
// configured via WithChannels; otherwise no subscription resources are
// allocated.
//
// ctx does not bound the lifetime of the client or the subscription: go-redis
// uses it only for the initial SUBSCRIBE command (whose failure surfaces
// through the subscription retry loop, not here) while the goroutine
// delivering messages to Receive runs until Close is called. Consequently,
// when channels are configured, Close is the only way to stop the
// subscription and must be called.
func New(ctx context.Context, srvopt *SrvOptions, opts ...Option) (*Client, error) {
	cfg, err := loadConfig(srvopt, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create a new redis client: %w", err)
	}

	rc := cfg.rclient

	if rc == nil {
		rc = libredis.NewClient(cfg.srvOpts)
	}

	c := &Client{
		rclient:           rc,
		messageEncodeFunc: cfg.messageEncodeFunc,
		messageDecodeFunc: cfg.messageDecodeFunc,
	}

	if len(cfg.channels) > 0 {
		ps := c.rclient.Subscribe(ctx, cfg.channels...)
		if ps == nil {
			return nil, fmt.Errorf("injected client returned a nil PubSub: %w", ErrInvalidOptions)
		}

		c.rpubsub = ps
		c.subch = ps.Channel(cfg.channelOpts...)
	}

	return c, nil
}

// Close gracefully closes Pub/Sub and Redis client resources.
//
// When no Pub/Sub subscription was configured at construction time, only the
// Redis client is closed. The Redis client is always closed even when closing
// the Pub/Sub subscription fails; in that case the errors are joined.
//
// Close is idempotent: only the first call releases the resources, and every
// subsequent call returns the result of the first. When channels are
// configured, Close must be called to stop the background subscription:
// canceling the context passed to New does not.
func (c *Client) Close() error {
	c.closeOnce.Do(func() {
		var errPubSub error

		if c.rpubsub != nil {
			err := c.rpubsub.Close()
			if err != nil {
				errPubSub = fmt.Errorf("failed to close Redis PubSub: %w", err)
			}
		}

		errClient := c.rclient.Close()
		if errClient != nil {
			errClient = fmt.Errorf("failed to close Redis Client: %w", errClient)
		}

		c.closeErr = errors.Join(errPubSub, errClient)
	})

	return c.closeErr
}

// Set stores a raw value for key with expiration.
//
// value must be a type supported by the go-redis protocol writer: a string,
// []byte, a numeric or bool type (or a pointer to one), time.Time,
// time.Duration, or a type implementing encoding.BinaryMarshaler; anything
// else fails at command time. Use SetData for arbitrary values.
//
// A zero or negative exp (other than libredis.KeepTTL, which retains the
// key's existing TTL) stores the key without expiration. Positive durations
// below one millisecond are truncated to 1ms by go-redis.
func (c *Client) Set(ctx context.Context, key string, value any, exp time.Duration) error {
	err := c.rclient.Set(ctx, key, value, exp).Err()
	if err != nil {
		return fmt.Errorf("cannot set key %s: %w", key, err)
	}

	return nil
}

// Get retrieves the raw value of key and scans it into value.
//
// value must be a pointer to a type supported by go-redis scanning: a string,
// a numeric or bool type, time.Time, time.Duration, []byte, or a type
// implementing encoding.BinaryUnmarshaler. Use GetData for values stored with
// SetData.
//
// When the key does not exist, the returned error satisfies
// errors.Is(err, ErrKeyNotFound).
func (c *Client) Get(ctx context.Context, key string, value any) error {
	err := c.rclient.Get(ctx, key).Scan(value)
	if err != nil {
		if errors.Is(err, libredis.Nil) {
			return fmt.Errorf("cannot retrieve key %s: %w", key, ErrKeyNotFound)
		}

		return fmt.Errorf("cannot retrieve key %s: %w", key, err)
	}

	return nil
}

// Del deletes key from the datastore.
func (c *Client) Del(ctx context.Context, key string) error {
	err := c.rclient.Del(ctx, key).Err()
	if err != nil {
		return fmt.Errorf("cannot delete key %s: %w", key, err)
	}

	return nil
}

// Send publishes a raw value to channel.
//
// message must be a type supported by the go-redis protocol writer (see Set);
// use SendData for arbitrary values.
func (c *Client) Send(ctx context.Context, channel string, message any) error {
	err := c.rclient.Publish(ctx, channel, message).Err()
	if err != nil {
		return fmt.Errorf("cannot send message to %s channel: %w", channel, err)
	}

	return nil
}

// Receive returns the next raw message from subscribed channels as channel name and payload.
//
// Each call consumes exactly one message delivered by the subscription
// established at construction time. It returns ErrNoSubscription when the
// client was constructed without any subscription channel (see WithChannels),
// a wrapped ctx.Err() when ctx is canceled or its deadline expires, and
// ErrSubscriptionClosed after Close. Match the sentinel errors with errors.Is.
// After Close, any messages already buffered are still delivered before
// ErrSubscriptionClosed is returned.
//
// Messages are buffered in a Go channel holding 100 messages by default; when
// the buffer stays full for one minute (the go-redis default send timeout),
// the incoming message is silently dropped and only logged by go-redis. Slow
// consumers should tune this via WithChannelOptions (libredis.WithChannelSize,
// libredis.WithChannelSendTimeout).
func (c *Client) Receive(ctx context.Context) (string, string, error) {
	if c.subch == nil {
		return "", "", ErrNoSubscription
	}

	for {
		select {
		case <-ctx.Done():
			return "", "", fmt.Errorf("context terminated: %w", ctx.Err())
		case msg, ok := <-c.subch:
			if !ok {
				return "", "", ErrSubscriptionClosed
			}

			// go-redis never delivers nil messages; skip them defensively
			// instead of dereferencing or misreporting a closed channel.
			if msg == nil {
				continue
			}

			return msg.Channel, msg.Payload, nil
		}
	}
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
//
// A zero or negative exp (other than libredis.KeepTTL, which retains the
// key's existing TTL) stores the key without expiration. Positive durations
// below one millisecond are truncated to 1ms by go-redis.
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
	var value string

	err := c.Get(ctx, key, &value)
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

// HealthCheck verifies Redis connectivity with a PING command.
func (c *Client) HealthCheck(ctx context.Context) error {
	err := c.rclient.Ping(ctx).Err()
	if err != nil {
		return fmt.Errorf("unable to connect to Redis: %w", err)
	}

	return nil
}
