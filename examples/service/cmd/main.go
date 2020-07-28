package main

import (
	"github.com/nexmoinc/gosrvlib/pkg/logging"
	"go.uber.org/zap"
)

var (
	Name    = "srvxmplname"
	Version = "0.0.0"
	Release = "0"

	appEnvPrefix = "SRVXMPLENVPREFIX"
	appShortDesc = "srvxmplshortdesc"
	appLongDesc  = "srvxmpllongdesc"
)

func main() {
	_, _ = logging.NewDefaultLogger(Name, Version, Release, "json", "debug")

	rootCmd, err := cli()
	if err != nil {
		zap.L().Fatal("UNABLE TO START THE PROGRAM", zap.Error(err))
		return
	}
	// execute the root command and log errors (if any)
	if err = rootCmd.Execute(); err != nil {
		zap.L().Fatal("UNABLE TO RUN THE COMMAND", zap.Error(err))
	}
}
