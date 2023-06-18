package cmd

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/serve/internal/local"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var request = &local.ServerRequest{}

var localCmd = &cobra.Command{
	Use:     "local [path]",
	Aliases: []string{"l"},
	Short:   "Start a local HTTP server",
	Args:    cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		request.Path = "./"

		var parsedFlags []string
		cmd.Flags().Visit(func(f *pflag.Flag) {
			parsedFlags = append(parsedFlags, fmt.Sprintf("%s:%s", f.Name, f.Value.String()))
		})

		log.Debug("Parameters", "args", args, "flags", parsedFlags)
		if len(args) > 0 {
			request.Path = args[0]
		}

		server := local.NewServer(*request)
		server.Start()
	},
}

func init() {
	rootCmd.AddCommand(localCmd)
	localCmd.Flags().StringVarP(&request.Host, "host", "", "", "Server host (default is empty)")
	localCmd.Flags().IntVarP(&request.Port, "port", "p", 0, "Listen on port (default is random)")
	localCmd.Flags().BoolVarP(&request.CORS, "cors", "c", false, "Enable CORS (default is false)")
	localCmd.Flags().BoolVarP(&request.Launch, "launch", "l", false, "Launch default browser (default is false)")
	localCmd.Flags().StringVarP(&request.Auth, "auth", "", "", "username:password for basic auth (default is empty)")
	localCmd.Flags().BoolVarP(&request.TLS.Auto, "auto-tls", "", false, "Start with embedded certificate (default is false)")
	localCmd.Flags().StringVarP(&request.TLS.CertFile, "cert-file", "", "", "Path to certificate (default is empty)")
	localCmd.Flags().StringVarP(&request.TLS.KeyFile, "key-file", "", "", "Path to key")
	localCmd.Flags().BoolVarP(&request.TUI, "tui", "", false, "Launch with TUI (experimental)")
	localCmd.MarkFlagsRequiredTogether("cert-file", "key-file")
	localCmd.MarkFlagsMutuallyExclusive("auto-tls", "cert-file")
}
