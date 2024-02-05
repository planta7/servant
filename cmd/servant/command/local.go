package command

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/servant/internal/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strconv"
)

var lConfig = &server.Configuration{}

var localCmd = &cobra.Command{
	Use:     "local [path]",
	Aliases: []string{"l"},
	Short:   "Start a local HTTP server",
	Args:    cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			lConfig.Path = args[0]
		}
		lConfig.Type = server.TypeLocal
		lConfig.Path = "./"

		var parsedFlags []string
		cmd.Flags().Visit(func(f *pflag.Flag) {
			if f.Name == "disable-tui" {
				lConfig.DisableTUI, _ = strconv.ParseBool(f.Value.String())
			}
			parsedFlags = append(parsedFlags, fmt.Sprintf("%s:%s", f.Name, f.Value.String()))
		})
		log.Debug("Parameters", "args", args, "flags", parsedFlags)

		servant := server.New(*lConfig)
		servant.Start()
	},
}

func init() {
	rootCmd.AddCommand(localCmd)
	localCmd.Flags().StringVarP(&lConfig.Host, "host", "", "", "Server host (default is empty)")
	localCmd.Flags().IntVarP(&lConfig.Port, "port", "p", 0, "Listen on port (default is random)")
	localCmd.Flags().BoolVarP(&lConfig.Expose, "expose", "e", false, "Expose through localtunnel (default is false)")
	localCmd.Flags().StringVarP(&lConfig.Subdomain, "subdomain", "s", "", "Subdomain (default is random)")
	localCmd.Flags().BoolVarP(&lConfig.CORS, "cors", "c", false, "Enable CORS (default is false)")
	localCmd.Flags().BoolVarP(&lConfig.Launch, "launch", "l", false, "Launch default browser (default is false)")
	localCmd.Flags().StringVarP(&lConfig.Auth, "auth", "", "", "username:password for basic auth (default is empty)")
	localCmd.Flags().BoolVarP(&lConfig.TLS.Auto, "auto-tls", "", false, "Start with embedded certificate (default is false)")
	localCmd.Flags().StringVarP(&lConfig.TLS.CertFile, "cert-file", "", "", "Path to certificate (default is empty)")
	localCmd.Flags().StringVarP(&lConfig.TLS.KeyFile, "key-file", "", "", "Path to key")
	localCmd.MarkFlagsRequiredTogether("cert-file", "key-file")
	localCmd.MarkFlagsMutuallyExclusive("auto-tls", "cert-file")
	localCmd.MarkFlagsMutuallyExclusive("expose", "cors", "auto-tls", "cert-file", "key-file")
}
