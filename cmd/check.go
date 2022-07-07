package cmd

import (
	"fmt"
	"io"

	"github.com/caarlos0/ctrlc"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/pipe/defaults"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/spf13/cobra"
)

type checkCmd struct {
	cmd        *cobra.Command
	config     string
	quiet      bool
	deprecated bool
}

func newCheckCmd() *checkCmd {
	root := &checkCmd{}
	cmd := &cobra.Command{
		Use:           "check",
		Aliases:       []string{"c"},
		Short:         "Checks if configuration is valid",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if root.quiet {
				log.Log = log.New(io.Discard)
			}

			cfg, err := loadConfig(root.config)
			if err != nil {
				return err
			}
			ctx := context.New(cfg)
			ctx.Deprecated = root.deprecated

			if err := ctrlc.Default.Run(ctx, func() error {
				log.Info(boldStyle.Render("checking config..."))
				return defaults.Pipe{}.Run(ctx)
			}); err != nil {
				log.WithError(err).Error(boldStyle.Render("config is invalid"))
				return fmt.Errorf("invalid config: %w", err)
			}

			if ctx.Deprecated {
				return wrapErrorWithCode(
					fmt.Errorf("config is valid, but uses deprecated properties, check logs above for details"),
					2,
					"",
				)
			}
			log.Infof(boldStyle.Render("config is valid"))
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
