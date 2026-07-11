package config

import (
	"io"
)

// RemoteLoaderFunc loads raw configuration data (in the default JSON format)
// from an arbitrary remote source described by rs.
//
// It is registered via WithRemoteLoader and invoked for any configured
// provider other than the built-in "envvar"; when no provider is configured
// the loader is not called. Returning a nil reader with a nil error means
// there is no remote data to merge. If the returned reader implements
// io.Closer it is closed after reading. This keeps remote-backend client
// dependencies out of this package: the application implements the fetch
// logic with whatever client library it needs.
type RemoteLoaderFunc func(rs *RemoteSourceConfig) (io.Reader, error)

// Option configures optional Load behaviors.
type Option func(o *options)

// options holds the optional Load settings collected from Option values.
type options struct {
	// remoteLoader loads the remote configuration for custom providers.
	remoteLoader RemoteLoaderFunc
}

// WithRemoteLoader registers the function used to load configuration data for
// any remoteConfigProvider other than the built-in "envvar" (e.g. consul,
// etcd, vault, S3, ...). The application supplies the implementation along
// with its client dependencies.
//
// To restore the legacy Viper remote backends plug in the ready-made
// ViperRemoteLoader (see its documentation for the required setup).
func WithRemoteLoader(fn RemoteLoaderFunc) Option {
	return func(o *options) {
		o.remoteLoader = fn
	}
}
