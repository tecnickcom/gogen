// Package cli contains the CLI entry point.
package cli

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/Vonage/gosrvlib/pkg/bootstrap"
	"github.com/Vonage/gosrvlib/pkg/config"
	"github.com/Vonage/gosrvlib/pkg/httputil/jsendx"
	"github.com/Vonage/gosrvlib/pkg/logging"
	"github.com/gosrvlibexampleowner/gosrvlibexample/internal/metrics"
	"github.com/spf13/cobra"
)

type bootstrapFunc func(bindFn bootstrap.BindFunc, opts ...bootstrap.Option) error

// New creates an new CLI instance.
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

	rootCmd.Flags().StringVarP(&argConfigDir, "configDir", "c", "", "Configuration directory to be added on top of the search list")
	rootCmd.Flags().StringVarP(&argLogFormat, "logFormat", "f", "", "Logging format: CONSOLE, JSON")
	rootCmd.Flags().StringVarP(&argLogLevel, "logLevel", "o", "", "Log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG")

	rootCmd.RunE = func(_ *cobra.Command, _ []string) error {
		// Read CLI configuration
		cfg := &appConfig{}
		if err := config.Load(AppName, argConfigDir, appEnvPrefix, cfg); err != nil {
			return fmt.Errorf("failed loading config: %w", err)
		}

		if argLogFormat != "" {
			cfg.Log.Format = argLogFormat
		}

		if argLogLevel != "" {
			cfg.Log.Level = argLogLevel
		}

		// Configure logger
		l, err := logging.NewDefaultLogger(AppName, version, release, cfg.Log.Format, cfg.Log.Level)
		if err != nil {
			return fmt.Errorf("failed configuring logger: %w", err)
		}

		appInfo := &jsendx.AppInfo{
			ProgramName:    AppName,
			ProgramVersion: version,
			ProgramRelease: release,
		}

		// Confifure metrics
		mtr := metrics.New()

		// Wait group used for graceful shutdown of all dependants (e.g.: servers).
		wg := &sync.WaitGroup{}

		// Channel used to signal the shutdown process to all dependants.
		sc := make(chan struct{})

		// Boostrap application
		return bootstrapFn(
			bind(cfg, appInfo, mtr, wg, sc),
			bootstrap.WithLogger(l),
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
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version) //nolint:forbidigo
		},
	}

	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.ParseFlags(os.Args); err != nil {
		return nil, fmt.Errorf("failed parsing comman-line arguments: %w", err)
	}

	return rootCmd, nil
}
