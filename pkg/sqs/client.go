package sqs

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/tecnickcom/gogen/pkg/encode"
)

// Internal constants used by SQS queue/metadata validation.
const (
	fifoSuffix          = ".fifo"
	regexMessageGroupID = `^[[:graph:]]{1,128}$`
)

// TEncodeFunc is the type of function used to replace the default message encoding function used by SendData().
type TEncodeFunc func(ctx context.Context, data any) (string, error)

// TDecodeFunc is the type of function used to replace the default message decoding function used by ReceiveData().
type TDecodeFunc func(ctx context.Context, msg string, data any) error

// SQS defines the AWS SDK SQS calls used by [Client].
type SQS interface {
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

// Client wraps AWS SQS operations and typed message encoding/decoding for a single queue URL.
type Client struct {
	// sqs provides the AWS SDK operations used by this client.
	sqs SQS

	// queueURL is the SQS queue URL. Names are case-sensitive and limited up to 80 chars.
	queueURL *string

	// messageGroupID is a tag that specifies that a message belongs to a specific message group.
	// This must be specified for FIFO queues and must be left nil for standard queues.
	messageGroupID *string

	// waitTimeSeconds is the duration (in seconds) for which the call waits for a message to arrive in the queue before returning.
	// If a message is available, the call returns sooner than WaitTimeSeconds.
	// If no messages are available and the wait time expires, the call returns successfully with an empty list of messages.
	// The value of this parameter must be smaller than the HTTP response timeout.
	waitTimeSeconds int32

	// visibilityTimeout is the duration (in seconds) that the received messages are hidden from subsequent retrieve requests after being retrieved by a ReceiveMessage request.
	// Values range: 0 to 43200. Maximum: 12 hours.
	visibilityTimeout int32

	// messageEncodeFunc is the function used by SendData()
	// to encode and serialize the input data to a string compatible with SQS.
	messageEncodeFunc TEncodeFunc

	// messageDecodeFunc is the function used by ReceiveData()
	// to decode a message encoded with messageEncodeFunc to the provided data object.
	// The value underlying data must be a pointer to the correct type for the next data item received.
	messageDecodeFunc TDecodeFunc

	// hcGetQueueAttributesInput is the input parameter for the GetQueueAttributes function used by the HealthCheck.
	hcGetQueueAttributesInput *sqs.GetQueueAttributesInput
}

// New builds a client for queueURL and validates FIFO message-group constraints.
// msgGroupID is required only when queueURL targets a FIFO queue.
func New(ctx context.Context, queueURL, msgGroupID string, opts ...Option) (*Client, error) {
	cfg, err := loadConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create a new sqs client: %w", err)
	}

	var awsMsgGroupID *string

	if strings.HasSuffix(queueURL, fifoSuffix) {
		re := regexp.MustCompile(regexMessageGroupID)
		if !re.MatchString(msgGroupID) {
			return nil, errors.New("a valid msgGroupID is required for FIFO queue")
		}

		awsMsgGroupID = aws.String(msgGroupID)
	}

	return &Client{
		sqs:               sqs.NewFromConfig(cfg.awsConfig, cfg.srvOptFns...),
		queueURL:          aws.String(queueURL),
		messageGroupID:    awsMsgGroupID,
		waitTimeSeconds:   cfg.waitTimeSeconds,
		visibilityTimeout: cfg.visibilityTimeout,
		messageEncodeFunc: cfg.messageEncodeFunc,
		messageDecodeFunc: cfg.messageDecodeFunc,
		hcGetQueueAttributesInput: &sqs.GetQueueAttributesInput{
			QueueUrl:       aws.String(queueURL),
			AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameLastModifiedTimestamp},
		},
	}, nil
}

// Message holds a received payload and the receipt handle used for deletion.
type Message struct {
	// Body is the message content and can contain: JSON, XML or plain text.
	Body string

	// ReceiptHandle is the identifier used to delete the message.
	ReceiptHandle string
}

