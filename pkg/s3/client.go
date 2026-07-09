package s3

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3 is the minimal AWS SDK S3 API surface required by [Client].
//
//nolint:dupl // the test mock necessarily mirrors this method set
type S3 interface {
	DeleteObject(ctx context.Context, params *s3.DeleteObjectInput, optFns ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
	HeadBucket(ctx context.Context, params *s3.HeadBucketInput, optFns ...func(*s3.Options)) (*s3.HeadBucketOutput, error)
	ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
	PutObject(ctx context.Context, params *s3.PutObjectInput, optFns ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// Client wraps AWS SDK S3 operations for a single target bucket.
type Client struct {
	s3         S3
	bucketName string
}

// New builds a client for bucketName using configured AWS credentials and S3 options.
//
// bucketName must not be empty; a cheap check runs before any AWS configuration
// is loaded, so misconfiguration fails fast with [ErrEmptyBucketName]. A custom
// client can be supplied with [WithS3Client], in which case no AWS configuration
// is loaded.
//
// The returned Client is safe for concurrent use.
func New(ctx context.Context, bucketName string, opts ...Option) (*Client, error) {
	if bucketName == "" {
		return nil, ErrEmptyBucketName
	}

	cfg, err := loadConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("cannot create a new s3 client: %w", err)
	}

	client := cfg.s3Client
	if client == nil {
		client = s3.NewFromConfig(cfg.awsConfig, cfg.srvOptFns...)
	}

	return &Client{
		s3:         client,
		bucketName: bucketName,
	}, nil
}

// Object contains metadata and body stream for a downloaded S3 object.
//
// The caller owns the underlying body stream and MUST call [Object.Close] when
// done to release the network connection and avoid leaking resources.
type Object struct {
	bucket        string
	key           string
	contentType   string
	contentLength int64
	etag          string
	lastModified  time.Time
	body          io.ReadCloser
}

// Bucket returns the name of the bucket this object was fetched from.
func (o *Object) Bucket() string {
	return o.bucket
}

// Key returns the object key this object was fetched with.
func (o *Object) Key() string {
	return o.key
}

// ContentType returns the object's Content-Type, or "" when the response did
// not carry one.
func (o *Object) ContentType() string {
	return o.contentType
}

// ContentLength returns the object's size in bytes. GetObject responses always
// carry Content-Length, so 0 denotes a genuinely empty object.
func (o *Object) ContentLength() int64 {
	return o.contentLength
}

// ETag returns the object's entity tag, or "" when the response did not carry one.
func (o *Object) ETag() string {
	return o.etag
}

// LastModified returns the object's last-modified time, or the zero time when
// the response did not carry one.
func (o *Object) LastModified() time.Time {
	return o.lastModified
}

// Body returns the streaming body of the object.
//
// The caller MUST call [Object.Close] (not the returned reader directly) once
// finished reading to release the underlying resources.
func (o *Object) Body() io.ReadCloser {
	return o.body
}

// Close releases the underlying body stream of the object.
//
// It MUST be called once the caller is done with the object to avoid leaking
// the underlying network connection.
func (o *Object) Close() error {
	err := o.body.Close()
	if err != nil {
		return fmt.Errorf("cannot close s3 object body: %w", err)
	}

	return nil
}

// Delete removes the object identified by key from the configured bucket.
//
// It returns [ErrEmptyKey] when key is empty, before any upstream call is made.
func (c *Client) Delete(ctx context.Context, key string) error {
	if key == "" {
		return ErrEmptyKey
	}

	_, err := c.s3.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(c.bucketName), Key: aws.String(key)})
	if err != nil {
		return fmt.Errorf("cannot delete s3 object: %w", err)
	}

	return nil
}

// Get fetches an object by key and returns an [Object] with its streaming body.
//
// It returns [ErrEmptyKey] when key is empty, and [ErrEmptyObjectBody] when the
// response carries no body stream.
func (c *Client) Get(ctx context.Context, key string) (*Object, error) {
	if key == "" {
		return nil, ErrEmptyKey
	}

	resp, err := c.s3.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, fmt.Errorf("cannot get s3 object: %w", err)
	}

	// The AWS SDK never returns a nil output (or nil Body) on success, but an
	// injected S3 client can: guard so the returned Object never wraps a nil body.
	if resp == nil || resp.Body == nil {
		return nil, fmt.Errorf("%w: %s", ErrEmptyObjectBody, key)
	}

	return &Object{
		bucket:        c.bucketName,
		key:           key,
		contentType:   aws.ToString(resp.ContentType),
		contentLength: aws.ToInt64(resp.ContentLength),
		etag:          aws.ToString(resp.ETag),
		lastModified:  aws.ToTime(resp.LastModified),
		body:          resp.Body,
	}, nil
}

