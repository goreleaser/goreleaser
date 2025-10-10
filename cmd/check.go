package cmd

import (
	"errors"
	"fmt"
	"io"
	"maps"
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

			results := map[string]checkResult{}
			for _, config := range args {
				cfg, path, err := loadConfigCheck(config)
				if err != nil {
					log.WithError(fmt.Errorf("configuration is invalid: %w", err)).Error(path)
					results[path] = checkResult{
						Err: err,
					}
					continue
				}
				ctx := context.Wrap(cmd.Context(), cfg)
				ctx.Deprecated = root.deprecated

				log.WithField("path", path).
					Debug(boldStyle.Render("checking"))

				if err := (defaults.Pipe{}).Run(ctx); err != nil {
					log.WithError(fmt.Errorf("configuration is invalid: %w", err)).Error(path)
					results[path] = checkResult{
						Err: err,
					}
					continue
				}

				results[path] = checkResult{
					Deprecated: slices.Collect(maps.Keys(ctx.NotifiedDeprecations)),
					Valid:      true,
				}
				if ctx.Deprecated {
					log.WithError(errors.New("configuration is valid, but uses deprecated properties")).Warn(path)
					continue
				}
				log.WithField("path", path).
					Info(boldStyle.Render("configuration is valid"))
			}

			root.checked = len(args)
			exit := 0
			issues := 0
			for _, f := range results {
				if f.Err != nil {
					exit = 1
					issues++
					continue
				}
				if len(f.Deprecated) > 0 {
					issues++
					if exit == 0 {
						exit = 2
					}
				}
			}
			if exit > 0 {
				return gerrors.WrapExit(
					fmt.Errorf(
						"%d out of %d configuration file(s) have issues",
						issues, len(args),
					),
					"check failed",
					exit,
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
	_ = cmd.Flags().MarkHidden("json")

	root.cmd = cmd
	return root
}

type checkResult struct {
	Valid      bool     `json:"valid"`
	Deprecated []string `json:"deprecated,omitempty"`
	Err        error    `json:"error,omitempty"`
}
