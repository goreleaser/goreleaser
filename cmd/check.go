package cmd

import (
	"errors"
	"fmt"
	"io"
	"slices"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/defaults"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/spf13/cobra"
)

type checkCmd struct {
	cmd        *cobra.Command
	config     string
	quiet      bool
	deprecated bool
	checked    int
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
		ValidArgsFunction: func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
			return []string{"yaml", "yml"}, cobra.ShellCompDirectiveFilterFileExt
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if root.quiet {
				log.Log = log.New(io.Discard)
			}

			if root.config != "" || len(args) == 0 {
				args = append(args, root.config)
			}

			exits := []int{}
			for _, config := range args {
				cfg, path, err := loadConfigCheck(config)
				if err != nil {
					return err
				}
				ctx := context.Wrap(cmd.Context(), cfg)
				ctx.Deprecated = root.deprecated

				log.WithField("path", path).
					Info(boldStyle.Render("checking"))

				if err := (defaults.Pipe{}).Run(ctx); err != nil {
					exits = append(exits, 1)
					log.WithError(fmt.Errorf("configuration is invalid: %w", err)).Error(path)
				}

				if ctx.Deprecated {
					exits = append(exits, 2)
					log.WithError(errors.New("configuration is valid, but uses deprecated properties")).Warn(path)
				}
			}

			root.checked = len(args)

			// so we get the exits in the right order, and can exit exits[0]
			slices.Sort(exits)

			if len(exits) > 0 {
				return gerrors.WrapExit(
					fmt.Errorf(
						"%d out of %d configuration file(s) have issues",
						len(exits), len(args),
					),
					"check failed",
					exits[0],
				)
			}

			log.Info(boldStyle.Render(fmt.Sprintf("%d configuration file(s) validated", len(args))))
			return nil
		},
	}

	cmd.Flags().StringVarP(&root.config, "config", "f", "", "Configuration file(s) to check")
	_ = cmd.MarkFlagFilename("config", "yaml", "yml")
	cmd.Flags().BoolVarP(&root.quiet, "quiet", "q", false, "Quiet mode: no output")
	cmd.Flags().BoolVar(&root.deprecated, "deprecated", false, "Force print the deprecation message - tests only")
	_ = cmd.Flags().MarkHidden("deprecated")
	_ = cmd.Flags().MarkHidden("config")

	root.cmd = cmd
	return root
}
