package s3

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/stretchr/testify/require"
	"github.com/tecnickcom/gogen/pkg/awsopt"
	"github.com/tecnickcom/gogen/pkg/testutil"
)

func TestNew(t *testing.T) {
	o := awsopt.Options{}

	got, err := New(
		t.Context(),
		"name",
		WithAWSOptions(o),
		WithEndpointImmutable("https://test.endpoint.invalid"),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "name", got.bucketName)

	got, err = New(
		t.Context(),
		"name",
		WithAWSOptions(o),
		WithEndpointMutable("https://test.endpoint.invalid"),
	)

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "name", got.bucketName)

	// empty bucket name fails fast before any AWS configuration is loaded
	got, err = New(t.Context(), "")
	require.ErrorIs(t, err, ErrEmptyBucketName)
	require.Nil(t, got)

	// make AWS lib to return an error
	t.Setenv("AWS_ENABLE_ENDPOINT_DISCOVERY", "ERROR")

	got, err = New(t.Context(), "name")
	require.Error(t, err)
	require.Nil(t, got)
}

type s3mock struct {
	delFn  func(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	getFn  func(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	headFn func(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	listFn func(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	putFn  func(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

func (s s3mock) DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	return s.delFn(ctx, params, optFns...)
}

func (s s3mock) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	return s.getFn(ctx, params, optFns...)
}

func (s s3mock) HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
	return s.headFn(ctx, params, optFns...)
}

func (s s3mock) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	return s.listFn(ctx, params, optFns...)
}

func (s s3mock) PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	return s.putFn(ctx, params, optFns...)
}

func TestS3Client_DeleteObject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		bucket  string
		mock    S3
		wantErr bool
	}{
		{
			name:   "success",
			key:    "k1",
			bucket: "bucket",
			mock: s3mock{delFn: func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				return &s3.DeleteObjectOutput{}, nil
			}},
			wantErr: false,
		},
		{
			name:   "error",
			key:    "k1",
			bucket: "bucket",
			mock: s3mock{delFn: func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
				return nil, errors.New("some err")
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			cli, err := New(ctx, tt.bucket, WithS3Client(tt.mock))
			require.NoError(t, err)
			require.NotNil(t, cli)

			err = cli.Delete(ctx, tt.key)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestS3Client_GetObject(t *testing.T) {
	t.Parallel()

	testTime := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name    string
		key     string
		bucket  string
		mock    S3
		want    *Object
		wantErr bool
	}{
		{
			name:   "success",
			key:    "k1",
			bucket: "bucket",
			mock: s3mock{getFn: func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{
					Body:          io.NopCloser(strings.NewReader("test str")),
					ContentType:   aws.String("text/plain"),
					ContentLength: aws.Int64(8),
					ETag:          aws.String(`"abc123"`),
					LastModified:  aws.Time(testTime),
				}, nil
			}},
			want: &Object{
				bucket:        "bucket",
				key:           "k1",
				contentType:   "text/plain",
				contentLength: 8,
				etag:          `"abc123"`,
				lastModified:  testTime,
				body:          io.NopCloser(strings.NewReader("test str")),
			},
			wantErr: false,
		},

		{
			name:   "error",
			key:    "k1",
			bucket: "bucket",
			mock: s3mock{getFn: func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, errors.New("some err")
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "nil response",
			key:    "k1",
			bucket: "bucket",
			mock: s3mock{getFn: func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return nil, nil //nolint:nilnil
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "nil body",
			key:    "k1",
			bucket: "bucket",
			mock: s3mock{getFn: func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
				return &s3.GetObjectOutput{Body: nil}, nil
			}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			cli, err := New(ctx, tt.bucket, WithS3Client(tt.mock))
			require.NoError(t, err)
			require.NotNil(t, cli)

			got, err := cli.Get(ctx, tt.key)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, got)
			require.Equal(t, tt.want, got)

			require.Equal(t, tt.want.ContentType(), got.ContentType())
			require.Equal(t, tt.want.ContentLength(), got.ContentLength())
			require.Equal(t, tt.want.ETag(), got.ETag())
			require.Equal(t, tt.want.LastModified(), got.LastModified())

			expectedBytes, err := io.ReadAll(tt.want.body)
			require.NoError(t, err)
			gotBytes, err := io.ReadAll(got.body)
			require.NoError(t, err)

			require.Equal(t, string(expectedBytes), string(gotBytes))
		})
	}
}

