package cmd

import (
	"fmt"

	"github.com/apex/log"
	"github.com/caarlos0/ctrlc"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/internal/pipe/defaults"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

type checkCmd struct {
	cmd        *cobra.Command
	config     string
	deprecated bool
}

func newCheckCmd() *checkCmd {
	var root = &checkCmd{}
	var cmd = &cobra.Command{
		Use:           "check",
		Aliases:       []string{"c"},
		Short:         "Checks if configuration is valid",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := loadConfig(root.config)
			if err != nil {
				return err
			}
			var ctx = context.New(cfg)
			ctx.Deprecated = root.deprecated

			if err := ctrlc.Default.Run(ctx, func() error {
				log.Info(color.New(color.Bold).Sprint("checking config:"))
				return defaults.Pipe{}.Run(ctx)
			}); err != nil {
				log.WithError(err).Error(color.New(color.Bold).Sprintf("config is invalid"))
				return errors.Wrap(err, "invalid config")
			}

			if ctx.Deprecated {
				return wrapErrorWithCode(
					fmt.Errorf("config is valid, but uses deprecated properties, check logs above for details"),
					2,
					"",
				)
			}
			log.Infof(color.New(color.Bold).Sprintf("config is valid"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&root.config, "config", "f", "", "Configuration file to check")
	cmd.Flags().BoolVar(&root.deprecated, "deprecated", false, "Force print the deprecation message - tests only")
	_ = cmd.Flags().MarkHidden("deprecated")

	root.cmd = cmd
	return root
}
