package cmd

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/serve/internal/local"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var config = &local.Configuration{}

var localCmd = &cobra.Command{
	Use:   "local",
	Short: "Start local HTTP server",
	Args:  cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		config.Path = "./"

		var parsedFlags []string
		cmd.Flags().Visit(func(f *pflag.Flag) {
			parsedFlags = append(parsedFlags, fmt.Sprintf("%s:%s", f.Name, f.Value.String()))
		})

		log.Debug("Parameters", "args", args, "flags", parsedFlags)
		if len(args) > 0 {
			config.Path = args[0]
		}

		server := local.NewServer(*config)
		server.Start()
	},
}

func init() {
	rootCmd.AddCommand(localCmd)
	localCmd.Flags().StringVarP(&config.Host, "host", "", "", "Server host (default is empty)")
	localCmd.Flags().IntVarP(&config.Port, "port", "p", 0, "Listen on port (default is random)")
	localCmd.Flags().BoolVarP(&config.CORS, "cors", "c", false, "Enable CORS (default is false)")
	localCmd.Flags().BoolVarP(&config.Launch, "launch", "l", false, "Launch default browser (default is empty)")
	localCmd.Flags().StringVarP(&config.TLS.CertFile, "certFile", "", "", "Path to certificate (default is empty)")
	localCmd.Flags().StringVarP(&config.TLS.KeyFile, "keyFile", "", "", "Path to key")
	localCmd.MarkFlagsRequiredTogether("certFile", "keyFile")
}
