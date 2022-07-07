package cmd

import (
	"strings"

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
		RunE: func(cmd *cobra.Command, args []string) error {
			root.cmd.Root().DisableAutoGenTag = true
			return doc.GenMarkdownTreeCustom(root.cmd.Root(), "www/docs/cmd", func(_ string) string {
				return ""
			}, func(s string) string {
				return "/cmd/" + strings.TrimSuffix(s, ".md") + "/"
			})
		},
	}

	root.cmd = cmd
	return root
}
