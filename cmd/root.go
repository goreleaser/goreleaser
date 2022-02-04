package cmd

import (
	"errors"
	"os"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/fatih/color"
	"github.com/muesli/coral"
)

func Execute(version string, exit func(int), args []string) {
	// enable colored output on travis
	if os.Getenv("CI") != "" {
		color.NoColor = false
	}
	log.SetHandler(cli.Default)
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
	cmd   *coral.Command
	debug bool
	exit  func(int)
}

func newRootCmd(version string, exit func(int)) *rootCmd {
	root := &rootCmd{
		exit: exit,
	}
	cmd := &coral.Command{
		Use:   "goreleaser",
		Short: "Deliver Go binaries as fast and easily as possible",
		Long: `GoReleaser is a release automation tool for Go projects.
Its goal is to simplify the build, release and publish steps while providing
variant customization options for all steps.

GoReleaser is built for CI tools, you only need to download and execute it
in your build script. Of course, you can also install it locally if you wish.

You can also customize your entire release process through a
single .goreleaser.yaml file.
`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          coral.NoArgs,
		PersistentPreRun: func(cmd *coral.Command, args []string) {
			if root.debug {
				log.SetLevel(log.DebugLevel)
				log.Debug("debug logs enabled")
			}
		},
	}

	cmd.PersistentFlags().BoolVar(&root.debug, "debug", false, "Enable debug mode")
	cmd.AddCommand(
		newBuildCmd().cmd,
		newReleaseCmd().cmd,
		newCheckCmd().cmd,
		newInitCmd().cmd,
		newDocsCmd().cmd,
		newManCmd().cmd,
		newSchemaCmd().cmd,
	)

	root.cmd = cmd
	return root
}

func shouldPrependRelease(cmd *coral.Command, args []string) bool {
	// find current cmd, if its not root, it means the user actively
	// set a command, so let it go
	xmd, _, _ := cmd.Find(args)
	if xmd != cmd {
		return false
	}

	// allow help and the two __complete commands.
	if len(args) > 0 && (args[0] == "help" || args[0] == "completion" ||
		args[0] == coral.ShellCompRequestCmd || args[0] == coral.ShellCompNoDescRequestCmd) {
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
