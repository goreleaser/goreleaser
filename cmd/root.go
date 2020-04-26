package cmd

import (
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/spf13/cobra"
)

func Execute(version string) {
	fmt.Println()
	defer fmt.Println()

	if err := NewRootCmd(version).Execute(); err != nil {
		log.WithError(err).Error("command failed")
		if gerr, ok := err.(*GoreleaserError); ok {
			os.Exit(gerr.Exit())
		}
		os.Exit(1)
	}
}

func NewRootCmd(version string) *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "goreleaser",
		Short:   "Deliver Go binaries as fast and easily as possible",
		Version: version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug, _ := cmd.Flags().GetBool("debug"); debug {
				log.SetLevel(log.DebugLevel)
				log.Debug("enabled debug logs")
			}
		},
	}

	cmd.PersistentFlags().Bool("debug", false, "Enable debug mode")
	cmd.AddCommand(
		NewReleaseCmd().cmd,
		NewCheckCmd().cmd,
		NewInitCmd().cmd,
	)

	return cmd
}
