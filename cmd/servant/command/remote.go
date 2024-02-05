package command

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/servant/internal/server"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"strconv"
)

var rConfig = &server.Configuration{}

var remoteCmd = &cobra.Command{
	Use:     "remote port",
	Aliases: []string{"r"},
	Short:   "Expose local server through localtunnel",
	Args:    cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		rConfig.Type = server.TypeRemote

		var parsedFlags []string
		cmd.Flags().Visit(func(f *pflag.Flag) {
			if f.Name == "disable-tui" {
				rConfig.DisableTUI, _ = strconv.ParseBool(f.Value.String())
			}
			parsedFlags = append(parsedFlags, fmt.Sprintf("%s:%s", f.Name, f.Value.String()))
		})

		log.Debug("Parameters", "args", args, "flags", parsedFlags)

		servant := server.New(*rConfig)
		servant.Start()
	},
}

func init() {
	rootCmd.AddCommand(remoteCmd)
	remoteCmd.Flags().StringVarP(&rConfig.Subdomain, "subdomain", "s", "", "Subdomain (default is random)")
	remoteCmd.Flags().IntVarP(&rConfig.Port, "port", "p", 0, "Port to expose")
	_ = remoteCmd.MarkFlagRequired("port")
}
