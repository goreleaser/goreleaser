package cmd

import (
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/spf13/cobra"
)

// nolint: gochecknoglobals
var (
	version = "dev"
	commit  = ""
	date    = ""
	builtBy = ""
)

func Execute() {
	fmt.Println()
	defer fmt.Println()

	if err := NewRootCmd().Execute(); err != nil {
		log.WithError(err).Error("command failed")
		if gerr, ok := err.(*GoreleaserError); ok {
			os.Exit(gerr.Exit())
		}
		os.Exit(1)
	}
}

func NewRootCmd() *cobra.Command {
	var cmd = &cobra.Command{
		Use:     "goreleaser",
		Short:   "Deliver Go binaries as fast and easily as possible",
		Version: buildVersion(version, commit, date, builtBy),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if debug, _ := cmd.Flags().GetBool("debug"); debug {
				log.SetLevel(log.DebugLevel)
				log.Debug("enabled debug logs")
			}
		},
	}

	cmd.PersistentFlags().Bool("debug", false, "Enable debug mode")
	cmd.AddCommand(
		NewReleaseCmd().cmd,
		NewCheckCmd().cmd,
		NewInitCmd().cmd,
	)

	return cmd
}

func buildVersion(version, commit, date, builtBy string) string {
	var result = fmt.Sprintf("version: %s", version)
	if commit != "" {
		result = fmt.Sprintf("%s\ncommit: %s", result, commit)
	}
	if date != "" {
		result = fmt.Sprintf("%s\nbuilt at: %s", result, date)
	}
	if builtBy != "" {
		result = fmt.Sprintf("%s\nbuilt by: %s", result, builtBy)
	}
	return result
}
