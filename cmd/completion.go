package cmd

import "github.com/spf13/cobra"

type completionCmd struct {
	cmd *cobra.Command
}

func newCompletionCmd() *completionCmd {
	var root = &completionCmd{}
	var cmd = &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Print shell autocompletion scripts for goreleaser",
		Long: `To load completions:

Bash:

$ source <(goreleaser completion bash)

# To load completions for each session, execute once:
Linux:
  $ goreleaser completion bash > /etc/bash_completion.d/goreleaser
MacOS:
  $ goreleaser completion bash > /usr/local/etc/bash_completion.d/goreleaser

Zsh:

# If shell completion is not already enabled in your environment you will need
# to enable it.  You can execute the following once:

$ echo "autoload -U compinit; compinit" >> ~/.zshrc

# To load completions for each session, execute once:
$ goreleaser completion zsh > "${fpath[1]}/_goreleaser"

# You will need to start a new shell for this setup to take effect.

Fish:

$ goreleaser completion fish | source

# To load completions for each session, execute once:
$ goreleaser completion fish > ~/.config/fish/completions/goreleaser.fish
`,
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish"},
		Args:                  cobra.ExactValidArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			switch args[0] {
			case "bash":
				err = cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				err = cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				err = cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			}

			return err
		},
	}

	root.cmd = cmd
	return root
}
