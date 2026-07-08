package kafka

import (
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/require"
)

func Test_defaultConfig(t *testing.T) {
	t.Parallel()

	cfg := defaultConfig()
	require.NotNil(t, cfg)
	require.Equal(t, defaultSessionTimeout, cfg.sessionTimeout)
	require.Equal(t, kafka.LastOffset, cfg.startOffset)
	require.NotNil(t, cfg.messageEncodeFunc)
	require.NotNil(t, cfg.messageDecodeFunc)
	require.NotNil(t, cfg.balancer)
	require.Equal(t, kafka.RequireAll, cfg.requiredAcks)
	require.Equal(t, 0, cfg.batchSize)
	require.Equal(t, defaultBatchTimeout, cfg.batchTimeout)
	require.Nil(t, cfg.reader)
	require.Nil(t, cfg.writer)
	require.Nil(t, cfg.checkFn)
}

func Test_validateTopic(t *testing.T) {
	t.Parallel()

	require.ErrorIs(t, validateTopic(""), ErrInvalidOptions)
	require.NoError(t, validateTopic("topic1"))
}

func Test_validateBrokers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		brokers []string
		wantErr bool
	}{
		{
			name:    "nil list",
			brokers: nil,
			wantErr: true,
		},
		{
			name:    "empty list",
			brokers: []string{},
			wantErr: true,
		},
		{
			name:    "empty entry",
			brokers: []string{""},
			wantErr: true,
		},
		{
			name:    "blank entry",
			brokers: []string{"url1:9092", "  "},
			wantErr: true,
		},
		{
			name:    "valid single entry without port",
			brokers: []string{"url1"},
			wantErr: false,
		},
		{
			name:    "valid entries",
			brokers: []string{"url1:9092", "url2:9092"},
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateBrokers(tt.brokers)

			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidOptions)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func Test_validateSessionTimeout(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		timeout time.Duration
		wantErr bool
	}{
		{
			name:    "negative",
			timeout: -time.Second,
			wantErr: true,
		},
		{
			name:    "zero",
			timeout: 0,
			wantErr: true,
		},
		{
			name:    "at the upper bound",
			timeout: maxSessionTimeout,
			wantErr: true,
		},
		{
			name:    "just below the upper bound",
			timeout: maxSessionTimeout - time.Millisecond,
			wantErr: false,
		},
		{
			name:    "valid",
			timeout: 10 * time.Second,
			wantErr: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			err := validateSessionTimeout(tt.timeout)

			if tt.wantErr {
				require.ErrorIs(t, err, ErrInvalidOptions)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
