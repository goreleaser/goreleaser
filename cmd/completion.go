package cmd

import "github.com/spf13/cobra"

type completionCmd struct {
	cmd *cobra.Command
}

func newCompletionCmd() *completionCmd {
	root := &completionCmd{}
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish]",
		Short: "Prints shell autocompletion scripts for GoReleaser",
		Long: `Allows you to setup your shell to autocomple GoReleaser commands and flags.

#### Bash

	$ source <(goreleaser completion bash)

To load completions for each session, execute once:

##### Linux

	$ goreleaser completion bash > /etc/bash_completion.d/goreleaser

##### MacOS

	$ goreleaser completion bash > /usr/local/etc/bash_completion.d/goreleaser

#### ZSH

If shell completion is not already enabled in your environment you will need to enable it.
You can execute the following once:

	$ echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions for each session, execute once:

	$ goreleaser completion zsh > "${fpath[1]}/_goreleaser"

You will need to start a new shell for this setup to take effect.

#### Fish

	$ goreleaser completion fish | source

To load completions for each session, execute once:

	$ goreleaser completion fish > ~/.config/fish/completions/goreleaser.fish

**NOTE**: If you are using an official GoReleaser package, it should setup autocompletions for you out of the box.
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
