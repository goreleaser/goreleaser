package cmd

import (
	"errors"
	"fmt"
	"time"

	goversion "github.com/caarlos0/go-version"
	"github.com/caarlos0/log"
	"github.com/charmbracelet/lipgloss"
	"github.com/goreleaser/goreleaser/pkg/context"
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

	if err := cmd.cmd.Execute(); err != nil {
		code := 1
		msg := "command failed"
		eerr := &exitError{}
		if errors.As(err, &eerr) {
			code = eerr.code
			if eerr.details != "" {
				msg = eerr.details
			}
		}
		log.WithError(err).Error(msg)
		cmd.exit(code)
	}
}

type rootCmd struct {
	cmd     *cobra.Command
	verbose bool
	exit    func(int)

	// Deprecated: use verbose instead.
	debug bool
}

func newRootCmd(version goversion.Info, exit func(int)) *rootCmd {
	root := &rootCmd{
		exit: exit,
	}
	cmd := &cobra.Command{
		Use:   "goreleaser",
		Short: "Deliver Go binaries as fast and easily as possible",
		Long: `GoReleaser is a release automation tool for Go projects.
Its goal is to simplify the build, release and publish steps while providing variant customization options for all steps.

GoReleaser is built for CI tools, you only need to download and execute it in your build script. Of course, you can also install it locally if you wish.

You can customize your entire release process through a single .goreleaser.yaml file.

Check out our website for more information, examples and documentation: https://goreleaser.com
`,
		Version:           version.String(),
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			if root.verbose || root.debug {
				log.SetLevel(log.DebugLevel)
				log.Debug("verbose output enabled")
			}
		},
		PersistentPostRun: func(_ *cobra.Command, _ []string) {
			log.Info("thanks for using goreleaser!")
		},
	}
	cmd.SetVersionTemplate("{{.Version}}")

	cmd.PersistentFlags().BoolVar(&root.debug, "debug", false, "Enable verbose mode")
	cmd.PersistentFlags().BoolVar(&root.verbose, "verbose", false, "Enable verbose mode")
	_ = cmd.Flags().MarkDeprecated("debug", "please use --verbose instead")
	_ = cmd.Flags().MarkHidden("debug")
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
	for _, s := range []string{"-h", "--help", "-v", "--version"} {
		if s == args[0] {
			// if it is, we should run the root cmd
			return false
		}
	}

	// otherwise, we should probably prepend release
	return true
}

func deprecateWarn(ctx *context.Context) {
	if ctx.Deprecated {
		log.Warn(boldStyle.Render("you are using deprecated options, check the output above for details"))
	}
}

func timedRunE(verb string, rune func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		start := time.Now()

		log.Infof(boldStyle.Render(fmt.Sprintf("starting %s...", verb)))

		if err := rune(cmd, args); err != nil {
			return wrapError(err, boldStyle.Render(fmt.Sprintf("%s failed after %s", verb, time.Since(start).Truncate(time.Second))))
		}

		log.Infof(boldStyle.Render(fmt.Sprintf("%s succeeded after %s", verb, time.Since(start).Truncate(time.Second))))
		return nil
	}
}
