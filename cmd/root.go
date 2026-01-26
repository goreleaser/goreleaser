package cmd

import (
	"cmp"
	stdctx "context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"charm.land/lipgloss/v2"
	goversion "github.com/caarlos0/go-version"
	"github.com/caarlos0/log"
	"github.com/charmbracelet/fang"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/spf13/cobra"
)

var (
	boldStyle = lipgloss.NewStyle().Bold(true)
	codeStyle = lipgloss.NewStyle().Italic(true)
)

func Execute(version goversion.Info, exit func(int), args []string) {
	newRootCmd(version, exit).Execute(args)
}

func (cmd *rootCmd) Execute(args []string) {
	cmd.cmd.SetArgs(args)

	if shouldPrependRelease(cmd.cmd, args) {
		cmd.cmd.SetArgs(append([]string{"release"}, args...))
	}

	if shouldDisableLogs(args) {
		log.SetLevel(log.FatalLevel)
	}

	if err := fang.Execute(
		stdctx.Background(),
		cmd.cmd,
		fang.WithVersion(cmd.cmd.Version),
		fang.WithErrorHandler(errorHandler),
		fang.WithColorSchemeFunc(fang.AnsiColorScheme),
		fang.WithNotifySignal(os.Interrupt, os.Kill),
	); err != nil {
		cmd.exit(gerrors.ExitOf(err))
	}
}

func errorHandler(_ io.Writer, _ fang.Styles, err error) {
	log := log.WithError(err)
	for k, v := range gerrors.DetailsOf(err) {
		log = log.WithField(k, v)
	}
	log.Error(cmp.Or(
		gerrors.MessageOf(err),
		"command failed",
	))
}

type rootCmd struct {
	cmd     *cobra.Command
	verbose bool
	exit    func(int)
}

func newRootCmd(version goversion.Info, exit func(int)) *rootCmd {
	root := &rootCmd{
		exit: exit,
	}
	cmd := &cobra.Command{
		Use:   "goreleaser",
		Short: "Release engineering, simplified",
		Long: `Release engineering, simplified.

GoReleaser is a release automation tool, built with love and care by @caarlos0 and many contributors.

Complete documentation is available at https://goreleaser.com`,
		Version:           version.String(),
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		Example: `
# Initialize your project:
goreleaser init

# Verify your configuration:
goreleaser check

# Verify dependencies:
goreleaser healthcheck

# Build the binaries only:
goreleaser build

# Run a snapshot release:
goreleaser release --snapshot

# Run a complete release:
goreleaser release
		`,
		PersistentPreRun: func(*cobra.Command, []string) {
			if root.verbose {
				log.SetLevel(log.DebugLevel)
				log.Debug("verbose output enabled")
			}
		},
		PersistentPostRun: func(*cobra.Command, []string) {
			log.Info("thanks for using GoReleaser!")
		},
	}
	cmd.SetVersionTemplate("{{.Version}}")

	cmd.PersistentFlags().BoolVar(&root.verbose, "verbose", false, "Enable verbose mode")
	cmd.AddCommand(
		newBuildCmd().cmd,
		newReleaseCmd().cmd,
		newCheckCmd().cmd,
		newHealthcheckCmd().cmd,
		newInitCmd().cmd,
		newDocsCmd().cmd,
		newManCmd().cmd,
		newSchemaCmd().cmd,
	)
	root.cmd = cmd
	return root
}

func shouldDisableLogs(args []string) bool {
	return len(args) > 0 && (args[0] == "help" ||
		args[0] == "completion" ||
		args[0] == "man" ||
		args[0] == "docs" ||
		args[0] == "jsonschema" ||
		args[0] == cobra.ShellCompRequestCmd ||
		args[0] == cobra.ShellCompNoDescRequestCmd)
}

func shouldPrependRelease(cmd *cobra.Command, args []string) bool {
	// find current cmd, if its not root, it means the user actively
	// set a command, so let it go
	xmd, _, _ := cmd.Find(args)
	if xmd != cmd {
		return false
	}

	// allow help and the two __complete commands.
	if len(args) > 0 && (args[0] == "help" || args[0] == "completion" ||
		args[0] == cobra.ShellCompRequestCmd || args[0] == cobra.ShellCompNoDescRequestCmd) {
		return false
	}

	// if we have != 1 args, assume its a release
	if len(args) != 1 {
		return true
	}

	// given that its 1, check if its one of the valid standalone flags
	// for the root cmd
	return !slices.Contains([]string{"-h", "--help", "-v", "--version"}, args[0])
}

func deprecateWarn(ctx *context.Context) {
	if ctx.Deprecated {
		log.Warn(boldStyle.Render("you are using deprecated options, check the output above for details"))
	}
}

func after(start time.Time) time.Duration {
	return time.Since(start).Truncate(time.Second)
}

func decorateWithCtxErr(ctx stdctx.Context, err error, verb string, after time.Duration) error {
	if errors.Is(ctx.Err(), stdctx.Canceled) {
		return gerrors.Wrap(ctx.Err(), fmt.Sprintf("%s interrupted after %s", verb, after))
	}
	if errors.Is(ctx.Err(), stdctx.DeadlineExceeded) {
		return gerrors.Wrap(ctx.Err(), fmt.Sprintf("%s timed out after %s", verb, after))
	}
	return gerrors.Wrap(err, fmt.Sprintf("%s failed after %s", verb, after))
}
