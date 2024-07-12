package awssecretcache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Vonage/gosrvlib/pkg/awsopt"
	awssm "github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/stretchr/testify/require"
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
		context.TODO(),
		1,
		1*time.Second,
		WithAWSOptions(o),
		WithEndpointImmutable("https://test.endpoint.invalid"),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.cache)

	got, err = New(
		context.TODO(),
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

	got, err = New(context.TODO(), 1, 1*time.Second)
	require.Error(t, err)
	require.Nil(t, got)
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
				context.TODO(),
				1,
				1*time.Second,
				WithSecretsManagerClient(tt.mock),
			)

			require.NoError(t, err)
			require.NotNil(t, c)

			got, err := c.GetSecretData(context.TODO(), "test_key")
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
			name: "success with nil SecretBinary",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{}, nil
				},
			},
			want:    nil,
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
				context.TODO(),
				1,
				1*time.Second,
				WithSecretsManagerClient(tt.mock),
			)

			require.NoError(t, err)
			require.NotNil(t, c)

			got, err := c.GetSecretBinary(context.TODO(), "test_key")
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
			name: "success with nil SecretBinary",
			mock: &mockSecretsManagerClient{
				getSecretValue: func(_ context.Context, _ *awssm.GetSecretValueInput, _ ...func(*awssm.Options)) (*awssm.GetSecretValueOutput, error) {
					return &awssm.GetSecretValueOutput{}, nil
				},
			},
			want:    "",
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
				context.TODO(),
				1,
				1*time.Second,
				WithSecretsManagerClient(tt.mock),
			)

			require.NoError(t, err)
			require.NotNil(t, c)

			got, err := c.GetSecretString(context.TODO(), "test_key")
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
		context.TODO(),
		3,
		10*time.Second,
		WithSecretsManagerClient(smclient),
	)

	require.NoError(t, err)
	require.NotNil(t, c)

	// cache miss
	got, err := c.GetSecretString(context.TODO(), "test_key_1")
	require.NoError(t, err)
	require.Equal(t, secval, got)

	require.Equal(t, 1, c.Len())

	// cache miss
	got, err = c.GetSecretString(context.TODO(), "test_key_2")
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
		context.TODO(),
		3,
		10*time.Second,
		WithSecretsManagerClient(smclient),
	)

	require.NoError(t, err)
	require.NotNil(t, c)

	// cache miss
	got, err := c.GetSecretString(context.TODO(), "test_key_1")
	require.NoError(t, err)
	require.Equal(t, secval, got)

	// cache miss
	got, err = c.GetSecretString(context.TODO(), "test_key_2")
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
		context.TODO(),
		3,
		10*time.Second,
		WithSecretsManagerClient(smclient),
	)

	require.NoError(t, err)
	require.NotNil(t, c)

	// cache miss
	got, err := c.GetSecretString(context.TODO(), "test_key_1")
	require.NoError(t, err)
	require.Equal(t, secval, got)

	// cache miss
	got, err = c.GetSecretString(context.TODO(), "test_key_2")
	require.NoError(t, err)
	require.Equal(t, secval, got)

	require.Equal(t, 2, c.Len())

	c.Remove("test_key_1")

	require.Equal(t, 1, c.Len())
}
