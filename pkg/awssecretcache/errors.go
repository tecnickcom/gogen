package awssecretcache

import "errors"

// Exported sentinel errors returned by this package. Match them with errors.Is.
var (
	// ErrEmptySecretID is returned by the Cache getters when the requested
	// secret id (key) is empty, before any upstream call is made.
	ErrEmptySecretID = errors.New("awssecretcache: secret id must not be empty")

	// ErrEmptySecret is returned by the Cache getters when the Secrets Manager
	// response carries no secret material: a nil response, or a response with
	// neither SecretString nor SecretBinary set.
	ErrEmptySecret = errors.New("awssecretcache: secret contains no value")
)
