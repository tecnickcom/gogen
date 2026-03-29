package s3

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3 is the minimal AWS SDK S3 API surface required by [Client].
type S3 interface {
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// Client wraps AWS SDK S3 operations for a single target bucket.
type Client struct {
	s3         S3
	bucketName string
}

// New builds a client for bucketName using configured AWS credentials and S3 options.
func New(ctx context.Context, bucketName string, opts ...Option) (*Client, error) {
	cfg, err := loadConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create a new s3 client: %w", err)
	}

	return &Client{
		s3:         s3.NewFromConfig(cfg.awsConfig, cfg.srvOptFns...),
		bucketName: bucketName,
	}, nil
}

// Object contains metadata and body stream for a downloaded S3 object.
type Object struct {
	bucket string
	key    string
	body   io.ReadCloser
}

// Delete removes the object identified by key from the configured bucket.
func (c *Client) Delete(ctx context.Context, key string) error {
	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(c.bucketName), Key: aws.String(key)})
	if err != nil {
		return fmt.Errorf("cannot delete s3 object: %w", err)
	}

	return nil
}

// Get fetches an object by key and returns an [Object] with its streaming body.
func (c *Client) Get(ctx context.Context, key string) (*Object, error) {
	resp, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get s3 object: %w", err)
	}

	return &Object{bucket: c.bucketName, key: key, body: resp.Body}, nil
}

// ListKeys returns object keys matching prefix; an empty prefix lists all keys in the bucket.
func (c *Client) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	l, err := c.s3.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(c.bucketName), Prefix: aws.String(prefix)})
	if err != nil {
		return nil, fmt.Errorf("cannot list s3 keys: %w", err)
	}

	keysList := make([]string, 0, len(l.Contents))
	for _, key := range l.Contents {
		keysList = append(keysList, aws.ToString(key.Key))
	}

	return keysList, nil
}

// Put uploads reader content to key in the configured bucket.
func (c *Client) Put(ctx context.Context, key string, reader io.Reader) error {
	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(c.bucketName), Key: aws.String(key), Body: reader})
	if err != nil {
		return fmt.Errorf("cannot put s3 object: %w", err)
	}

	return nil
}
