package cmd

import (
	"strings"

	"github.com/muesli/coral"
	"github.com/muesli/coral/doc"
)

type docsCmd struct {
	cmd *coral.Command
}

func newDocsCmd() *docsCmd {
	root := &docsCmd{}
	cmd := &coral.Command{
		Use:                   "docs",
		Short:                 "Generates GoReleaser's command line docs",
		SilenceUsage:          true,
		DisableFlagsInUseLine: true,
		Hidden:                true,
		Args:                  coral.NoArgs,
		RunE: func(cmd *coral.Command, args []string) error {
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
