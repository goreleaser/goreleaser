package cmd

import (
	"fmt"

	"github.com/apex/log"
	"github.com/spf13/cobra"
)

func Execute(version string, exit func(int), args []string) {
	fmt.Println()
	defer fmt.Println()
	NewRootCmd(version, exit).Execute(args)
}

func (cmd *rootCmd) Execute(args []string) {
	cmd.cmd.SetArgs(args)
	if len(args) != 1 {
		// defaults to the release command
		cmd.cmd.SetArgs(append([]string{"release"}, args...))
	}

	if err := cmd.cmd.Execute(); err != nil {
		var code = 1
		var msg = "command fail"
		if eerr, ok := err.(*exitError); ok {
			code = eerr.code
			if eerr.details != "" {
				msg = eerr.details
			}
		}
		log.WithError(err).Error(msg)
		cmd.exit(code)
	}
}

type rootCmd struct {
	cmd   *cobra.Command
	debug bool
	exit  func(int)
}

func NewRootCmd(version string, exit func(int)) *rootCmd {
	var root = &rootCmd{
		exit: exit,
	}
	var cmd = &cobra.Command{
		Use:           "goreleaser",
		Short:         "Deliver Go binaries as fast and easily as possible",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if root.debug {
				log.SetLevel(log.DebugLevel)
				log.Debug("debug logs enabled")
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
