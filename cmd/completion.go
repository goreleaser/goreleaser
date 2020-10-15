package cmd

import (
	"github.com/spf13/cobra"
)

type completionCmd struct {
	cmd *cobra.Command
}

func newCompletionCmd() *completionCmd {
	var root = &completionCmd{}
	var cmd = &cobra.Command{
		Use:          "completion",
		Short:        "Print shell autocompletion scripts for goreleaser for bash and zsh",
		SilenceUsage: true,
		ValidArgs:    []string{"bash", "zsh"},
		Args:         cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			switch args[0] {
			case "bash":
				err = cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				err = cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			}

			return err
		},
	}

	root.cmd = cmd
	return root
}
