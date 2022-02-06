package cmd

import (
	"fmt"
	"io"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/caarlos0/ctrlc"
	"github.com/fatih/color"
	"github.com/goreleaser/goreleaser/internal/pipe/defaults"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/muesli/coral"
)

type checkCmd struct {
	cmd        *coral.Command
	config     string
	quiet      bool
	deprecated bool
}

func newCheckCmd() *checkCmd {
	root := &checkCmd{}
	cmd := &coral.Command{
		Use:           "check",
		Aliases:       []string{"c"},
		Short:         "Checks if configuration is valid",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          coral.NoArgs,
		RunE: func(cmd *coral.Command, args []string) error {
			if root.quiet {
				log.SetHandler(cli.New(io.Discard))
			}

			cfg, err := loadConfig(root.config)
			if err != nil {
				return err
			}
			ctx := context.New(cfg)
			ctx.Deprecated = root.deprecated

			if err := ctrlc.Default.Run(ctx, func() error {
				log.Info(color.New(color.Bold).Sprint("checking config:"))
				return defaults.Pipe{}.Run(ctx)
			}); err != nil {
				log.WithError(err).Error(color.New(color.Bold).Sprintf("config is invalid"))
				return fmt.Errorf("invalid config: %w", err)
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
	cmd.Flags().BoolVarP(&root.quiet, "quiet", "q", false, "Quiet mode: no output")
	cmd.Flags().BoolVar(&root.deprecated, "deprecated", false, "Force print the deprecation message - tests only")
	_ = cmd.Flags().MarkHidden("deprecated")

	root.cmd = cmd
	return root
}