// Send publishes a raw string message to the queue.
func (c *Client) Send(ctx context.Context, message string) error {
	_, err := c.sqs.SendMessage(
		ctx,
		&sqs.SendMessageInput{
			QueueUrl:       c.queueURL,
			MessageGroupId: c.messageGroupID,
			MessageBody:    aws.String(message),
		})
	if err != nil {
		return fmt.Errorf("cannot send message to the queue: %w", err)
	}

	return nil
}

// Receive retrieves one raw message from the queue with configured wait/visibility settings.
// Returns nil message when no message is available within waitTimeSeconds.
func (c *Client) Receive(ctx context.Context) (*Message, error) {
	resp, err := c.sqs.ReceiveMessage(
		ctx,
		&sqs.ReceiveMessageInput{
			QueueUrl:          c.queueURL,
			WaitTimeSeconds:   c.waitTimeSeconds,
			VisibilityTimeout: c.visibilityTimeout,
		})
	if err != nil {
		return nil, fmt.Errorf("cannot retrieve message from the queue: %w", err)
	}

	if len(resp.Messages) < 1 {
		return nil, nil //nolint:nilnil
	}

	return &Message{
		Body:          aws.ToString(resp.Messages[0].Body),
		ReceiptHandle: aws.ToString(resp.Messages[0].ReceiptHandle),
	}, nil
}

// Delete removes a message from the queue by receipt handle.
func (c *Client) Delete(ctx context.Context, receiptHandle string) error {
	if receiptHandle == "" {
		return nil
	}

	_, err := c.sqs.DeleteMessage(
		ctx,
		&sqs.DeleteMessageInput{
			QueueUrl:      c.queueURL,
			ReceiptHandle: aws.String(receiptHandle),
		})
	if err != nil {
		return fmt.Errorf("cannot delete message from the queue: %w", err)
	}

	return nil
}

// MessageEncode encodes and serializes data into an SQS-compatible string payload.
func MessageEncode(data any) (string, error) {
	return encode.Encode(data) //nolint:wrapcheck
}

// MessageDecode decodes a MessageEncode payload into data, which must be a pointer.
func MessageDecode(msg string, data any) error {
	return encode.Decode(msg, data) //nolint:wrapcheck
}

// DefaultMessageEncodeFunc is the default serializer used by SendData.
func DefaultMessageEncodeFunc(_ context.Context, data any) (string, error) {
	return MessageEncode(data)
}

// DefaultMessageDecodeFunc is the default deserializer used by ReceiveData.
func DefaultMessageDecodeFunc(_ context.Context, msg string, data any) error {
	return MessageDecode(msg, data)
}

// SendData encodes data via configured codec and publishes it to the queue.
func (c *Client) SendData(ctx context.Context, data any) error {
	message, err := c.messageEncodeFunc(ctx, data)
	if err != nil {
		return err
	}

	return c.Send(ctx, message)
}

// ReceiveData receives one message, decodes its payload into data, and returns the receipt handle.
// If decoding fails, receipt handle is still returned so callers can decide whether to delete or requeue.
func (c *Client) ReceiveData(ctx context.Context, data any) (string, error) {
	message, err := c.Receive(ctx)
	if err != nil {
		return "", err
	}

	if message == nil {
		return "", nil
	}

	err = c.messageDecodeFunc(ctx, message.Body, data)

	return message.ReceiptHandle, err
}

// HealthCheck verifies queue reachability by fetching a known queue attribute.
func (c *Client) HealthCheck(ctx context.Context) error {
	q, err := c.sqs.GetQueueAttributes(ctx, c.hcGetQueueAttributesInput)
	if err != nil {
		return fmt.Errorf("unable to connect to AWS SQS: %w", err)
	}

	if _, ok := q.Attributes[string(types.QueueAttributeNameLastModifiedTimestamp)]; ok {
		return nil
	}

	return fmt.Errorf("the AWS SQS queue is not responding: %s", aws.ToString(c.queueURL))
}
