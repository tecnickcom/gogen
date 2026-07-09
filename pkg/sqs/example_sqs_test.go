package sqs_test

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/tecnickcom/gogen/pkg/sqs"
)

// exampleSQSClient is a minimal in-memory stand-in for the AWS SQS client,
// implementing only the sqs.SQS interface. Real code would let New build the
// client from aws.Config instead.
type exampleSQSClient struct {
	body string
}

func (c *exampleSQSClient) SendMessage(
	_ context.Context,
	params *awssqs.SendMessageInput,
	_ ...func(*awssqs.Options),
) (*awssqs.SendMessageOutput, error) {
	c.body = aws.ToString(params.MessageBody)

	return &awssqs.SendMessageOutput{}, nil
}

func (c *exampleSQSClient) ReceiveMessage(
	_ context.Context,
	_ *awssqs.ReceiveMessageInput,
	_ ...func(*awssqs.Options),
) (*awssqs.ReceiveMessageOutput, error) {
	return &awssqs.ReceiveMessageOutput{
		Messages: []types.Message{{
			Body:          aws.String(c.body),
			ReceiptHandle: aws.String("example-receipt-handle"),
		}},
	}, nil
}

func (c *exampleSQSClient) DeleteMessage(
	_ context.Context,
	_ *awssqs.DeleteMessageInput,
	_ ...func(*awssqs.Options),
) (*awssqs.DeleteMessageOutput, error) {
	return &awssqs.DeleteMessageOutput{}, nil
}

func (c *exampleSQSClient) GetQueueAttributes(
	_ context.Context,
	_ *awssqs.GetQueueAttributesInput,
	_ ...func(*awssqs.Options),
) (*awssqs.GetQueueAttributesOutput, error) {
	return &awssqs.GetQueueAttributesOutput{}, nil
}

func Example() {
	// A caller would normally configure a real client via options such as
	// WithAWSOptions or WithEndpointMutable (and the region would be derived from
	// the queue URL); here an injected client keeps the example self-contained, so
	// no AWS configuration is loaded.
	c, err := sqs.New(
		context.TODO(),
		"https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
		"", // standard (non-FIFO) queue: no message group ID
		sqs.WithSQSClient(&exampleSQSClient{}),
	)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	err = c.Send(context.TODO(), "hello world")
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	msg, err := c.Receive(context.TODO())
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	fmt.Println(msg.Body)

	// After processing, acknowledge the message by deleting it.
	err = c.Delete(context.TODO(), msg.ReceiptHandle)
	if err != nil {
		fmt.Println("error:", err)

		return
	}

	// Output:
	// hello world
}
