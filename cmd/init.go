package cmd

import (
	"os"

	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/internal/static"
	"github.com/muesli/coral"
)

type initCmd struct {
	cmd    *coral.Command
	config string
}

func newInitCmd() *initCmd {
	root := &initCmd{}
	cmd := &coral.Command{
		Use:           "init",
		Aliases:       []string{"i"},
		Short:         "Generates a .goreleaser.yaml file",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          coral.NoArgs,
		RunE: func(cmd *coral.Command, args []string) error {
			conf, err := os.OpenFile(root.config, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0o644)
			if err != nil {
				return err
			}
			defer conf.Close()

			log.Infof(color.New(color.Bold).Sprintf("Generating %s file", root.config))
			if _, err := conf.WriteString(static.ExampleConfig); err != nil {
				return err
			}

			gitignore, err := os.OpenFile(".gitignore", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
			if err != nil {
				return err
			}
			defer gitignore.Close()
			if _, err := gitignore.WriteString("\ndist/\n"); err != nil {
				return err
			}

			log.WithField("file", root.config).Info("config created; please edit accordingly to your needs")
			return nil
		},
	}

	cmd.Flags().StringVarP(&root.config, "config", "f", ".goreleaser.yaml", "Load configuration from file")

	root.cmd = cmd
	return root
}
