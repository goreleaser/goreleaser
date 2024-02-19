package cmd

import (
	"fmt"
	"io"
	"os/exec"
	"sync"

	"github.com/caarlos0/ctrlc"
	"github.com/caarlos0/log"
	"github.com/charmbracelet/lipgloss"
	"github.com/goreleaser/goreleaser/internal/middleware/skip"
	"github.com/goreleaser/goreleaser/internal/pipe/defaults"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/goreleaser/goreleaser/pkg/healthcheck"
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
		RunE: func(_ *cobra.Command, _ []string) error {
			if root.quiet {
				log.Log = log.New(io.Discard)
			}

			cfg, err := loadConfig(root.config)
			if err != nil {
				return err
			}
			ctx := context.New(cfg)

			if err := ctrlc.Default.Run(ctx, func() error {
				log.Info(boldStyle.Render("checking tools..."))

				err := defaults.Pipe{}.Run(ctx)
				if err != nil {
					return err
				}

				log.IncreasePadding()
				defer log.ResetPadding()

				var errs []error
				for _, hc := range healthcheck.Healthcheckers {
					_ = skip.Maybe(hc, func(ctx *context.Context) error {
						for _, tool := range hc.Dependencies(ctx) {
							if err := checkPath(tool); err != nil {
								errs = append(errs, err)
							}
						}
						return nil
					})(ctx)
				}

				if len(errs) == 0 {
					return nil
				}

				return fmt.Errorf("one or more needed tools are not present")
			}); err != nil {
				return err
			}

			log.Infof(boldStyle.Render("done!"))
			return nil
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

func checkPath(tool string) error {
	if _, ok := toolsChecked.LoadOrStore(tool, true); ok {
		return nil
	}
	if _, err := exec.LookPath(tool); err != nil {
		st := log.Styles[log.ErrorLevel]
		log.Warnf("%s %s - %s", st.Render("⚠"), codeStyle.Render(tool), st.Render("not present in path"))
		return err
	}
	st := lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Bold(true)
	log.Infof("%s %s", st.Render("✓"), codeStyle.Render(tool))
	return nil
}
