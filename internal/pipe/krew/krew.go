package krew

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"gopkg.in/yaml.v2"
)

const (
	krewConfigExtra = "KrewConfig"
	pluginsFolder   = "plugins"
	kind            = "Plugin"
	apiVersion      = "krew.googlecontainertools.github.com/v1alpha2"
)

var ErrNoArchivesFound = errors.New("no archives found")

// Pipe for krew plugin deployment.
type Pipe struct{}

func (Pipe) String() string                 { return "krew" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Krews) == 0 }

func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Krews {
		krew := &ctx.Config.Krews[i]

		if krew.CommitAuthor.Name == "" {
			krew.CommitAuthor.Name = "goreleaserbot"
		}
		if krew.CommitAuthor.Email == "" {
			krew.CommitAuthor.Email = "goreleaser@carlosbecker.com"
		}
		if krew.CommitMessageTemplate == "" {
			krew.CommitMessageTemplate = "Krew plugin update for {{ .ProjectName }} version {{ .Tag }}"
		}
		if krew.Name == "" {
			krew.Name = ctx.Config.ProjectName
		}
		if krew.Goarm == "" {
			krew.Goarm = "6"
		}
	}

	return nil
}

func (Pipe) Run(ctx *context.Context) error {
	cli, err := client.New(ctx)
	if err != nil {
		return err
	}

	return runAll(ctx, cli)
}

func runAll(ctx *context.Context, cli client.Client) error {
	for _, krew := range ctx.Config.Krews {
		err := doRun(ctx, krew, cli)
		if err != nil {
			return err
		}
	}
	return nil
}

func doRun(ctx *context.Context, krew config.Krew, cl client.Client) error {
	if krew.Index.Name == "" {
		return pipe.Skip("krew plugin name is not set")
	}

	filters := []artifact.Filter{
		artifact.Or(
			artifact.ByGoos("darwin"),
			artifact.ByGoos("linux"),
			artifact.ByGoos("windows"),
		),
		artifact.Or(
			artifact.ByGoarch("amd64"),
			artifact.ByGoarch("arm64"),
			artifact.ByGoarch("all"),
			artifact.And(
				artifact.ByGoarch("arm"),
				artifact.ByGoarm(krew.Goarm),
			),
		),
		artifact.ByType(artifact.UploadableArchive),
	}
	if len(krew.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(krew.IDs...))
	}

	archives := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return ErrNoArchivesFound
	}

	name, err := tmpl.New(ctx).Apply(krew.Name)
	if err != nil {
		return err
	}
	krew.Name = name

	content, err := buildPlugin(ctx, krew, cl, archives)
	if err != nil {
		return err
	}

	filename := krew.Name + ".yml"
	yamlPath := filepath.Join(ctx.Config.Dist, filename)
	log.WithField("plugin", yamlPath).Info("writing")
	if err := os.WriteFile(yamlPath, []byte(content), 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to write krew plugin: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filename,
		Path: yamlPath,
		Type: artifact.KrewPlugin,
		Extra: map[string]interface{}{
			krewConfigExtra: krew,
		},
	})

	return nil
}

func buildPlugin(ctx *context.Context, krew config.Krew, client client.Client, artifacts []*artifact.Artifact) (string, error) {
	data, err := dataFor(ctx, krew, client, artifacts)
	if err != nil {
		return "", err
	}
	return doBuildPlugin(ctx, data)
}

func doBuildPlugin(ctx *context.Context, data Plugin) (string, error) {
	out, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("krew: failed to marshal yaml: %w", err)
	}
	return string(out), nil
}

