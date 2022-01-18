package cmd

import (
	"compress/gzip"
	"fmt"
	"os"
	"path/filepath"

	"github.com/muesli/mango/mcobra"
	"github.com/muesli/roff"
	"github.com/spf13/cobra"
)

type manCmd struct {
	cmd    *cobra.Command
	output string
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
			_ = os.RemoveAll(root.output)
			if err := os.MkdirAll(root.output, 0o755); err != nil {
				return err
			}
			root.cmd.Root().DisableAutoGenTag = true
			manPage, err := mcobra.NewManPageFromCobra(1, root.cmd.Root())
			if err != nil {
				return err
			}

			f, err := os.Create(filepath.Join(root.output, "goreleaser.1.gz"))
			if err != nil {
				return err
			}
			defer f.Close()

			w := gzip.NewWriter(f)
			defer w.Close()
			_, err = fmt.Fprint(w, manPage.Build(roff.NewDocument()))
			if err != nil {
				return err
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVarP(&root.output, "output", "o", "manpages", "output directory")
	root.cmd = cmd
	return root
}
