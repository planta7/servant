package cmd

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/serve/internal/local"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var lRequest = &local.ServerRequest{}

var localCmd = &cobra.Command{
	Use:     "local [path]",
	Aliases: []string{"l"},
	Short:   "Start a local HTTP server",
	Args:    cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		lRequest.Path = "./"

		var parsedFlags []string
		cmd.Flags().Visit(func(f *pflag.Flag) {
			parsedFlags = append(parsedFlags, fmt.Sprintf("%s:%s", f.Name, f.Value.String()))
		})

		log.Debug("Parameters", "args", args, "flags", parsedFlags)
		if len(args) > 0 {
			lRequest.Path = args[0]
		}

		server := local.NewServer(*lRequest)
		server.Start()
	},
}

func init() {
	rootCmd.AddCommand(localCmd)
	localCmd.Flags().StringVarP(&lRequest.Host, "host", "", "", "Server host (default is empty)")
	localCmd.Flags().IntVarP(&lRequest.Port, "port", "p", 0, "Listen on port (default is random)")
	localCmd.Flags().BoolVarP(&lRequest.Expose, "expose", "e", false, "Expose through localtunnel (default is false)")
	localCmd.Flags().BoolVarP(&lRequest.CORS, "cors", "c", false, "Enable CORS (default is false)")
	localCmd.Flags().BoolVarP(&lRequest.Launch, "launch", "l", false, "Launch default browser (default is false)")
	localCmd.Flags().StringVarP(&lRequest.Auth, "auth", "", "", "username:password for basic auth (default is empty)")
	localCmd.Flags().BoolVarP(&lRequest.TLS.Auto, "auto-tls", "", false, "Start with embedded certificate (default is false)")
	localCmd.Flags().StringVarP(&lRequest.TLS.CertFile, "cert-file", "", "", "Path to certificate (default is empty)")
	localCmd.Flags().StringVarP(&lRequest.TLS.KeyFile, "key-file", "", "", "Path to key")
	localCmd.Flags().BoolVarP(&lRequest.DisableTUI, "disable-tui", "", false, "Disable TUI")
	localCmd.MarkFlagsRequiredTogether("cert-file", "key-file")
	localCmd.MarkFlagsMutuallyExclusive("auto-tls", "cert-file")
	localCmd.MarkFlagsMutuallyExclusive("expose", "cors", "auto-tls", "cert-file", "key-file")
}
