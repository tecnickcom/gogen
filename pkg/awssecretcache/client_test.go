package awssecretcache

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/nurago/pkg/awsopt"
)

type mockSecretsManagerClient struct {
	getSecretValue func(ctx context.Context, params *awssm.GetSecretValueInput, optFns ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error)
}

func (m *mockSecretsManagerClient) GetSecretValue(ctx context.Context, params *awssm.GetSecretValueInput, optFns ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
	return m.getSecretValue(ctx, params, optFns...)
}

func TestNew(t *testing.T) {
	o := awsopt.Options{}
	o.WithRegion("eu-west-1")

	got, err := New(
		t.Context(),
		1,
		1*time.Second,
		WithAWSOptions(o),
		WithEndpointImmutable("https://test.endpoint.invalid"),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.cache)

	got, err = New(
		t.Context(),
		1,
		1*time.Second,
		WithAWSOptions(o),
		WithEndpointMutable("https://test.endpoint.invalid"),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.cache)

	// make AWS lib to return an error
	t.Setenv("AWS_ENABLE_ENDPOINT_DISCOVERY", "ERROR")

	got, err = New(t.Context(), 1, 1*time.Second)
	require.Error(t, err)
	require.Nil(t, got)

	// An injected client bypasses AWS config loading entirely, so New must
	// succeed even though the environment above would otherwise fail the load.
	got, err = New(
		t.Context(),
		1,
		1*time.Second,
		WithSecretsManagerClient(&mockSecretsManagerClient{
			getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
				return &awssm.GetSecretValueOutput{}, nil
			},
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.cache)
}

func Test_GetSecretData(t *testing.T) {
	t.Parallel()

	secval := "secret_binary_value"

	tests := []struct {
		name    string
		mock    SecretsManagerClient
		wantErr bool
	}{
		{
			name: "success",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{
						SecretBinary: []byte(secval),
						SecretString: &secval,
					}, nil
				},
			},
			wantErr: false,
		},
		{
			name: "error",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return nil, errors.New("error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(
				t.Context(),
				1,
				1*time.Second,
				WithSecretsManagerClient(tt.mock),
			)

			require.NoError(t, err)
			require.NotNil(t, c)

			got, err := c.GetSecretData(t.Context(), "test_key")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, []byte(secval), got.SecretBinary)
				require.Equal(t, &secval, got.SecretString)
			}
		})
	}
}

func Test_GetSecretBinary(t *testing.T) {
	t.Parallel()

	secval := "secret_binary_value"

	tests := []struct {
		name    string
		mock    SecretsManagerClient
		want    []byte
		wantErr bool
	}{
		{
			name: "success with SecretBinary",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{SecretBinary: []byte(secval)}, nil
				},
			},
			want:    []byte(secval),
			wantErr: false,
		},
		{
			name: "success with SecretString",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{SecretString: &secval}, nil
				},
			},
			want:    []byte(secval),
			wantErr: false,
		},
		{
			name: "empty SecretString is a real value, not ErrEmptySecret",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{SecretString: aws.String("")}, nil
				},
			},
			want:    []byte{},
			wantErr: false,
		},
		{
			name: "empty SecretBinary is a real value, not ErrEmptySecret",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{SecretBinary: []byte{}}, nil
				},
			},
			want:    []byte{},
			wantErr: false,
		},
		{
			name: "empty secret (neither SecretString nor SecretBinary)",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{}, nil
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "error",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return nil, errors.New("error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(
				t.Context(),
				1,
				1*time.Second,
				WithSecretsManagerClient(tt.mock),
			)

			require.NoError(t, err)
			require.NotNil(t, c)

			got, err := c.GetSecretBinary(t.Context(), "test_key")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_GetSecretString(t *testing.T) {
	t.Parallel()

	secval := "secret_string_value"

	tests := []struct {
		name    string
		mock    SecretsManagerClient
		want    string
		wantErr bool
	}{
		{
			name: "success with SecretBinary",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{SecretBinary: []byte(secval)}, nil
				},
			},
			want:    secval,
			wantErr: false,
		},
		{
			name: "success with SecretString",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{SecretString: &secval}, nil
				},
			},
			want:    secval,
			wantErr: false,
		},
		{
			name: "empty SecretString is a real value, not ErrEmptySecret",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{SecretString: aws.String("")}, nil
				},
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "empty SecretBinary is a real value, not ErrEmptySecret",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{SecretBinary: []byte{}}, nil
				},
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "empty secret (neither SecretString nor SecretBinary)",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{}, nil
				},
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "error",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return nil, errors.New("error")
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			c, err := New(
				t.Context(),
				1,
				1*time.Second,
				WithSecretsManagerClient(tt.mock),
			)

			require.NoError(t, err)
			require.NotNil(t, c)

			got, err := c.GetSecretString(t.Context(), "test_key")
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.want, got)
			}
		})
	}
}

