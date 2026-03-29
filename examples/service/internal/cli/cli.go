// Package cli contains the CLI entry point.
package cli

import (
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/gogenexampleowner/gogenexample/internal/metrics"
	"github.com/spf13/cobra"
	"github.com/tecnickcom/gogen/pkg/bootstrap"
	"github.com/tecnickcom/gogen/pkg/config"
	"github.com/tecnickcom/gogen/pkg/httputil/jsendx"
	"github.com/tecnickcom/gogen/pkg/logutil"
)

type bootstrapFunc func(bindFn bootstrap.BindFunc, opts ...bootstrap.Option) error

// New builds the root CLI command and wires the full service startup flow.
//
// It solves a common operational problem for service binaries: turning runtime
// concerns (configuration, logging, metrics, and graceful shutdown) into a
// single repeatable entry point so business features can be added without
// re-implementing process scaffolding.
//
// Top features for developers:
//   - Loads configuration from files and environment variables with CLI
//     overrides for log format and level.
//   - Initializes structured logging with program metadata for easier
//     debugging and release correlation.
//   - Creates and registers a metrics client used by HTTP and DB layers.
//   - Delegates lifecycle orchestration to bootstrap, including shutdown
//     timeout, wait group coordination, and shared stop signal.
//
// The result is a predictable startup contract that improves maintainability
// and reduces copy-paste infrastructure code in new services.
//
//nolint:gocognit
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

		// Configure logger

		if argLogFormat != "" {
			cfg.Log.Format = argLogFormat
		}

		logFormat, err := logutil.ParseFormat(cfg.Log.Format)
		if err != nil {
			return fmt.Errorf("log config error: %w", err)
		}

		if argLogLevel != "" {
			cfg.Log.Level = argLogLevel
		}

		logLevel, err := logutil.ParseLevel(cfg.Log.Level)
		if err != nil {
			return fmt.Errorf("log config error: %w", err)
		}

		logattr := []logutil.Attr{
			slog.String("program", AppName),
			slog.String("version", version),
			slog.String("release", release),
		}

		logcfg, _ := logutil.NewConfig(
			logutil.WithOutWriter(os.Stderr),
			logutil.WithFormat(logFormat),
			logutil.WithLevel(logLevel),
			logutil.WithCommonAttr(logattr...),
		)

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

		// Boostrap application
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

	err := rootCmd.ParseFlags(os.Args)
	if err != nil {
		return nil, fmt.Errorf("failed parsing comman-line arguments: %w", err)
	}

	return rootCmd, nil
}
