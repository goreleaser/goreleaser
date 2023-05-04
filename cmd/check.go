package cmd

import (
	"fmt"
	"io"
	"os"

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
		Use:           "check [configuration files]",
		Aliases:       []string{"c"},
		Short:         "Checks if configuration is valid",
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.ArbitraryArgs,
		RunE: func(_ *cobra.Command, args []string) error {
			if root.quiet {
				log.Log = log.New(io.Discard)
			}

			var errs []*exitError
			if root.config != "" {
				args = append(args, root.config)
			}

			for _, config := range args {
				cfg, err := loadConfig(config)
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
					errs = append(errs, wrapErrorWithCode(
						fmt.Errorf("configuration is invalid: %w", err),
						1,
						config,
					))
				}

				if ctx.Deprecated {
					errs = append(errs, wrapErrorWithCode(
						fmt.Errorf("configuration is valid, but uses deprecated properties"),
						2,
						config,
					))
				}
			}

			exit := 0
			for _, err := range errs {
				if err.code < exit || exit == 0 {
					exit = err.code
				}
				log.Log = log.New(os.Stderr)
				if err.code == 1 {
					log.WithError(err.err).Error(err.details)
				} else {
					log.WithError(err.err).Warn(err.details)
				}
			}
			if exit > 0 {
				return wrapErrorWithCode(fmt.Errorf("%d out of %d configuration file(s) have issues", len(errs), len(args)), exit, "")
			}

			log.Info(boldStyle.Render(fmt.Sprintf("%d configuration file(s) validated", len(args))))
			return nil
		},
	}

	cmd.Flags().StringVarP(&root.config, "config", "f", "", "Configuration file(s) to check")
	cmd.Flags().BoolVarP(&root.quiet, "quiet", "q", false, "Quiet mode: no output")
	cmd.Flags().BoolVar(&root.deprecated, "deprecated", false, "Force print the deprecation message - tests only")
	_ = cmd.Flags().MarkHidden("deprecated")
	_ = cmd.Flags().MarkHidden("config")

	root.cmd = cmd
	return root
}
