package sqs

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/require"
)

// nolint: paralleltest
func TestNew(t *testing.T) {
	var (
		wt int32 = 13
		vt int32 = 17
	)

	got, err := New(
		context.TODO(),
		"test_queue_url_0",
		"TEST_MSG_GROUP_ID_0",
		WithEndpoint("test", true),
		WithWaitTimeSeconds(wt),
		WithVisibilityTimeout(vt),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, aws.String("test_queue_url_0"), got.queueURL)
	require.Equal(t, aws.String("TEST_MSG_GROUP_ID_0"), got.messageGroupID)
	require.Equal(t, wt, got.waitTimeSeconds)
	require.Equal(t, vt, got.visibilityTimeout)

	// make AWS lib to return an error
	t.Setenv("AWS_ENABLE_ENDPOINT_DISCOVERY", "ERROR")

	got, err = New(context.TODO(), "", "")
	require.Error(t, err)
	require.Nil(t, got)
}

type sqsmock struct {
	sendFn    func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
	receiveFn func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	deleteFn  func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

func (s sqsmock) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	return s.sendFn(ctx, params, optFns...)
}

func (s sqsmock) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	return s.receiveFn(ctx, params, optFns...)
}

func (s sqsmock) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	return s.deleteFn(ctx, params, optFns...)
}

func TestSend(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mock    SQS
		wantErr bool
	}{
		{
			name: "success",
			mock: sqsmock{sendFn: func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
				return &sqs.SendMessageOutput{}, nil
			}},
			wantErr: false,
		},
		{
			name: "error",
			mock: sqsmock{sendFn: func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
				return nil, fmt.Errorf("some err")
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.TODO()
			cli, err := New(ctx, "test_queue_url_1", "TEST_MSG_GROUP_ID_1")
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.sqs = tt.mock

			err = cli.Send(ctx, "test")
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestReceive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mock    SQS
		want    *Message
		wantErr bool
	}{
		{
			name: "success",
			mock: sqsmock{receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				return &sqs.ReceiveMessageOutput{
					Messages: []types.Message{
						{
							Body:          aws.String("testBody01"),
							ReceiptHandle: aws.String("TestReceiptHandle01"),
						},
					},
				}, nil
			}},
			want: &Message{
				Body:          "testBody01",
				receiptHandle: "TestReceiptHandle01",
			},
			wantErr: false,
		},
		{
			name: "empty",
			mock: sqsmock{receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				return &sqs.ReceiveMessageOutput{}, nil
			}},
			want:    &Message{},
			wantErr: false,
		},
		{
			name: "error",
			mock: sqsmock{receiveFn: func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				return nil, fmt.Errorf("some err")
			}},
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.TODO()
			cli, err := New(ctx, "test_queue_url_2", "TEST_MSG_GROUP_ID_2")
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.sqs = tt.mock

			got, err := cli.Receive(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestDelete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mock    SQS
		wantErr bool
	}{
		{
			name: "success",
			mock: sqsmock{deleteFn: func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
				return &sqs.DeleteMessageOutput{}, nil
			}},
			wantErr: false,
		},
		{
			name: "error",
			mock: sqsmock{deleteFn: func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
				return nil, fmt.Errorf("some err")
			}},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.TODO()
			cli, err := New(ctx, "test_queue_url_3", "TEST_MSG_GROUP_ID_3")
			require.NoError(t, err)
			require.NotNil(t, cli)

			cli.sqs = tt.mock

			err = cli.Delete(ctx, &Message{})
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}
