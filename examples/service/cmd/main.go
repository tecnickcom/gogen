// Package main is an example gogen service.
package main

import (
	"log/slog"
	"os"

	"github.com/gogenexampleowner/gogenexample/internal/cli"
	"github.com/tecnickcom/gogen/pkg/bootstrap"
)

var (
	// programVersion contains the version of the application injected at compile time.
	programVersion = "0.0.0" //nolint:gochecknoglobals

	// programRelease contains the release of the application injected at compile time.
	programRelease = "0" //nolint:gochecknoglobals
)

func main() {
	// _, _ = logging.NewDefaultLogger(cli.AppName, programVersion, programRelease, "json", "debug")
	l := slog.Default()

	rootCmd, err := cli.New(programVersion, programRelease, bootstrap.Bootstrap)
	if err != nil {
		l.With(slog.Any("error", err)).Error("UNABLE TO START THE PROGRAM")
		os.Exit(1)
	}

	// execute the root command and log errors (if any)
	err = rootCmd.Execute()
	if err != nil {
		l.With(slog.Any("error", err)).Error("UNABLE TO RUN THE COMMAND")
		os.Exit(2)
	}
}
