package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

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
				log.Info("Project contains a " + codeStyle.Render("build.zig") + " file, using default zig configuration")
				return
			}
			if _, err := os.Stat("Cargo.toml"); err == nil {
				root.lang = "rust"
				log.Info("Project contains a " + codeStyle.Render("Cargo.toml") + " file, using default rust configuration")
				return
			}
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			if _, err := os.Stat(root.config); err == nil {
				return errors.New(root.config + " already exists, delete it and run the command again")
			}
			conf, err := os.OpenFile(root.config, os.O_WRONLY|os.O_CREATE|os.O_TRUNC|os.O_EXCL, 0o644)
			if err != nil {
				return err
			}
			defer conf.Close()

			log.Infof(boldStyle.Render("Generating ") + codeStyle.Render(root.config))

			gitignoreLines := []string{"dist/"}
			var example []byte
			switch root.lang {
			case "zig":
				example = static.ZigExampleConfig
				gitignoreLines = append(gitignoreLines, ".intentionally-empty-file.o", "zig-out/", ".zig-cache/")
			case "rust":
				example = static.RustExampleConfig
				gitignoreLines = append(gitignoreLines, ".intentionally-empty-file.o", "target/")
			case "go":
				example = static.GoExampleConfig
			default:
				return fmt.Errorf("invalid language: %s", root.lang)
			}

			if _, err := conf.Write(example); err != nil {
				return err
			}

			gitignoreModified, err := setupGitignore(gitignorePath, gitignoreLines)
			if gitignoreModified {
				log.Infof(boldStyle.Render("Setting up " + gitignorePath))
			}
			if err != nil {
				return err
			}

			done := []string{
				boldStyle.Render("Done!"),
				"Please edit", codeStyle.Render(root.config),
			}
			if gitignoreModified {
				done = append(done, "and", codeStyle.Render(gitignorePath))
			}
			done = append(done, "accordingly.")
			log.Info(strings.Join(done, " "))
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

func setupGitignore(path string, lines []string) (bool, error) {
	ignored, _ := os.ReadFile(path)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return false, err
	}
	defer f.Close()
	var modified bool
	for _, line := range lines {
		if !strings.Contains(string(ignored), line+"\n") {
			if !modified {
				line = "# Added by goreleaser init:\n" + line
				modified = true
			}
			if _, err := f.WriteString(line + "\n"); err != nil {
				return true, err
			}
		}
	}
	return modified, nil
}
