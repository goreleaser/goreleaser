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
	NewRootCmd(version).Execute()
}

func (cmd *rootCmd) Execute() {
	var args = os.Args[1:]

	if len(args) != 1 {
		// defaults to the release command
		cmd.cmd.SetArgs(append([]string{"release"}, args...))
	}

	if err := cmd.cmd.Execute(); err != nil {
		log.WithError(err).Error("command failed")
		if gerr, ok := err.(*GoreleaserError); ok {
			os.Exit(gerr.Exit())
		}
		os.Exit(1)
	}
}

type rootCmd struct {
	cmd   *cobra.Command
	debug bool
}

func NewRootCmd(version string) *rootCmd {
	var root = &rootCmd{}
	var cmd = &cobra.Command{
		Use:           "goreleaser",
		Short:         "Deliver Go binaries as fast and easily as possible",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if root.debug {
				log.SetLevel(log.DebugLevel)
				log.Debug("enabled debug logs")
			}
		},
	}

	cmd.PersistentFlags().BoolVar(&root.debug, "debug", false, "Enable debug mode")
	cmd.AddCommand(
		NewReleaseCmd().cmd,
		NewCheckCmd().cmd,
		NewInitCmd().cmd,
	)

	root.cmd = cmd
	return root
}