func TestObject_Body(t *testing.T) {
	t.Parallel()

	body := io.NopCloser(strings.NewReader("test str"))
	obj := &Object{bucket: "bucket", key: "k1", body: body}

	got := obj.Body()
	require.NotNil(t, got)

	gotBytes, err := io.ReadAll(got)
	require.NoError(t, err)
	require.Equal(t, "test str", string(gotBytes))
}

func TestObject_Close(t *testing.T) {
	t.Parallel()

	t.Run("success", func(t *testing.T) {
		t.Parallel()

		obj := &Object{
			bucket: "bucket",
			key:    "k1",
			body:   io.NopCloser(strings.NewReader("test str")),
		}

		err := obj.Close()
		require.NoError(t, err)
	})

	t.Run("error", func(t *testing.T) {
		t.Parallel()

		obj := &Object{
			bucket: "bucket",
			key:    "k1",
			body:   testutil.NewErrorCloser("close failed"),
		}

		err := obj.Close()
		require.Error(t, err)
	})
}

func TestS3Client_ListObject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prefix  string
		bucket  string
		mock    S3
		want    []string
		wantErr bool
	}{
		{
			name:   "success - all",
			prefix: "",
			bucket: "bucket",
			mock: s3mock{listFn: func(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{
					Contents: []types.Object{
						{Key: aws.String("key1")},
						{Key: aws.String("another_key")},
					},
				}, nil
			}},
			want:    []string{"key1", "another_key"},
			wantErr: false,
		},
		{
			name:   "success - prefix",
			prefix: "ke",
			bucket: "bucket",
			mock: s3mock{listFn: func(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				return &s3.ListObjectsV2Output{
					Contents: []types.Object{
						{Key: aws.String("key1")},
					},
				}, nil
			}},
			want:    []string{"key1"},
			wantErr: false,
		},
		{
			name:   "success - paginated over multiple pages",
			prefix: "",
			bucket: "bucket",
			mock: s3mock{listFn: func(_ context.Context, params *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				switch aws.ToString(params.ContinuationToken) {
				case "":
					return &s3.ListObjectsV2Output{
						Contents: []types.Object{
							{Key: aws.String("key1")},
							{Key: aws.String("key2")},
						},
						IsTruncated:           aws.Bool(true),
						NextContinuationToken: aws.String("token1"),
					}, nil
				case "token1":
					return &s3.ListObjectsV2Output{
						Contents: []types.Object{
							{Key: aws.String("key3")},
						},
						IsTruncated:           aws.Bool(true),
						NextContinuationToken: aws.String("token2"),
					}, nil
				case "token2":
					return &s3.ListObjectsV2Output{
						Contents: []types.Object{
							{Key: aws.String("key4")},
						},
						IsTruncated: aws.Bool(false),
					}, nil
				default:
					return nil, errors.New("unexpected continuation token")
				}
			}},
			want:    []string{"key1", "key2", "key3", "key4"},
			wantErr: false,
		},
		{
			name:   "success - nil response is a terminal page",
			prefix: "",
			bucket: "bucket",
			mock: s3mock{listFn: func(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				return nil, nil //nolint:nilnil
			}},
			want:    []string{},
			wantErr: false,
		},
		{
			name:   "success - unchanged continuation token stops pagination",
			prefix: "",
			bucket: "bucket",
			mock: s3mock{listFn: func(_ context.Context, params *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				if params.ContinuationToken == nil {
					return &s3.ListObjectsV2Output{
						Contents:              []types.Object{{Key: aws.String("key1")}},
						IsTruncated:           aws.Bool(true),
						NextContinuationToken: aws.String("stuck"),
					}, nil
				}

				// A non-conformant endpoint that keeps returning the same token.
				return &s3.ListObjectsV2Output{
					Contents:              []types.Object{{Key: aws.String("key2")}},
					IsTruncated:           aws.Bool(true),
					NextContinuationToken: aws.String("stuck"),
				}, nil
			}},
			want:    []string{"key1", "key2"},
			wantErr: false,
		},
		{
			name:   "error",
			prefix: "k1",
			bucket: "bucket",
			mock: s3mock{listFn: func(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				return nil, errors.New("some err")
			}},
			want:    nil,
			wantErr: true,
		},
		{
			name:   "error - second page",
			prefix: "",
			bucket: "bucket",
			mock: s3mock{listFn: func(_ context.Context, params *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
				if params.ContinuationToken == nil {
					return &s3.ListObjectsV2Output{
						Contents: []types.Object{
							{Key: aws.String("key1")},
						},
						IsTruncated:           aws.Bool(true),
						NextContinuationToken: aws.String("token1"),
					}, nil
				}

				return nil, errors.New("some err")
			}},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			cli, err := New(ctx, tt.bucket, WithS3Client(tt.mock))
			require.NoError(t, err)
			require.NotNil(t, cli)

			got, err := cli.ListKeys(ctx, tt.prefix)
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

func TestS3Client_ListObjects(t *testing.T) {
	t.Parallel()

	modified := time.Date(2024, 5, 6, 7, 8, 9, 0, time.UTC)

	mock := s3mock{listFn: func(_ context.Context, _ *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
		return &s3.ListObjectsV2Output{
			Contents: []types.Object{
				{Key: aws.String("key1"), Size: aws.Int64(123), LastModified: aws.Time(modified), ETag: aws.String(`"e1"`)},
				{Key: aws.String("key2")},
			},
		}, nil
	}}

	ctx := t.Context()
	cli, err := New(ctx, "bucket", WithS3Client(mock))
	require.NoError(t, err)
	require.NotNil(t, cli)

	got, err := cli.ListObjects(ctx, "")
	require.NoError(t, err)
	require.Equal(t, []ObjectInfo{
		{Key: "key1", Size: 123, LastModified: modified, ETag: `"e1"`},
		{Key: "key2"},
	}, got)
}

func TestS3Client_PutObject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		bucket  string
		mock    S3
		wantErr bool
	}{
		{
			name:   "success",
			key:    "k1",
			bucket: "bucket",
			mock: s3mock{putFn: func(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return &s3.PutObjectOutput{}, nil
			}},
			wantErr: false,
		},
		{
			name:   "error",
			key:    "k1",
			bucket: "bucket",
			mock: s3mock{putFn: func(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
				return nil, errors.New("some err")
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			cli, err := New(ctx, tt.bucket, WithS3Client(tt.mock))
			require.NoError(t, err)
			require.NotNil(t, cli)

			err = cli.Put(ctx, tt.key, nil)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestS3Client_EmptyKey(t *testing.T) {
	t.Parallel()

	// A mock whose methods panic if called: an empty key must fail fast before
	// any upstream call is made.
	panicMock := s3mock{
		delFn: func(_ context.Context, _ *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
			panic("must not be called")
		},
		getFn: func(_ context.Context, _ *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
			panic("must not be called")
		},
		putFn: func(_ context.Context, _ *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
			panic("must not be called")
		},
	}

	ctx := t.Context()
	cli, err := New(ctx, "bucket", WithS3Client(panicMock))
	require.NoError(t, err)
	require.NotNil(t, cli)

	require.ErrorIs(t, cli.Delete(ctx, ""), ErrEmptyKey)

	obj, err := cli.Get(ctx, "")
	require.ErrorIs(t, err, ErrEmptyKey)
	require.Nil(t, obj)

	require.ErrorIs(t, cli.Put(ctx, "", nil), ErrEmptyKey)
}

func TestS3Client_HealthCheck(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mock    S3
		wantErr bool
	}{
		{
			name: "success",
			mock: s3mock{headFn: func(_ context.Context, _ *s3.HeadBucketInput, _ ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
				return &s3.HeadBucketOutput{}, nil
			}},
			wantErr: false,
		},
		{
			name: "error",
			mock: s3mock{headFn: func(_ context.Context, _ *s3.HeadBucketInput, _ ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
				return nil, errors.New("some err")
			}},
			wantErr: true,
		},
		{
			name: "nil response",
			mock: s3mock{headFn: func(_ context.Context, _ *s3.HeadBucketInput, _ ...func(*s3.Options)) (*s3.HeadBucketOutput, error) {
				return nil, nil //nolint:nilnil
			}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()
			cli, err := New(ctx, "bucket", WithS3Client(tt.mock))
			require.NoError(t, err)
			require.NotNil(t, cli)

			err = cli.HealthCheck(ctx)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestObject_BucketKey(t *testing.T) {
	t.Parallel()

	obj := &Object{
		bucket: "bucket",
		key:    "k1",
		body:   io.NopCloser(strings.NewReader("test str")),
	}

	require.Equal(t, "bucket", obj.Bucket())
	require.Equal(t, "k1", obj.Key())
}
