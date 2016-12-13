package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func cli() (*cobra.Command, error) {

	// parse the configDir argument
	cfgCmd := new(cobra.Command)
	cfgCmd.Flags().StringVarP(&configDir, "configDir", "c", "", "Configuration directory to be added on top of the search list")
	cfgCmd.ParseFlags(os.Args)

	// configuration parameters
	cfgParams, err := getConfigParams()
	if err != nil {
		return nil, err
	}

	// overwrites the configuration parameters with the ones specified in the command line (if any)
	appParams = &cfgParams
	rootCmd := new(cobra.Command)
	rootCmd.Flags().StringVarP(&configDir, "configDir", "c", "", "Configuration directory to be added on top of the search list")
	rootCmd.Flags().StringVarP(&appParams.log.Level, "logLevel", "o", cfgParams.log.Level, "Log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG")

	rootCmd.Use = "~#PROJECT#~"
	rootCmd.Short = "~#SHORTDESCRIPTION#~"
	rootCmd.Long = `~#PROJECT#~ - ~#SHORTDESCRIPTION#~`
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {
		// check values
		err := checkParams(appParams)
		if err != nil {
			return err
		}

		// initialize StatsD client (ignore errors)
		initStats(appParams.stats)
		defer stats.Close()

		// start the HTTP server
		return startServer(appParams.serverAddress)
	}

	// sub-command to print the version
	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "print this program version",
		Long:  `print this program version`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(ProgramVersion)
		},
	}
	rootCmd.AddCommand(versionCmd)

	return rootCmd, nil
}
