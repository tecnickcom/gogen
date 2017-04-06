package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func cli() (*cobra.Command, error) {

	var logLevel string
	var quantity int

	rootCmd := new(cobra.Command)
	rootCmd.Flags().StringVarP(&configDir, "configDir", "c", "", "Configuration directory to be added on top of the search list")
	rootCmd.Flags().StringVarP(&logLevel, "logLevel", "o", "*", "Log level: EMERGENCY, ALERT, CRITICAL, ERROR, WARNING, NOTICE, INFO, DEBUG")
	rootCmd.Flags().IntVarP(&quantity, "quantity", "r", 0, "Number of results to return")
	err := rootCmd.ParseFlags(os.Args)
	if err != nil {
		return nil, err
	}

	rootCmd.Use = "~#PROJECT#~"
	rootCmd.Short = "~#SHORTDESCRIPTION#~"
	rootCmd.Long = `~#PROJECT#~ - ~#SHORTDESCRIPTION#~`
	rootCmd.RunE = func(cmd *cobra.Command, args []string) error {

		// configuration parameters
		cfgParams, err := getConfigParams()
		if err != nil {
			return err
		}
		appParams = &cfgParams
		if logLevel != "*" {
			appParams.log.Level = logLevel
		}
		if quantity > 0 {
			appParams.quantity = quantity
		}

		// check values
		err = checkParams(appParams)
		if err != nil {
			return err
		}

		// get results
		for i := 0; i < appParams.quantity; i++ {
			fmt.Printf("%s\n", getResult())
		}

		return nil
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
