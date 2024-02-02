// MIT Licensed
// Copyright (c) 2023 Roberto García <roberto@planta7.io>

package command

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/servant/internal"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
	"strings"
)

const (
	EnvPrefix = "SERVANT"
)

var (
	cfgFile    string
	verbose    bool
	disableTUI bool
)

var rootCmd = &cobra.Command{
	Use:   "servant",
	Short: "Create an HTTP server in a jiffy",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		log.Info(fmt.Sprintf("servant %s (%s)",
			internal.ServantInfo.Version,
			internal.ServantInfo.GetShortCommit()))
		bindFlags(cmd)
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
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./servant and $HOME/.servant)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose mode (default is false)")
	rootCmd.PersistentFlags().BoolVarP(&disableTUI, "disable-tui", "", false, "Disable TUI (default is false)")
	viper.SetDefault("disable-tui", false)
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
		viper.SetConfigName(".servant")
	}

	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		log.Info("Using config file", "config", viper.ConfigFileUsed())
	}
}

func bindFlags(cmd *cobra.Command) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed && viper.IsSet(f.Name) {
			val := viper.Get(f.Name)
			_ = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}
