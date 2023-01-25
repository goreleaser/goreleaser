package cmd

import (
	"fmt"
	"os"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/static"
	"github.com/spf13/cobra"
)

type initCmd struct {
	cmd    *cobra.Command
	config string
}

func newInitCmd() *initCmd {
	root := &initCmd{}
	cmd := &cobra.Command{
		Use:           "init",
		Aliases:       []string{"i"},
		Short:         "Generates a .goreleaser.yaml file",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			conf, err := os.OpenFile(root.config, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0o644)
			if err != nil {
				return err
			}
			defer conf.Close()

			log.Infof(boldStyle.Render(fmt.Sprintf("Generating %s file", root.config)))
			if _, err := conf.Write(static.ExampleConfig); err != nil {
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
