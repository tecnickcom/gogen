package s3

import "errors"

// Exported sentinel errors returned by this package. Match them with errors.Is.
var (
	// ErrEmptyBucketName is returned by New when bucketName is empty.
	ErrEmptyBucketName = errors.New("s3: bucket name must not be empty")

	// ErrEmptyKey is returned by Get, Put and Delete when key is empty, before
	// any upstream call is made.
	ErrEmptyKey = errors.New("s3: object key must not be empty")

	// ErrEmptyObjectBody is returned by Get when the response carries no body
	// stream (a nil response or a nil Body).
	ErrEmptyObjectBody = errors.New("s3: object response has no body")

	// ErrBucketNotResponding is returned by HealthCheck when the bucket probe
	// succeeds but returns a nil response.
	ErrBucketNotResponding = errors.New("s3: the bucket is not responding")
)
