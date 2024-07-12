package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

type docsCmd struct {
	cmd *cobra.Command
}

func newDocsCmd() *docsCmd {
	root := &docsCmd{}
	cmd := &cobra.Command{
		Use:                   "docs",
		Short:                 "Generates GoReleaser's command line docs",
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Hidden:                true,
		Args:                  cobra.NoArgs,
		ValidArgsFunction:     cobra.NoFileCompletions,
		RunE: func(_ *cobra.Command, _ []string) error {
			root.cmd.Root().DisableAutoGenTag = true
			return doc.GenMarkdownTreeCustom(root.cmd.Root(), "www/docs/cmd", func(_ string) string {
				return ""
			}, func(s string) string {
				return s
			})
		},
	}

	root.cmd = cmd
	return root
}