func Test_Len(t *testing.T) {
	t.Parallel()

	secval := "secret_string_value_len"

	smclient := &mockSecretsManagerClient{
		getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
			return &awssm.GetSecretValueOutput{SecretString: &secval}, nil
		},
	}

	c, err := New(
		t.Context(),
		3,
		10*time.Second,
		WithSecretsManagerClient(smclient),
	)

	require.NoError(t, err)
	require.NotNil(t, c)

	// cache miss
	got, err := c.GetSecretString(t.Context(), "test_key_1")
	require.NoError(t, err)
	require.Equal(t, secval, got)

	require.Equal(t, 1, c.Len())

	// cache miss
	got, err = c.GetSecretString(t.Context(), "test_key_2")
	require.NoError(t, err)
	require.Equal(t, secval, got)

	require.Equal(t, 2, c.Len())
}

func Test_Reset(t *testing.T) {
	t.Parallel()

	secval := "secret_string_value_reset"

	smclient := &mockSecretsManagerClient{
		getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
			return &awssm.GetSecretValueOutput{SecretString: &secval}, nil
		},
	}

	c, err := New(
		t.Context(),
		3,
		10*time.Second,
		WithSecretsManagerClient(smclient),
	)

	require.NoError(t, err)
	require.NotNil(t, c)

	// cache miss
	got, err := c.GetSecretString(t.Context(), "test_key_1")
	require.NoError(t, err)
	require.Equal(t, secval, got)

	// cache miss
	got, err = c.GetSecretString(t.Context(), "test_key_2")
	require.NoError(t, err)
	require.Equal(t, secval, got)

	require.Equal(t, 2, c.Len())

	c.Reset()

	require.Empty(t, c.Len())
}

func Test_Remove(t *testing.T) {
	t.Parallel()

	secval := "secret_string_value_reset"

	smclient := &mockSecretsManagerClient{
		getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
			return &awssm.GetSecretValueOutput{SecretString: &secval}, nil
		},
	}

	c, err := New(
		t.Context(),
		3,
		10*time.Second,
		WithSecretsManagerClient(smclient),
	)

	require.NoError(t, err)
	require.NotNil(t, c)

	// cache miss
	got, err := c.GetSecretString(t.Context(), "test_key_1")
	require.NoError(t, err)
	require.Equal(t, secval, got)

	// cache miss
	got, err = c.GetSecretString(t.Context(), "test_key_2")
	require.NoError(t, err)
	require.Equal(t, secval, got)

	require.Equal(t, 2, c.Len())

	c.Remove("test_key_1")

	require.Equal(t, 1, c.Len())
}

// Test_GetSecretBinary_returns_copy verifies that zeroing the returned bytes
// (common secret hygiene) does not corrupt the cached entry shared with other
// callers.
func Test_GetSecretBinary_returns_copy(t *testing.T) {
	t.Parallel()

	var calls int

	smclient := &mockSecretsManagerClient{
		getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
			calls++

			return &awssm.GetSecretValueOutput{
				SecretBinary: []byte("secret_binary_value"),
			}, nil
		},
	}

	c, err := New(
		t.Context(),
		2,
		1*time.Minute,
		WithSecretsManagerClient(smclient),
	)

	require.NoError(t, err)
	require.NotNil(t, c)

	val, err := c.GetSecretBinary(t.Context(), "test_key_copy")
	require.NoError(t, err)
	require.Equal(t, []byte("secret_binary_value"), val)

	// Zeroing the returned bytes must not corrupt the cached entry.
	for i := range val {
		val[i] = 0
	}

	val, err = c.GetSecretBinary(t.Context(), "test_key_copy")
	require.NoError(t, err)
	require.Equal(t, []byte("secret_binary_value"), val)
	require.Equal(t, 1, calls, "the second call must be served from cache")
}

// Test_stale_if_error verifies the WithStaleIfError pass-through: a failed
// refresh serves the last known good secret, and PurgeExpired forfeits the
// stale protection.
func Test_stale_if_error(t *testing.T) {
	t.Parallel()

	secval := "secret_string_stale"

	var calls int

	smclient := &mockSecretsManagerClient{
		getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
			calls++

			if calls == 1 {
				return &awssm.GetSecretValueOutput{SecretString: &secval}, nil
			}

			return nil, errors.New("mock AWS outage")
		},
	}

	c, err := New(
		t.Context(),
		3,
		100*time.Millisecond,
		WithSecretsManagerClient(smclient),
		WithStaleIfError(1*time.Minute),
	)

	require.NoError(t, err)
	require.NotNil(t, c)

	got, err := c.GetSecretString(t.Context(), "test_key_stale")
	require.NoError(t, err)
	require.Equal(t, secval, got)

	time.Sleep(150 * time.Millisecond) // let the entry expire

	// The refresh fails: the last known good secret is served.
	got, err = c.GetSecretString(t.Context(), "test_key_stale")
	require.NoError(t, err, "a failed refresh must serve the stale secret")
	require.Equal(t, secval, got)
	require.Equal(t, 2, calls)

	// Purging expired entries forfeits the stale protection.
	require.Equal(t, 1, c.PurgeExpired())

	_, err = c.GetSecretString(t.Context(), "test_key_stale")
	require.Error(t, err, "after PurgeExpired the stale secret must be gone")
}

