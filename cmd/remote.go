package cmd

import (
	"fmt"
	"github.com/charmbracelet/log"
	"github.com/planta7/serve/internal/remote"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var rRequest = &remote.ServerRequest{}

var remoteCmd = &cobra.Command{
	Use:     "remote port",
	Aliases: []string{"r"},
	Short:   "Expose local server through localtunnel",
	Args:    cobra.MaximumNArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		var parsedFlags []string
		cmd.Flags().Visit(func(f *pflag.Flag) {
			parsedFlags = append(parsedFlags, fmt.Sprintf("%s:%s", f.Name, f.Value.String()))
		})

		log.Debug("Parameters", "args", args, "flags", parsedFlags)

		server := remote.NewServer(*rRequest)
		server.Start()
	},
}

func init() {
	rootCmd.AddCommand(remoteCmd)
	remoteCmd.Flags().IntVarP(&rRequest.Port, "port", "p", 0, "Port to expose")
	_ = remoteCmd.MarkFlagRequired("port")
}
