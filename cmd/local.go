package cmd

import (
	"github.com/charmbracelet/log"
	"serve/internal/local"

	"github.com/spf13/cobra"
)

var config = &local.Configuration{}

var localCmd = &cobra.Command{
	Use:   "local",
	Short: "Start local HTTP server",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		config.Path = "./"
		log.Debug("Arguments", "args", args)
		if len(args) > 0 {
			config.Path = args[0]
		}
		server := local.NewServer(*config)
		server.Start()
	},
}

func init() {
	rootCmd.AddCommand(localCmd)
	localCmd.Flags().StringVarP(&config.Host, "host", "", "", "Server host (default is \"\")")
	localCmd.Flags().IntVarP(&config.Port, "port", "p", 0, "Listen on port (default is random)")
	localCmd.Flags().BoolVarP(&config.Launch, "launch", "l", false, "Launch default browser")
}