// ObjectInfo describes a single object returned by [Client.ListObjects].
type ObjectInfo struct {
	// Key is the object key.
	Key string

	// Size is the object size in bytes.
	Size int64

	// LastModified is the object's last-modified time, or the zero time when unset.
	LastModified time.Time

	// ETag is the object's entity tag, or "" when unset.
	ETag string
}

// ListKeys returns object keys matching prefix; an empty prefix lists all keys in the bucket.
// It transparently paginates over ListObjectsV2 results, so all matching keys are
// returned even when they exceed the per-request AWS limit (1000 keys).
//
// It is a thin projection over [Client.ListObjects]; use ListObjects when the
// per-object size, last-modified time, or ETag is also needed.
func (c *Client) ListKeys(ctx context.Context, prefix string) ([]string, error) {
	objects, err := c.ListObjects(ctx, prefix)
	if err != nil {
		return nil, err
	}

	keysList := make([]string, 0, len(objects))
	for _, obj := range objects {
		keysList = append(keysList, obj.Key)
	}

	return keysList, nil
}

// ListObjects returns metadata for objects matching prefix; an empty prefix lists
// all objects in the bucket. Like [Client.ListKeys], it transparently paginates
// over ListObjectsV2 results, so all matching objects are returned even when they
// exceed the per-request AWS limit (1000 objects).
func (c *Client) ListObjects(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	input := &s3.ListObjectsV2Input{Bucket: aws.String(c.bucketName), Prefix: aws.String(prefix)}
	list := make([]ObjectInfo, 0)

	for {
		l, err := c.s3.ListObjectsV2(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("cannot list s3 objects: %w", err)
		}

		list = appendObjects(list, l)

		next := nextContinuationToken(l, input.ContinuationToken)
		if next == nil {
			break
		}

		input.ContinuationToken = next
	}

	return list, nil
}

// appendObjects appends the objects contained in a ListObjectsV2 page to dst.
// A nil page (which an injected client can return on success) contributes nothing.
func appendObjects(dst []ObjectInfo, page *s3.ListObjectsV2Output) []ObjectInfo {
	if page == nil {
		return dst
	}

	for _, obj := range page.Contents {
		dst = append(dst, ObjectInfo{
			Key:          aws.ToString(obj.Key),
			Size:         aws.ToInt64(obj.Size),
			LastModified: aws.ToTime(obj.LastModified),
			ETag:         aws.ToString(obj.ETag),
		})
	}

	return dst
}

// nextContinuationToken returns the token to fetch the next page, or nil when
// pagination should stop: on a nil or non-truncated page, a missing token, or a
// token that did not advance (which guards against a non-conformant endpoint
// that keeps reporting IsTruncated with an unchanging token and would loop forever).
func nextContinuationToken(page *s3.ListObjectsV2Output, current *string) *string {
	if page == nil || !aws.ToBool(page.IsTruncated) || page.NextContinuationToken == nil {
		return nil
	}

	if aws.ToString(page.NextContinuationToken) == aws.ToString(current) {
		return nil
	}

	return page.NextContinuationToken
}

// Put uploads reader content to key in the configured bucket.
//
// It returns [ErrEmptyKey] when key is empty, before any upstream call is made.
// A nil reader uploads an empty (zero-byte) object.
func (c *Client) Put(ctx context.Context, key string, reader io.Reader) error {
	if key == "" {
		return ErrEmptyKey
	}

	if reader == nil {
		reader = http.NoBody
	}

	_, err := c.s3.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(c.bucketName), Key: aws.String(key), Body: reader})
	if err != nil {
		return fmt.Errorf("cannot put s3 object: %w", err)
	}

	return nil
}

// HealthCheck verifies bucket reachability and access permissions via HeadBucket.
//
// The probe uses HeadBucket and therefore requires s3:ListBucket permission on
// the bucket; a permissions error surfaces as a wrapped AWS error, not as a
// bucket outage.
//
// It returns [ErrBucketNotResponding] when the probe succeeds but returns a nil
// response; the underlying AWS error is wrapped otherwise.
func (c *Client) HealthCheck(ctx context.Context) error {
	resp, err := c.s3.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: aws.String(c.bucketName)})
	if err != nil {
		return fmt.Errorf("unable to connect to AWS S3 bucket %q: %w", c.bucketName, err)
	}

	// The AWS SDK never returns a nil output on success, but an injected S3
	// client can: guard against it.
	if resp == nil {
		return fmt.Errorf("%w: %s", ErrBucketNotResponding, c.bucketName)
	}

	return nil
}
