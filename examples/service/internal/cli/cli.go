// Package cli contains the CLI entry point.
package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/nuragoexampleowner/nuragoexample/internal/metrics"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/tecnickcom/nurago/pkg/bootstrap"
	"github.com/tecnickcom/nurago/pkg/config"
	"github.com/tecnickcom/nurago/pkg/httputil/jsendx"
	"github.com/tecnickcom/nurago/pkg/logutil"
)

type bootstrapFunc func(bindFn bootstrap.BindFunc, opts ...bootstrap.Option) error

// New builds the root CLI command and wires the full service startup flow,
// turning configuration, logging, metrics, and graceful shutdown into a single
// entry point:
//
//   - Loads configuration from files and environment variables with CLI
//     overrides for log format and level.
//   - Initializes structured logging with program metadata for debugging and
//     release correlation.
//   - Creates and registers a metrics client used by HTTP and DB layers.
//   - Delegates lifecycle orchestration to bootstrap, including shutdown
//     timeout, wait group coordination, and shared stop signal.
func New(version, release string, bootstrapFn bootstrapFunc) (*cobra.Command, error) {
	var (
		argConfigDir string
		argLogFormat string
		argLogLevel  string
		rootCmd      = &cobra.Command{
			Use:   AppName,
			Short: appShortDesc,
			Long:  appLongDesc,
		}
	)

	// command-line arguments
	rootCmd.Flags().StringVarP(&argConfigDir, "configDir", "c", "", "Configuration directory to be added on top of the search list")
	rootCmd.Flags().StringVarP(&argLogFormat, "logFormat", "f", "", "Logging format: CONSOLE, JSON")
	rootCmd.Flags().StringVarP(&argLogLevel, "logLevel", "o", "", "Log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG")

	rootCmd.RunE = func(_ *cobra.Command, _ []string) error {
		// Read CLI configuration
		cfg := &appConfig{}

		err := config.Load(AppName, argConfigDir, appEnvPrefix, cfg)
		if err != nil {
			return fmt.Errorf("failed loading config: %w", err)
		}

		// Configure logger, applying the optional CLI overrides.
		logcfg, err := newLogConfig(cfg, version, release, argLogFormat, argLogLevel)
		if err != nil {
			return err
		}

		appInfo := &jsendx.AppInfo{
			ProgramName:    AppName,
			ProgramVersion: version,
			ProgramRelease: release,
		}

		// Initialize metrics

		mtr := metrics.New()

		// Wait group used for graceful shutdown of all dependants (e.g.: servers).
		wg := &sync.WaitGroup{}

		// Channel used to signal the shutdown process to all dependants.
		sc := make(chan struct{})

		// Bootstrap application
		return bootstrapFn(
			bind(cfg, appInfo, mtr, wg, sc),
			bootstrap.WithLogConfig(logcfg),
			bootstrap.WithCreateMetricsClientFunc(mtr.CreateMetricsClientFunc),
			bootstrap.WithShutdownTimeout(time.Duration(cfg.ShutdownTimeout)*time.Second),
			bootstrap.WithShutdownWaitGroup(wg),
			bootstrap.WithShutdownSignalChan(sc),
		)
	}

	// sub-command to print the version
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print this program version",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(version) //nolint:forbidigo
		},
	}

	rootCmd.AddCommand(versionCmd)

	// Parse the flags early so invalid command-line arguments are reported by
	// New (exit code 1) instead of at execution time. pflag returns ErrHelp
	// for -h/--help because cobra registers its default help flag only inside
	// Execute; it is not an error here and Execute prints the help text.
	err := rootCmd.ParseFlags(os.Args[1:])
	if err != nil && !errors.Is(err, pflag.ErrHelp) {
		return nil, fmt.Errorf("failed parsing command-line arguments: %w", err)
	}

	return rootCmd, nil
}

// newLogConfig builds the logger configuration from the loaded config, applying
// the optional command-line overrides for log format and level and tagging every
// record with the program name, version, and release for release correlation.
func newLogConfig(cfg *appConfig, version, release, argLogFormat, argLogLevel string) (*logutil.Config, error) {
	if argLogFormat != "" {
		cfg.Log.Format = argLogFormat
	}

	logFormat, err := logutil.ParseFormat(cfg.Log.Format)
	if err != nil {
		return nil, fmt.Errorf("log config error: %w", err)
	}

	if argLogLevel != "" {
		cfg.Log.Level = argLogLevel
	}

	logLevel, err := logutil.ParseLevel(cfg.Log.Level)
	if err != nil {
		return nil, fmt.Errorf("log config error: %w", err)
	}

	logattr := []logutil.Attr{
		slog.String("program", AppName),
		slog.String("version", version),
		slog.String("release", release),
	}

	// logFormat and logLevel were already parsed and validated above and the
	// other options are static, so NewConfig cannot fail here; the error is
	// intentionally discarded.
	logcfg, _ := logutil.NewConfig(
		logutil.WithOutWriter(os.Stderr),
		logutil.WithFormat(logFormat),
		logutil.WithLevel(logLevel),
		logutil.WithCommonAttr(logattr...),
	)

	return logcfg, nil
}
