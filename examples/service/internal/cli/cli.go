// Package cli contains the CLI entry point.
package cli

import (
	"fmt"
	"os"

	"github.com/gosrvlibexampleowner/gosrvlibexample/internal/metrics"
	"github.com/nexmoinc/gosrvlib/pkg/bootstrap"
	"github.com/nexmoinc/gosrvlib/pkg/config"
	"github.com/nexmoinc/gosrvlib/pkg/httputil/jsendx"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
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

		// Boostrap application
		return bootstrapFn(
			bind(cfg, appInfo, mtr),
			bootstrap.WithLogger(l),
			bootstrap.WithCreateMetricsClientFunc(mtr.CreateMetricsClientFunc),
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
