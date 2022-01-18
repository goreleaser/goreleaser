package cmd

import (
	"compress/gzip"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

type manCmd struct {
	cmd    *cobra.Command
	output string
}

func newManCmd() *manCmd {
	root := &manCmd{}
	cmd := &cobra.Command{
		Use:                   "man",
		Short:                 "Generates GoReleaser's command line man pages",
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Hidden:                true,
		Args:                  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			root.cmd.Root().DisableAutoGenTag = true
			_ = os.RemoveAll(root.output)
			if err := os.MkdirAll(root.output, 0o755); err != nil {
				return err
			}
			if err := doc.GenManTree(root.cmd.Root(), &doc.GenManHeader{
				Title:   "goreleaser",
				Section: "1",
				Source:  "https://goreleaser.com",
			}, root.output); err != nil {
				return err
			}
			return filepath.Walk(root.output, func(path string, info fs.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if info.IsDir() {
					return nil
				}
				fw, err := os.Create(path + ".gz")
				if err != nil {
					return err
				}
				defer fw.Close()
				fr, err := os.Open(path)
				if err != nil {
					return err
				}
				defer fw.Close()
				w := gzip.NewWriter(fw)
				defer w.Close()
				if _, err := io.Copy(w, fr); err != nil {
					return err
				}
				return os.Remove(path)
			})
		},
	}

	cmd.PersistentFlags().StringVarP(&root.output, "output", "o", "manpages", "output directory")
	root.cmd = cmd
	return root
}
