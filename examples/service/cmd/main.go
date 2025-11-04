// Package main is an example gogen service.
package main

import (
	"github.com/tecnickcom/gogen/pkg/bootstrap"
	"github.com/tecnickcom/gogen/pkg/logging"
	"github.com/gogenexampleowner/gogenexample/internal/cli"
	"go.uber.org/zap"
)

var (
	// programVersion contains the version of the application injected at compile time.
	programVersion = "0.0.0" //nolint:gochecknoglobals

	// programRelease contains the release of the application injected at compile time.
	programRelease = "0" //nolint:gochecknoglobals
)

func main() {
	_, _ = logging.NewDefaultLogger(cli.AppName, programVersion, programRelease, "json", "debug")

	rootCmd, err := cli.New(programVersion, programRelease, bootstrap.Bootstrap)
	if err != nil {
		logging.LogFatal("UNABLE TO START THE PROGRAM", zap.Error(err))
		return
	}

	// execute the root command and log errors (if any)
	err = rootCmd.Execute()
	if err != nil {
		logging.LogFatal("UNABLE TO RUN THE COMMAND", zap.Error(err))
	}
}
