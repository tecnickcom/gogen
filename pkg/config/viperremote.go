package config

import (
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/spf13/viper"
)

// ErrViperRemoteNotRegistered indicates that ViperRemoteLoader was used
// without registering the legacy Viper remote backends in the application via
// a blank import of github.com/spf13/viper/remote.
var ErrViperRemoteNotRegistered = errors.New(`viper remote support is not registered: add a blank import of "github.com/spf13/viper/remote" to the application`)

// viperRemoteProvider adapts RemoteSourceConfig to the viper.RemoteProvider
// interface expected by the legacy Viper remote backends.
type viperRemoteProvider struct {
	rs *RemoteSourceConfig
}

func (p *viperRemoteProvider) Provider() string      { return p.rs.Provider }
func (p *viperRemoteProvider) Endpoint() string      { return p.rs.Endpoint }
func (p *viperRemoteProvider) Path() string          { return p.rs.Path }
func (p *viperRemoteProvider) SecretKeyring() string { return p.rs.SecretKeyring }

// ViperRemoteLoader is a RemoteLoaderFunc backed by the legacy Viper remote
// backends (consul, etcd, etcd3, firestore, nats), to be registered with the
// WithRemoteLoader option.
//
// The backends and their client dependencies are NOT included in this package:
// the application MUST register them manually with a blank import of the viper
// remote package (which sets viper.RemoteConfig at init time):
//
//	import _ "github.com/spf13/viper/remote" // registers the legacy remote backends
//
//	err := config.Load(cmdName, configDir, envPrefix, cfg, config.WithRemoteLoader(config.ViperRemoteLoader))
//
// This keeps github.com/spf13/viper/remote (and its transitive backend
// clients) in the application module only. If the blank import is missing,
// loading from a remote provider fails with ErrViperRemoteNotRegistered.
//
// The provider name must be one of viper.SupportedRemoteProviders (unknown
// names would otherwise be silently routed to the consul backend by the viper
// remote package), and both remoteConfigEndpoint and remoteConfigPath must be
// set (an empty endpoint would otherwise silently fall back to the backend
// client's default address, e.g. localhost:8500 for consul).
func ViperRemoteLoader(rs *RemoteSourceConfig) (io.Reader, error) {
	if viper.RemoteConfig == nil {
		return nil, ErrViperRemoteNotRegistered
	}

	if !slices.Contains(viper.SupportedRemoteProviders, rs.Provider) {
		return nil, fmt.Errorf("%w: remoteConfigProvider %q is not supported by the legacy viper remote backends (must be one of: %s)", ErrInvalidRemoteConfig, rs.Provider, strings.Join(viper.SupportedRemoteProviders, ", "))
	}

	if rs.Endpoint == "" {
		return nil, fmt.Errorf("%w: the %q provider requires remoteConfigEndpoint to be set", ErrMissingRemoteVar, rs.Provider)
	}

	if rs.Path == "" {
		return nil, fmt.Errorf("%w: the %q provider requires remoteConfigPath to be set", ErrMissingRemoteVar, rs.Provider)
	}

	// The error is wrapped with the provider context by loadFromRemoteLoader.
	return viper.RemoteConfig.Get(&viperRemoteProvider{rs: rs}) //nolint:wrapcheck
}
