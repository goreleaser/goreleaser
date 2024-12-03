package cmd

import (
	"fmt"
	"os"
	"regexp"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/static"
	"github.com/spf13/cobra"
)

type initCmd struct {
	cmd    *cobra.Command
	config string
	lang   string
}

const gitignorePath = ".gitignore"

func newInitCmd() *initCmd {
	root := &initCmd{}
	cmd := &cobra.Command{
		Use:               "init",
		Aliases:           []string{"i"},
		Short:             "Generates a .goreleaser.yaml file",
		SilenceUsage:      true,
		SilenceErrors:     true,
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		PreRun: func(cmd *cobra.Command, _ []string) {
			if cmd.Flags().Lookup("language").Changed {
				return
			}

			// try to figure out which kind of project is this...
			if _, err := os.Stat("build.zig"); err == nil {
				root.lang = "zig"
				log.Info("project contains a 'build.zig', using default zig configuration")
				return
			}
			if _, err := os.Stat("Cargo.toml"); err == nil {
				root.lang = "rust"
				log.Info("project contains a 'Cargo.toml', using default rust configuration")
				return
			}
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			if _, err := os.Stat(root.config); err == nil {
				return fmt.Errorf("%s already exists, delete it and run the command again", root.config)
			}
			conf, err := os.OpenFile(root.config, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0o644)
			if err != nil {
				return err
			}
			defer conf.Close()

			log.Infof(boldStyle.Render(fmt.Sprintf("Generating %s file", root.config)))

			var example []byte
			switch root.lang {
			case "zig":
				example = static.ZigExampleConfig
			case "rust":
				example = static.RustExampleConfig
			case "go":
				example = static.GoExampleConfig
			default:
				return fmt.Errorf("invalid language: %s", root.lang)
			}

			if _, err := conf.Write(example); err != nil {
				return err
			}

			if !hasDistIgnored(gitignorePath) {
				gitignore, err := os.OpenFile(gitignorePath, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
				if err != nil {
					return err
				}
				defer gitignore.Close()
				if _, err := gitignore.WriteString("\ndist/\n"); err != nil {
					return err
				}
			}
			log.WithField("file", root.config).Info("config created; please edit accordingly to your needs")
			return nil
		},
	}

	cmd.Flags().StringVarP(&root.lang, "language", "l", "go", "Which language will be used")
	cmd.Flags().StringVarP(&root.config, "config", "f", ".goreleaser.yaml", "Load configuration from file")
	_ = cmd.MarkFlagFilename("config", "yaml", "yml")

	_ = cmd.RegisterFlagCompletionFunc(
		"language",
		cobra.FixedCompletions(
			[]string{"go", "rust", "zig"},
			cobra.ShellCompDirectiveDefault,
		),
	)

	root.cmd = cmd
	return root
}

func hasDistIgnored(path string) bool {
	bts, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	exp := regexp.MustCompile("(?m)^dist/$")
	return exp.Match(bts)
}
