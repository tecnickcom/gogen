package main

import (
	"github.com/nexmoinc/gosrvlib-sample-service/internal/cli"
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
)

var (
	ProgramVersion = "0.0.0"
	ProgramRelease = "0"
)

func main() {
	_, _ = logging.NewDefaultLogger(cli.AppName, ProgramVersion, ProgramRelease, "json", "debug")

	rootCmd, err := cli.New(ProgramVersion, ProgramRelease)
	if err != nil {
		zap.L().Fatal("UNABLE TO START THE PROGRAM", zap.Error(err))
		return
	}
	// execute the root command and log errors (if any)
	if err = rootCmd.Execute(); err != nil {
		zap.L().Fatal("UNABLE TO RUN THE COMMAND", zap.Error(err))
	}
}
