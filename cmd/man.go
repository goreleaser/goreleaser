package cmd

import (
	"fmt"
	"os"

	"github.com/muesli/mango/mcobra"
	"github.com/muesli/roff"
	"github.com/spf13/cobra"
)

type manCmd struct {
	cmd *cobra.Command
}

func newManCmd() *manCmd {
	root := &manCmd{}
	cmd := &cobra.Command{
		Use:                   "man",
		Short:                 "Generates GoReleaser's command line manpages",
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Hidden:                true,
		Args:                  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			manPage, err := mcobra.NewManPageFromCobra(1, root.cmd.Root())
			if err != nil {
				return err
			}

			_, err = fmt.Fprint(os.Stdout, manPage.Build(roff.NewDocument()))
			return err
		},
	}

	root.cmd = cmd
	return root
}
