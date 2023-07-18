package cmd

import (
	"fmt"
	"os"
	"regexp"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/static"
	"github.com/spf13/cobra"
)

type initCmd struct {
	cmd    *cobra.Command
	config string
}

const gitignorePath = ".gitignore"

func newInitCmd() *initCmd {
	root := &initCmd{}
	cmd := &cobra.Command{
		Use:               "init",
		Aliases:           []string{"i"},
		Short:             "Generates a .goreleaser.yaml file",
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(_ *cobra.Command, _ []string) error {
			if _, err := os.Stat(root.config); err == nil {
				return fmt.Errorf("%s already exists, delete it and run the command again", root.config)
			}
			conf, err := os.OpenFile(root.config, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0o644)
			if err != nil {
				return err
			}
			defer conf.Close()

			log.Infof(boldStyle.Render(fmt.Sprintf("Generating %s file", root.config)))
			if _, err := conf.Write(static.ExampleConfig); err != nil {
				return err
			}

			if !hasDistIgnored(gitignorePath) {
				gitignore, err := os.OpenFile(gitignorePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
				if err != nil {
					return err
				}
				defer gitignore.Close()
				if _, err := gitignore.WriteString("\ndist/\n"); err != nil {
					return err
				}
			}
			log.WithField("file", root.config).Info("config created; please edit accordingly to your needs")
			return nil
		},
	}

	cmd.Flags().StringVarP(&root.config, "config", "f", ".goreleaser.yaml", "Load configuration from file")
	_ = cmd.MarkFlagFilename("config", "yaml", "yml")

	root.cmd = cmd
	return root
}

func hasDistIgnored(path string) bool {
	bts, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	exp := regexp.MustCompile("(?m)^dist/$")
	return exp.Match(bts)
}
