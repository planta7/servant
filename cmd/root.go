package cmd

import (
	"github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"os"
)

var cfgFile string
var verbose bool

var rootCmd = &cobra.Command{
	Use:   "serve",
	Short: "Create an HTTP server in a jiffy",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if verbose {
			log.SetLevel(log.DebugLevel)
			log.SetReportCaller(true)
		}
	},
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./serve.yaml and $HOME/.serve.yaml)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "verbose mode (default is false)")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		viper.AddConfigPath(".")
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".serve")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Info("Using config file", "config", viper.ConfigFileUsed())
	}
}
