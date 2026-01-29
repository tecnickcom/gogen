// Package main is an example gogen service.
package main

import (
	"log/slog"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gogenexampleowner/gogenexample/internal/cli"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/tecnickcom/gogen/pkg/bootstrap"
	"github.com/tecnickcom/gogen/pkg/logsrv"
	"github.com/tecnickcom/gogen/pkg/logutil"
)

var (
	// programVersion contains the version of the application injected at compile time.
	programVersion = "0.0.0" //nolint:gochecknoglobals

	// programRelease contains the release of the application injected at compile time.
	programRelease = "0" //nolint:gochecknoglobals
)

// exitFn define tha exit function and can be overwritten for testing.
var exitFn = os.Exit //nolint:gochecknoglobals

func main() {
	// set default logger
	logattr := []logutil.Attr{
		slog.String("program", cli.AppName),
		slog.String("version", programVersion),
		slog.String("release", programRelease),
	}
	logcfg, _ := logutil.NewConfig(
		logutil.WithOutWriter(os.Stderr),
		logutil.WithFormat(logutil.FormatJSON),
		logutil.WithLevel(logutil.LevelDebug),
		logutil.WithCommonAttr(logattr...),
	)
	l := logsrv.NewLogger(logcfg)

	rootCmd, err := cli.New(programVersion, programRelease, bootstrap.Bootstrap)
	if err != nil {
		l.With(slog.Any("error", err)).Error("UNABLE TO START THE PROGRAM")
		exitFn(1)
	}

	// execute the root command and log errors (if any)
	err = rootCmd.Execute()
	if err != nil {
		l.With(slog.Any("error", err)).Error("UNABLE TO RUN THE COMMAND")
		exitFn(2)
	}
}