// Test_EmptySecret verifies that a response carrying no secret material is
// surfaced as ErrEmptySecret rather than panicking or silently returning an
// empty value. A nil upstream output (which an injected client can produce)
// fails all three getters; an output present but with neither SecretString nor
// SecretBinary set fails only the value getters, while GetSecretData still
// returns the raw output so metadata stays accessible.
func Test_EmptySecret(t *testing.T) {
	t.Parallel()

	t.Run("nil output", func(t *testing.T) {
		t.Parallel()

		c, err := New(
			t.Context(),
			1,
			1*time.Second,
			WithSecretsManagerClient(&mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return nil, nil //nolint:nilnil
				},
			}),
		)
		require.NoError(t, err)

		_, err = c.GetSecretData(t.Context(), "k")
		require.ErrorIs(t, err, ErrEmptySecret)

		_, err = c.GetSecretBinary(t.Context(), "k")
		require.ErrorIs(t, err, ErrEmptySecret)

		_, err = c.GetSecretString(t.Context(), "k")
		require.ErrorIs(t, err, ErrEmptySecret)
	})

	t.Run("output without value", func(t *testing.T) {
		t.Parallel()

		c, err := New(
			t.Context(),
			1,
			1*time.Second,
			WithSecretsManagerClient(&mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{}, nil
				},
			}),
		)
		require.NoError(t, err)

		// GetSecretData returns the raw (empty) output so metadata stays usable.
		got, err := c.GetSecretData(t.Context(), "k")
		require.NoError(t, err)
		require.NotNil(t, got)

		_, err = c.GetSecretBinary(t.Context(), "k")
		require.ErrorIs(t, err, ErrEmptySecret)

		_, err = c.GetSecretString(t.Context(), "k")
		require.ErrorIs(t, err, ErrEmptySecret)
	})
}

// Test_EmptySecretID verifies that an empty secret id is rejected up front with
// ErrEmptySecretID, without reaching the upstream client or creating a cache
// entry.
func Test_EmptySecretID(t *testing.T) {
	t.Parallel()

	var calls int

	c, err := New(
		t.Context(),
		1,
		1*time.Second,
		WithSecretsManagerClient(&mockSecretsManagerClient{
			getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
				calls++

				return &awssm.GetSecretValueOutput{SecretString: aws.String("value")}, nil
			},
		}),
	)
	require.NoError(t, err)
	require.NotNil(t, c)

	_, err = c.GetSecretData(t.Context(), "")
	require.ErrorIs(t, err, ErrEmptySecretID)

	_, err = c.GetSecretBinary(t.Context(), "")
	require.ErrorIs(t, err, ErrEmptySecretID)

	_, err = c.GetSecretString(t.Context(), "")
	require.ErrorIs(t, err, ErrEmptySecretID)

	require.Zero(t, calls, "an empty secret id must not reach the upstream client")
	require.Zero(t, c.Len(), "an empty secret id must not create a cache entry")
}

// Test_single_flight verifies the headline guarantee at this layer: many
// goroutines racing a cold key collapse into a single upstream GetSecretValue
// call and all observe the same value.
func Test_single_flight(t *testing.T) {
	t.Parallel()

	secval := "single_flight_secret"

	var calls atomic.Int32

	smclient := &mockSecretsManagerClient{
		getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
			calls.Add(1)

			return &awssm.GetSecretValueOutput{SecretString: &secval}, nil
		},
	}

	c, err := New(t.Context(), 4, 1*time.Minute, WithSecretsManagerClient(smclient))
	require.NoError(t, err)
	require.NotNil(t, c)

	const n = 32

	var wg sync.WaitGroup

	results := make([]string, n)
	errs := make([]error, n)

	wg.Add(n)

	for i := range n {
		go func(i int) {
			defer wg.Done()

			results[i], errs[i] = c.GetSecretString(t.Context(), "same_key")
		}(i)
	}

	wg.Wait()

	require.Equal(t, int32(1), calls.Load(), "concurrent cold callers must collapse to a single upstream lookup")

	for i := range n {
		require.NoError(t, errs[i])
		require.Equal(t, secval, results[i])
	}
}
