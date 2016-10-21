package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func cli() (*cobra.Command, error) {

	// configuration parameters
	cfgParams, err := getConfigParams()
	if err != nil {
		return nil, err
	}

	// set the root command
	rootCmd := new(cobra.Command)

	// overwrites the configuration parameters with the ones specified in the command line (if any)
	rootCmd.Flags().StringVarP(&appParams.serverAddress, "serverAddress", "u", cfgParams.serverAddress, "HTTP API address (ip:port) or just (:port)")
	rootCmd.Flags().StringVarP(&appParams.statsPrefix, "statsPrefix", "p", cfgParams.statsPrefix, "StatsD bucket prefix name")
	rootCmd.Flags().StringVarP(&appParams.statsNetwork, "statsNetwork", "k", cfgParams.statsNetwork, "StatsD client network type (udp or tcp)")
	rootCmd.Flags().StringVarP(&appParams.statsAddress, "statsAddress", "m", cfgParams.statsAddress, "StatsD daemon address (ip:port) or just (:port)")
	rootCmd.Flags().IntVarP(&appParams.statsFlushPeriod, "statsFlushPeriod", "r", cfgParams.statsFlushPeriod, "StatsD client flush period in milliseconds")
	rootCmd.Flags().StringVarP(&appParams.logLevel, "logLevel", "o", cfgParams.logLevel, "Log level: panic, fatal, error, warning, info, debug")

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
		initStats(appParams)
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
