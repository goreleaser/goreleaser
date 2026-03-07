package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/packagejson"
	"github.com/goreleaser/goreleaser/v2/internal/pyproject"
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
			root.lang = langDetect()
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

			log.Infof(boldStyle.Render("generating ") + codeStyle.Render(root.config))

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
			case "bun":
				example = static.BunExampleConfig
			case "deno":
				example = static.DenoExampleConfig
			case "uv":
				example = static.UVExampleConfig
				gitignoreLines = append(gitignoreLines, "build/")
			case "poetry":
				example = static.PoetryExampleConfig
			default:
				return fmt.Errorf("invalid language: %s", root.lang)
			}

			if _, err := conf.Write(example); err != nil {
				return err
			}

			gitignoreModified, err := setupGitignore(gitignorePath, gitignoreLines)
			if gitignoreModified {
				log.Infof(boldStyle.Render("setting up " + codeStyle.Render(gitignorePath)))
			}
			if err != nil {
				return err
			}

			done := []string{
				boldStyle.Render("done!"),
				"please edit", codeStyle.Render(root.config),
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
			[]string{"go", "bun", "deno", "rust", "zig"},
			cobra.ShellCompDirectiveDefault,
		),
	)

	root.cmd = cmd
	return root
}

func setupGitignore(path string, lines []string) (bool, error) {
	ignored, _ := os.ReadFile(path)
	content := strings.ReplaceAll(string(ignored), "\r\n", "\n")

	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return false, err
	}
	defer f.Close()

	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		if _, err := f.WriteString("\n"); err != nil {
			return false, err
		}
	}

	var modified bool
	for _, line := range lines {
		if !strings.Contains(content, line+"\n") {
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

const (
	packageJSON   = "package.json"
	pyprojectToml = "pyproject.toml"
)

func langDetect() string {
	code := func(s string) string {
		return codeStyle.Render(s)
	}
	for lang, file := range map[string]string{
		"zig":  "build.zig",
		"rust": "Cargo.toml",
		"bun":  "bun.lockb",
		"deno": "deno.json",
	} {
		if _, err := os.Stat(file); err == nil {
			log.Info("project contains a " + code(file) + " file, using default " + code(lang) + " configuration")
			return lang
		}
	}

	if pkg, err := packagejson.Open(packageJSON); err == nil && pkg.IsBun() {
		log.Info("project contains a " + code(packageJSON) + " with " + code("@types/bun") + " in its " + code("devDependencies") + ", using default " + code("bun") + " configuration")
		return "bun"
	}

	pyproj, err := pyproject.Open(pyprojectToml)
	if err == nil {
		if pyproj.IsPoetry() {
			log.Info("project contains a " + code(pyprojectToml) + " with " + code("[tool.poetry]") + " in it, using default " + code("poetry") + " configuration")
			return "poetry"
		}
		log.Info("project contains a " + code(pyprojectToml) + " file, using default " + code("uv") + " configuration")
		return "uv"
	}

	return "go"
}
