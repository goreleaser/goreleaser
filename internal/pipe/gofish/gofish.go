package gofish

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/client"
	"github.com/goreleaser/goreleaser/internal/commitauthor"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const (
	goFishConfigExtra = "GoFishConfig"
	foodFolder        = "Food"
)

var ErrNoArchivesFound = errors.New("no linux/macos/windows archives found")

// Pipe for goFish deployment.
type Pipe struct{}

func (Pipe) String() string                 { return "gofish fish food cookbook" }
func (Pipe) Skip(ctx *context.Context) bool { return len(ctx.Config.Rigs) == 0 }

func (Pipe) Default(ctx *context.Context) error {
	for i := range ctx.Config.Rigs {
		goFish := &ctx.Config.Rigs[i]

		goFish.CommitAuthor = commitauthor.Default(goFish.CommitAuthor)
		if goFish.CommitMessageTemplate == "" {
			goFish.CommitMessageTemplate = "GoFish fish food update for {{ .ProjectName }} version {{ .Tag }}"
		}
		if goFish.Name == "" {
			goFish.Name = ctx.Config.ProjectName
		}
		if goFish.Goarm == "" {
			goFish.Goarm = "6"
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
	for _, goFish := range ctx.Config.Rigs {
		err := doRun(ctx, goFish, cli)
		if err != nil {
			return err
		}
	}
	return nil
}

func doRun(ctx *context.Context, goFish config.GoFish, cl client.Client) error {
	if goFish.Rig.Name == "" {
		return pipe.Skip("Rigs rig name is not set")
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
				artifact.ByGoarm(goFish.Goarm),
			),
		),
		artifact.Or(
			artifact.ByType(artifact.UploadableArchive),
			artifact.ByType(artifact.UploadableBinary),
		),
		artifact.OnlyReplacingUnibins,
	}
	if len(goFish.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(goFish.IDs...))
	}

	archives := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(archives) == 0 {
		return ErrNoArchivesFound
	}

	name, err := tmpl.New(ctx).Apply(goFish.Name)
	if err != nil {
		return err
	}
	goFish.Name = name

	content, err := buildFood(ctx, goFish, cl, archives)
	if err != nil {
		return err
	}

	filename := goFish.Name + ".lua"
	luaPath := filepath.Join(ctx.Config.Dist, filename)
	log.WithField("food", luaPath).Info("writing")
	if err := os.WriteFile(luaPath, []byte(content), 0o644); err != nil { //nolint: gosec
		return fmt.Errorf("failed to write gofish food: %w", err)
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filename,
		Path: luaPath,
		Type: artifact.GoFishRig,
		Extra: map[string]interface{}{
			goFishConfigExtra: goFish,
		},
	})

	return nil
}

func buildFood(ctx *context.Context, goFish config.GoFish, client client.Client, artifacts []*artifact.Artifact) (string, error) {
	data, err := dataFor(ctx, goFish, client, artifacts)
	if err != nil {
		return "", err
	}
	return doBuildFood(ctx, data)
}

func doBuildFood(ctx *context.Context, data templateData) (string, error) {
	t, err := template.
		New(data.Name).
		Parse(foodTemplate)
	if err != nil {
		return "", err
	}
	var out bytes.Buffer
	if err := t.Execute(&out, data); err != nil {
		return "", err
	}

	content, err := tmpl.New(ctx).Apply(out.String())
	if err != nil {
		return "", err
	}
	out.Reset()

	// Sanitize the template output and get rid of trailing whitespace.
	var (
		r = strings.NewReader(content)
		s = bufio.NewScanner(r)
	)
	for s.Scan() {
		l := strings.TrimRight(s.Text(), " ")
		_, _ = out.WriteString(l)
		_ = out.WriteByte('\n')
	}
	if err := s.Err(); err != nil {
		return "", err
	}

	return out.String(), nil
}

func dataFor(ctx *context.Context, cfg config.GoFish, cl client.Client, artifacts []*artifact.Artifact) (templateData, error) {
	result := templateData{
		Name:     cfg.Name,
		Desc:     cfg.Description,
		Homepage: cfg.Homepage,
		Version:  ctx.Version,
		License:  cfg.License,
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
			releasePackage := releasePackage{
				DownloadURL: url,
				SHA256:      sum,
				OS:          art.Goos,
				Arch:        arch,
				Binaries:    []binary{},
			}
			switch art.Type {
			case artifact.UploadableArchive:
				for _, bin := range art.ExtraOr(artifact.ExtraBinaries, []string{}).([]string) {
					releasePackage.Binaries = append(releasePackage.Binaries, binary{
						Name:   bin,
						Target: bin,
					})
				}
			case artifact.UploadableBinary:
				releasePackage.Binaries = append(releasePackage.Binaries, binary{
					Name:   art.Name,
					Target: art.ExtraOr(artifact.ExtraBinary, art.Name).(string),
				})
			}
			result.ReleasePackages = append(result.ReleasePackages, releasePackage)
		}
	}

	return result, nil
}

// Publish gofish rig.
func (Pipe) Publish(ctx *context.Context) error {
	cli, err := client.New(ctx)
	if err != nil {
		return err
	}
	return publishAll(ctx, cli)
}

func publishAll(ctx *context.Context, cli client.Client) error {
	skips := pipe.SkipMemento{}
	for _, rig := range ctx.Artifacts.Filter(artifact.ByType(artifact.GoFishRig)).List() {
		err := doPublish(ctx, rig, cli)
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

func doPublish(ctx *context.Context, food *artifact.Artifact, cl client.Client) error {
	rig := food.Extra[goFishConfigExtra].(config.GoFish)
	var err error
	cl, err = client.NewIfToken(ctx, cl, rig.Rig.Token)
	if err != nil {
		return err
	}

	if strings.TrimSpace(rig.SkipUpload) == "true" {
		return pipe.Skip("rig.skip_upload is set")
	}

	if strings.TrimSpace(rig.SkipUpload) == "auto" && ctx.Semver.Prerelease != "" {
		return pipe.Skip("prerelease detected with 'auto' upload, skipping gofish publish")
	}

	repo := client.RepoFromRef(rig.Rig)

	gpath := buildFoodPath(foodFolder, food.Name)
	log.WithField("food", gpath).
		WithField("repo", repo.String()).
		Info("pushing")

	msg, err := tmpl.New(ctx).Apply(rig.CommitMessageTemplate)
	if err != nil {
		return err
	}

	author, err := commitauthor.Get(ctx, rig.CommitAuthor)
	if err != nil {
		return err
	}

	content, err := os.ReadFile(food.Path)
	if err != nil {
		return err
	}

	return cl.CreateFile(ctx, author, repo, content, gpath, msg)
}

func buildFoodPath(folder, filename string) string {
	return path.Join(folder, filename)
}
