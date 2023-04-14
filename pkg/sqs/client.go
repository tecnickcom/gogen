package sqs

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/Vonage/gosrvlib/pkg/typeutil"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

const (
	fifoSuffix = ".fifo"

	regexMessageGroupID = `^[[:graph:]]{1,128}$`
)

// SQS represents the mockable functions in the AWS SDK SQS client.
type SQS interface {
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
	GetQueueAttributes(ctx context.Context, params *sqs.GetQueueAttributesInput, optFns ...func(*sqs.Options)) (*sqs.GetQueueAttributesOutput, error)
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

// Client is a wrapper for the SQS client in the AWS SDK.
type Client struct {
	// sqs is the interface for the upstream functions.
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

	// hcGetQueueAttributesInput is the input parameter for the GetQueueAttributes function used by the HealthCheck.
	hcGetQueueAttributesInput *sqs.GetQueueAttributesInput
}

// New creates a new instance of the SQS client wrapper.
// msgGroupID is required for FIFO queues.
func New(ctx context.Context, queueURL, msgGroupID string, opts ...Option) (*Client, error) {
	cfg, err := loadConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create a new sqs client: %w", err)
	}

	var awsMsgGroupID *string

	if strings.HasSuffix(queueURL, fifoSuffix) {
		re := regexp.MustCompile(regexMessageGroupID)
		if !re.MatchString(msgGroupID) {
			return nil, fmt.Errorf("a valid msgGroupID is required for FIFO queue")
		}

		awsMsgGroupID = aws.String(msgGroupID)
	}

	return &Client{
		sqs:               sqs.NewFromConfig(cfg.awsConfig),
		queueURL:          aws.String(queueURL),
		messageGroupID:    awsMsgGroupID,
		waitTimeSeconds:   cfg.waitTimeSeconds,
		visibilityTimeout: cfg.visibilityTimeout,
		hcGetQueueAttributesInput: &sqs.GetQueueAttributesInput{
			QueueUrl:       aws.String(queueURL),
			AttributeNames: []types.QueueAttributeName{types.QueueAttributeNameLastModifiedTimestamp},
		},
	}, nil
}

// Message represents a message in the queue.
type Message struct {
	// can contain: JSON, XML, plain text.
	Body string

	// ReceiptHandle is the identifier used to delete the message.
	ReceiptHandle string
}

// Send delivers a raw string message to the queue.
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

// Receive retrieves a raw string message from the queue.
// This function will wait up to WaitTimeSeconds seconds for a message to be available, otherwise it will return nil.
// Once retrieved, a message will not be visible for up to VisibilityTimeout seconds.
// Once processed the message should be removed from the queue by calling the Delete method.
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
		return nil, nil
	}

	return &Message{
		Body:          aws.ToString(resp.Messages[0].Body),
		ReceiptHandle: aws.ToString(resp.Messages[0].ReceiptHandle),
	}, nil
}

// Delete deletes the specified message from the queue.
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

// MessageEncode encodes and serialize the input data to a string compatible with SQS.
func MessageEncode(data interface{}) (string, error) {
	return typeutil.Encode(data) //nolint:wrapcheck
}

// MessageDecode decodes a message encoded with MessageEncode to the provided data object.
// The value underlying data must be a pointer to the correct type for the next data item received.
func MessageDecode(msg string, data interface{}) error {
	return typeutil.Decode(msg, data) //nolint:wrapcheck
}

// SendData delivers the specified data as message to the queue.
func (c *Client) SendData(ctx context.Context, data interface{}) error {
	message, err := MessageEncode(data)
	if err != nil {
		return err
	}

	return c.Send(ctx, message)
}

// ReceiveData retrieves a message from the queue, extract its content in the data and returns the ReceiptHandle.
// The value underlying data must be a pointer to the correct type for the next data item received.
// This function will wait up to WaitTimeSeconds seconds for a message to be available, otherwise it will return an empty ReceiptHandle.
// Once retrieved, a message will not be visible for up to VisibilityTimeout seconds.
// Once processed the message should be removed from the queue by calling the Delete method.
// In case of decoding error the returned receipt handle will be not empty, so it can be used to delete the message.
func (c *Client) ReceiveData(ctx context.Context, data interface{}) (string, error) {
	message, err := c.Receive(ctx)
	if err != nil {
		return "", err
	}

	if message == nil {
		return "", nil
	}

	err = MessageDecode(message.Body, data)

	return message.ReceiptHandle, err
}

// HealthCheck checks if the current queue is present in the current region and returns an error otherwise.
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
