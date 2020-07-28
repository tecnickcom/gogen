package main

import (
	"context"
	"fmt"
	"os"

	"github.com/nexmoinc/gosrvlib-sample-service/internal/httphandler"
	"github.com/nexmoinc/gosrvlib/pkg/bootstrap"
	"github.com/nexmoinc/gosrvlib/pkg/config"
	"github.com/nexmoinc/gosrvlib/pkg/httpserver"
	"github.com/nexmoinc/gosrvlib/pkg/httputil/jsendx"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"github.com/nexmoinc/gosrvlib/pkg/uid"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

// NOTE: This is the config struct to be customized
type appConfig struct {
	config.BaseConfig `mapstructure:",squash"`
	Enabled           bool   `mapstructure:"enabled"`
	MonitoringAddress string `mapstructure:"monitoring_address"`
	ServerAddress     string `mapstructure:"server_address"`
}

func (c *appConfig) SetDefaults(v config.Viper) {
	// v.SetDefault("enabled", true)
	//
	// // NOTE: Set the default monitoring_address port to the same as service_port to start a single HTTP server
	// v.SetDefault("monitoring_address", ":8082")
	// v.SetDefault("server_address", ":8081")

	// NOTE: Set other configuration defaults here
	// v.SetDefault("db.dsn", "<DSN>")
}

func (c *appConfig) Validate() error {
	// NOTE: Implement validation for configuration here
	return nil
}

func cli() (*cobra.Command, error) {
	var argConfigDir string
	var argLogFormat string
	var argLogLevel string
	var rootCmd = &cobra.Command{
		Use:   Name,
		Short: appShortDesc,
		Long:  appLongDesc,
	}

	rootCmd.Flags().StringVarP(&argConfigDir, "configDir", "c", "", "Configuration directory to be added on top of the search list")
	rootCmd.Flags().StringVarP(&argLogFormat, "logFormat", "f", "", "Logging format: CONSOLE, JSON")
	rootCmd.Flags().StringVarP(&argLogLevel, "logLevel", "o", "", "Log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG")

	rootCmd.RunE = func(_ *cobra.Command, _ []string) error {
		// initialize seed for random ID generator
		if err := uid.InitRandSeed(); err != nil {
			return fmt.Errorf("failed initializing random seeder: %w", err)
		}

		appInfo := &jsendx.AppInfo{
			ProgramName:    Name,
			ProgramVersion: Version,
			ProgramRelease: Release,
		}

		// Read CLI configuration
		cfg := &appConfig{}
		if err := config.Load(Name, argConfigDir, appEnvPrefix, cfg); err != nil {
			return fmt.Errorf("failed loading config: %w", err)
		}

		if argLogFormat != "" {
			cfg.Log.Format = argLogFormat
		}

		if argLogLevel != "" {
			cfg.Log.Level = argLogLevel
		}

		// Configure logger
		l, err := logging.NewDefaultLogger(Name, Version, Release, cfg.Log.Format, cfg.Log.Level)
		if err != nil {
			return fmt.Errorf("failed configuring logger: %w", err)
		}

		// Boostrap application
		return bootstrap.Bootstrap(bind(cfg, appInfo), bootstrap.WithLogger(l))
	}

	// sub-command to print the version
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print this program version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(Version)
		},
	}
	rootCmd.AddCommand(versionCmd)

	if err := rootCmd.ParseFlags(os.Args); err != nil {
		return nil, err
	}
	return rootCmd, nil
}

func bind(cfg *appConfig, appInfo *jsendx.AppInfo) bootstrap.BindFunc {
	return func(ctx context.Context, l *zap.Logger) error {
		// NOTE: Add initialization and wiring of external components and service here
		//
		// <INIT CODE HERE>
		//

		// NOTE: Uncomment this line if only default HTTP routes exist
		// hh := httpserver.NopBinder()
		hh := httphandler.New(nil)

		// NOTE: Uncomment to create a custom healthcheck handler
		// healthCheckHandler := healthcheck.Handler(healthcheck.HealthCheckerMap{
		// 	"<extCompName>": extCompInstance,
		// }, appInfo)

		defaultServerOpts := []httpserver.Option{
			// NOTE: Uncomment to use the JSendX router for 404, 405 and panic handlers
			// httpserver.WithRouter(jsendx.NewRouter(appInfo)),

			httpserver.WithRoutesIndexHandlerFunc(jsendx.DefaultRoutesIndexHandler(appInfo)),
			httpserver.WithPingHandlerFunc(jsendx.DefaultPingHandler(appInfo)),

			// NOTE: Uncomment this line to enable custom health check reporting
			// httpserver.WithStatusHandlerFunc(healthCheckHandler),
			httpserver.WithStatusHandlerFunc(jsendx.DefaultStatusHandler(appInfo)),
		}

		// Use a separate server for monitoring routes if monitor_address and server_address are different
		if cfg.MonitoringAddress != cfg.ServerAddress {
			httpMonitoringOpts := append(defaultServerOpts, []httpserver.Option{
				httpserver.WithServerAddr(cfg.MonitoringAddress),
			}...)
			if err := httpserver.Start(ctx, httpserver.NopBinder(), httpMonitoringOpts...); err != nil {
				l.Fatal("error starting monitoring HTTP server", zap.Error(err))
			}
		}

		httpServiceOpts := []httpserver.Option{
			httpserver.WithServerAddr(cfg.ServerAddress),
		}

		// Disable default routes if we are starting the monitoring routes on a separate server instance
		if cfg.MonitoringAddress != cfg.ServerAddress {
			httpServiceOpts = append(defaultServerOpts, httpServiceOpts...)
			httpServiceOpts = append(httpServiceOpts, []httpserver.Option{
				httpserver.WithDisableDefaultRoutes(),
			}...)
		}

		if err := httpserver.Start(ctx, hh, httpServiceOpts...); err != nil {
			l.Fatal("error starting service HTTP server", zap.Error(err))
		}

		return nil
	}
}