func dataFor(ctx *context.Context, cfg config.Krew, cl client.Client, artifacts []*artifact.Artifact) (Plugin, error) {
	result := Plugin{
		APIVersion: apiVersion,
		Kind:       kind,
		Metadata: Metadata{
			Name: cfg.Name,
		},
		Spec: Spec{
			Homepage:         cfg.Homepage,
			Version:          ctx.Version,
			ShortDescription: cfg.ShortDescription,
			Description:      cfg.Description,
			Caveats:          cfg.Caveats,
		},
	}

	for _, art := range artifacts {
		sum, err := art.Checksum("sha256")
		if err != nil {
			return result, err
		}

		if cfg.URLTemplate == "" {
			url, err := cl.ReleaseURLTemplate(ctx)
			if err != nil {
				return result, err
			}
			cfg.URLTemplate = url
		}
		url, err := tmpl.New(ctx).WithArtifact(art, map[string]string{}).Apply(cfg.URLTemplate)
		if err != nil {
			return result, err
		}

		goarch := []string{art.Goarch}
		if art.Goarch == "all" {
			goarch = []string{"amd64", "arm64"}
		}

		for _, arch := range goarch {
			platform := Platform{
				URI:    url,
				Sha256: sum,
				Selector: Selector{
					MatchLabels: MatchLabels{
						Os:   art.Goos,
						Arch: arch,
					},
				},
			}
			for _, bin := range art.ExtraOr(artifact.ExtraBinaries, []string{}).([]string) {
				platform.Bin = bin
				break
			}
			result.Spec.Platforms = append(result.Spec.Platforms, platform)
		}
	}

	return result, nil
}

// Publish krew plugin.
func (Pipe) Publish(ctx *context.Context) error {
	cli, err := client.New(ctx)
	if err != nil {
		return err
	}
	return publishAll(ctx, cli)
}

func publishAll(ctx *context.Context, cli client.Client) error {
	skips := pipe.SkipMemento{}
	for _, plugin := range ctx.Artifacts.Filter(artifact.ByType(artifact.KrewPlugin)).List() {
		err := doPublish(ctx, plugin, cli)
		if err != nil && pipe.IsSkip(err) {
			skips.Remember(err)
			continue
		}
		if err != nil {
			return err
		}
	}
	return skips.Evaluate()
}

func doPublish(ctx *context.Context, plugin *artifact.Artifact, cl client.Client) error {
	cfg := plugin.Extra[krewConfigExtra].(config.Krew)
	var err error
	cl, err = client.NewIfToken(ctx, cl, cfg.Index.Token)
	if err != nil {
		return err
	}

	if strings.TrimSpace(cfg.SkipUpload) == "true" {
		return pipe.Skip("krews.skip_upload is set")
	}

	if strings.TrimSpace(cfg.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' upload, skipping krew publish")
	}

	repo := client.RepoFromRef(cfg.Index)

	gpath := buildPluginPath(pluginsFolder, plugin.Name)
	log.WithField("plugin", gpath).
		WithField("repo", repo.String()).
		Info("pushing")

	msg, err := tmpl.New(ctx).Apply(cfg.CommitMessageTemplate)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(plugin.Path)
	if err != nil {
		return err
	}

	return cl.CreateFile(ctx, cfg.CommitAuthor, repo, content, gpath, msg)
}

func buildPluginPath(folder, filename string) string {
	return path.Join(folder, filename)
}

type Plugin struct {
	APIVersion string   `yaml:"apiVersion,omitempty"`
	Kind       string   `yaml:"kind,omitempty"`
	Metadata   Metadata `yaml:"metadata,omitempty"`
	Spec       Spec     `yaml:"spec,omitempty"`
}

type Metadata struct {
	Name string `yaml:"name,omitempty"`
}

type MatchLabels struct {
	Os   string `yaml:"os,omitempty"`
	Arch string `yaml:"arch,omitempty"`
}

type Selector struct {
	MatchLabels MatchLabels `yaml:"matchLabels,omitempty"`
}

type Platform struct {
	Bin      string   `yaml:"bin,omitempty"`
	URI      string   `yaml:"uri,omitempty"`
	Sha256   string   `yaml:"sha256,omitempty"`
	Selector Selector `yaml:"selector,omitempty"`
}

type Spec struct {
	Version          string     `yaml:"version,omitempty"`
	Platforms        []Platform `yaml:"platforms,omitempty"`
	ShortDescription string     `yaml:"shortDescription,omitempty"`
	Homepage         string     `yaml:"homepage,omitempty"`
	Caveats          string     `yaml:"caveats,omitempty"`
	Description      string     `yaml:"description,omitempty"`
}
