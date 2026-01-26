package cmd

import (
	stdctx "context"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/middleware/skip"
	"github.com/goreleaser/goreleaser/v2/internal/pipe/defaults"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/goreleaser/goreleaser/v2/pkg/healthcheck"
	"github.com/spf13/cobra"
)

type healthcheckCmd struct {
	cmd    *cobra.Command
	config string
	quiet  bool
}

func newHealthcheckCmd() *healthcheckCmd {
	root := &healthcheckCmd{}
	cmd := &cobra.Command{
		Use:               "healthcheck",
		Aliases:           []string{"hc"},
		Short:             "Checks if needed tools are installed",
		Long:              `Check if the needed tools are available in your $PATH, exits 1 if any of them are missing.`,
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if root.quiet {
				log.Log = log.New(io.Discard)
			}

			cfg, err := loadConfig(true, root.config)
			if err != nil {
				return err
			}
			ctx := context.Wrap(cmd.Context(), cfg)

			log.Info(boldStyle.Render("checking tools..."))

			if err := (defaults.Pipe{}).Run(ctx); err != nil {
				return err
			}

			log.IncreasePadding()
			defer log.ResetPadding()

			var errs []error
			for _, hc := range healthcheck.DependencyCheckers {
				_ = skip.Maybe(hc, func(ctx *context.Context) error {
					for _, tool := range hc.Dependencies(ctx) {
						if err := checkPath(ctx, tool); err != nil {
							errs = append(errs, err)
						}
					}
					return nil
				})(ctx)
			}
			for _, hc := range healthcheck.HealthCheckers {
				_ = skip.Maybe(hc, func(ctx *context.Context) error {
					if err := check(hc.String(), hc.Healthcheck(ctx)); err != nil {
						errs = append(errs, err)
					}
					return nil
				})(ctx)
			}

			if len(errs) == 0 {
				log.Infof(boldStyle.Render("done!"))
				return nil
			}

			return errors.New("one or more checks failed")
		},
	}

	cmd.Flags().StringVarP(&root.config, "config", "f", "", "Configuration file")
	_ = cmd.MarkFlagFilename("config", "yaml", "yml")
	cmd.Flags().BoolVarP(&root.quiet, "quiet", "q", false, "Quiet mode: no output")
	_ = cmd.Flags().MarkHidden("deprecated")

	root.cmd = cmd
	return root
}

var toolsChecked = &sync.Map{}

func check(name string, err error) error {
	if err == nil {
		st := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
		log.Infof("%s %s", st.Render("✓"), codeStyle.Render(name))
		return nil
	}
	st := log.Styles[log.ErrorLevel]
	log.Warnf("%s %s - %s", st.Render("⚠"), codeStyle.Render(name), st.Render(err.Error()))
	return err
}

func checkPath(ctx stdctx.Context, tool string) error {
	if _, ok := toolsChecked.LoadOrStore(tool, true); ok {
		return nil
	}
	args := strings.Fields(tool)
	if _, err := exec.LookPath(args[0]); err != nil {
		st := log.Styles[log.ErrorLevel]
		log.Warnf("%s %s - %s", st.Render("⚠"), codeStyle.Render(tool), st.Render("not present in path"))
		return err
	}
	if len(args) > 1 {
		if err := exec.CommandContext(ctx, args[0], args[1:]...).Run(); err != nil {
			st := log.Styles[log.ErrorLevel]
			log.Warnf("%s %s - %s", st.Render("⚠"), codeStyle.Render(tool), st.Render("command failed"))
			return err
		}
	}
	st := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	log.Infof("%s %s", st.Render("✓"), codeStyle.Render(tool))
	return nil
}
