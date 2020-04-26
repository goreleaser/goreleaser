package cmd

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/apex/log"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/internal/static"
	"github.com/spf13/cobra"
)

type initCmd struct {
	cmd    *cobra.Command
	config string
}

func NewInitCmd() *initCmd {
	var root = &initCmd{}
	var cmd = &cobra.Command{
		Use:           "init",
		Aliases:       []string{"i"},
		Short:         "Generates a .goreleaser.yml file",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := os.Stat(root.config); !os.IsNotExist(err) {
				if err != nil {
					return err
				}
				return fmt.Errorf("%s already exists", root.config)
			}
			log.Infof(color.New(color.Bold).Sprintf("Generating %s file", root.config))
			if err := ioutil.WriteFile(root.config, []byte(static.ExampleConfig), 0644); err != nil {
				return err
			}

			log.WithField("file", root.config).Info("config created; please edit accordingly to your needs")
			return nil
		},
	}

	cmd.Flags().StringVarP(&root.config, "config", "f", ".goreleaser.yml", "Load configuration from file")

	root.cmd = cmd
	return root
}
