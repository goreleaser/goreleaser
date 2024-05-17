// Package snapcraft implements the Pipe interface providing Snapcraft bindings.
//
//nolint:tagliatelle
package snapcraft

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/ids"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/semerrgroup"
	"github.com/goreleaser/goreleaser/internal/skips"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/internal/yaml"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

const releasesExtra = "releases"

// ErrNoSnapcraft is shown when snapcraft cannot be found in $PATH.
var ErrNoSnapcraft = errors.New("snapcraft not present in $PATH")

// ErrNoDescription is shown when no description provided.
var ErrNoDescription = errors.New("no description provided for snapcraft")

// ErrNoSummary is shown when no summary provided.
var ErrNoSummary = errors.New("no summary provided for snapcraft")

// Metadata to generate the snap package.
type Metadata struct {
	Name          string
	Title         string `yaml:",omitempty"`
	Version       string
	Summary       string
	Description   string
	Icon          string `yaml:",omitempty"`
	Base          string `yaml:",omitempty"`
	License       string `yaml:",omitempty"`
	Grade         string `yaml:",omitempty"`
	Confinement   string `yaml:",omitempty"`
	Architectures []string
	Assumes       []string                  `yaml:",omitempty"`
	Layout        map[string]LayoutMetadata `yaml:",omitempty"`
	Apps          map[string]AppMetadata
	Hooks         map[string]interface{} `yaml:",omitempty"`
	Plugs         map[string]interface{} `yaml:",omitempty"`
}

// AppMetadata for the binaries that will be in the snap package.
// See: https://snapcraft.io/docs/snapcraft-app-and-service-metadata
type AppMetadata struct {
	Command string

	Adapter          string                 `yaml:",omitempty"`
	After            []string               `yaml:",omitempty"`
	Aliases          []string               `yaml:",omitempty"`
	Autostart        string                 `yaml:",omitempty"`
	Before           []string               `yaml:",omitempty"`
	BusName          string                 `yaml:"bus-name,omitempty"`
	CommandChain     []string               `yaml:"command-chain,omitempty"`
	CommonID         string                 `yaml:"common-id,omitempty"`
	Completer        string                 `yaml:",omitempty"`
	Daemon           string                 `yaml:",omitempty"`
	Desktop          string                 `yaml:",omitempty"`
	Environment      map[string]interface{} `yaml:",omitempty"`
	Extensions       []string               `yaml:",omitempty"`
	InstallMode      string                 `yaml:"install-mode,omitempty"`
	Passthrough      map[string]interface{} `yaml:",omitempty"`
	Plugs            []string               `yaml:",omitempty"`
	PostStopCommand  string                 `yaml:"post-stop-command,omitempty"`
	RefreshMode      string                 `yaml:"refresh-mode,omitempty"`
	ReloadCommand    string                 `yaml:"reload-command,omitempty"`
	RestartCondition string                 `yaml:"restart-condition,omitempty"`
	RestartDelay     string                 `yaml:"restart-delay,omitempty"`
	Slots            []string               `yaml:",omitempty"`
	Sockets          map[string]interface{} `yaml:",omitempty"`
	StartTimeout     string                 `yaml:"start-timeout,omitempty"`
	StopCommand      string                 `yaml:"stop-command,omitempty"`
	StopMode         string                 `yaml:"stop-mode,omitempty"`
	StopTimeout      string                 `yaml:"stop-timeout,omitempty"`
	Timer            string                 `yaml:",omitempty"`
	WatchdogTimeout  string                 `yaml:"watchdog-timeout,omitempty"`
}

type LayoutMetadata struct {
	Symlink  string `yaml:",omitempty"`
	Bind     string `yaml:",omitempty"`
	BindFile string `yaml:"bind-file,omitempty"`
	Type     string `yaml:",omitempty"`
}

const defaultNameTemplate = `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}{{ with .Arm }}v{{ . }}{{ end }}{{ with .Mips }}_{{ . }}{{ end }}{{ if not (eq .Amd64 "v1") }}{{ .Amd64 }}{{ end }}`

// Pipe for snapcraft packaging.
type Pipe struct{}

func (Pipe) String() string                           { return "snapcraft packages" }
func (Pipe) ContinueOnError() bool                    { return true }
func (Pipe) Dependencies(_ *context.Context) []string { return []string{"snapcraft"} }
func (Pipe) Skip(ctx *context.Context) bool {
	return skips.Any(ctx, skips.Snapcraft) || len(ctx.Config.Snapcrafts) == 0
}

// Default sets the pipe defaults.
func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("snapcrafts")
	for i := range ctx.Config.Snapcrafts {
		snap := &ctx.Config.Snapcrafts[i]
		if snap.NameTemplate == "" {
			snap.NameTemplate = defaultNameTemplate
		}
		grade, err := tmpl.New(ctx).Apply(snap.Grade)
		if err != nil {
			return err
		}
		snap.Grade = grade
		if snap.Grade == "" {
			snap.Grade = "stable"
		}
		if len(snap.ChannelTemplates) == 0 {
			switch snap.Grade {
			case "devel":
				snap.ChannelTemplates = []string{"edge", "beta"}
			default:
				snap.ChannelTemplates = []string{"edge", "beta", "candidate", "stable"}
			}
		}
		if len(snap.Builds) == 0 {
			for _, b := range ctx.Config.Builds {
				snap.Builds = append(snap.Builds, b.ID)
			}
		}
		ids.Inc(snap.ID)
	}
	return ids.Validate()
}

// Run the pipe.
func (Pipe) Run(ctx *context.Context) error {
	for _, snap := range ctx.Config.Snapcrafts {
		// TODO: deal with pipe.skip?
		if err := doRun(ctx, snap); err != nil {
			return err
		}
	}
	return nil
}

func doRun(ctx *context.Context, snap config.Snapcraft) error {
	tpl := tmpl.New(ctx)
	if err := tpl.ApplyAll(
		&snap.Summary,
		&snap.Description,
		&snap.Disable,
	); err != nil {
		return err
	}
	if snap.Disable == "true" {
		return pipe.Skip("configuration is disabled")
	}
	if snap.Summary == "" && snap.Description == "" {
		return pipe.Skip("no summary nor description were provided")
	}
	if snap.Summary == "" {
		return ErrNoSummary
	}
	if snap.Description == "" {
		return ErrNoDescription
	}
	if _, err := exec.LookPath("snapcraft"); err != nil {
		return ErrNoSnapcraft
	}

	g := semerrgroup.New(ctx.Parallelism)
	for platform, binaries := range ctx.Artifacts.Filter(
		artifact.And(
			artifact.ByGoos("linux"),
			artifact.ByType(artifact.Binary),
			artifact.ByIDs(snap.Builds...),
		),
	).GroupByPlatform() {
		arch := linuxArch(platform)
		if !isValidArch(arch) {
			log.WithField("arch", arch).Warn("ignored unsupported arch")
			continue
		}
		g.Go(func() error {
			return create(ctx, snap, arch, binaries)
		})
	}
	return g.Wait()
}

func isValidArch(arch string) bool {
	// https://snapcraft.io/docs/architectures
	for _, a := range []string{"s390x", "ppc64el", "arm64", "armhf", "i386", "amd64"} {
		if arch == a {
			return true
		}
	}
	return false
}

// Publish packages.
func (Pipe) Publish(ctx *context.Context) error {
	snaps := ctx.Artifacts.Filter(artifact.ByType(artifact.PublishableSnapcraft)).List()
	for _, snap := range snaps {
		if err := push(ctx, snap); err != nil {
			return err
		}
	}
	return nil
}

func create(ctx *context.Context, snap config.Snapcraft, arch string, binaries []*artifact.Artifact) error {
	log := log.WithField("arch", arch)
	folder, err := tmpl.New(ctx).WithArtifact(binaries[0]).Apply(snap.NameTemplate)
	if err != nil {
		return err
	}

	channels, err := processChannelsTemplates(ctx, snap)
	if err != nil {
		return err
	}

	// prime is the directory that then will be compressed to make the .snap package.
	folderDir := filepath.Join(ctx.Config.Dist, folder)
	primeDir := filepath.Join(folderDir, "prime")
	metaDir := filepath.Join(primeDir, "meta")
	// #nosec
	if err = os.MkdirAll(metaDir, 0o755); err != nil {
		return err
	}

	for _, file := range snap.Files {
		if file.Destination == "" {
			file.Destination = file.Source
		}
		if file.Mode == 0 {
			file.Mode = 0o644
		}
		destinationDir := filepath.Join(primeDir, filepath.Dir(file.Destination))
		if err := os.MkdirAll(destinationDir, 0o755); err != nil {
			return fmt.Errorf("failed to create directory '%s': %w", destinationDir, err)
		}
		if err := gio.CopyWithMode(file.Source, filepath.Join(primeDir, file.Destination), os.FileMode(file.Mode)); err != nil {
			return fmt.Errorf("failed to link extra file '%s': %w", file.Source, err)
		}
	}

	file := filepath.Join(primeDir, "meta", "snap.yaml")
	log.WithField("file", file).Debug("creating snap metadata")

	metadata := &Metadata{
		Version:       ctx.Version,
		Summary:       snap.Summary,
		Description:   snap.Description,
		Grade:         snap.Grade,
		Confinement:   snap.Confinement,
		Architectures: []string{arch},
		Layout:        map[string]LayoutMetadata{},
		Apps:          map[string]AppMetadata{},
	}

	if snap.Title != "" {
		metadata.Title = snap.Title
	}

	if snap.Icon != "" {
		metadata.Icon = snap.Icon
	}

	if snap.Base != "" {
		metadata.Base = snap.Base
	}

	if snap.License != "" {
		metadata.License = snap.License
	}

	metadata.Name = ctx.Config.ProjectName
	if snap.Name != "" {
		metadata.Name = snap.Name
	}

	for targetPath, layout := range snap.Layout {
		metadata.Layout[targetPath] = LayoutMetadata{
			Symlink:  layout.Symlink,
			Bind:     layout.Bind,
			BindFile: layout.BindFile,
			Type:     layout.Type,
		}
	}

	// if the user didn't specify any apps then
	// default to the main binary being the command:
	if len(snap.Apps) == 0 {
		name := snap.Name
		if name == "" {
			name = filepath.Base(binaries[0].Name)
		}
		metadata.Apps[name] = AppMetadata{
			Command: filepath.Base(filepath.Base(binaries[0].Name)),
		}
	}

	for _, binary := range binaries {
		// build the binaries and link resources
		destBinaryPath := filepath.Join(primeDir, filepath.Base(binary.Path))
		log.WithField("src", binary.Path).
			WithField("dst", destBinaryPath).
			Debug("copying")

		if err = gio.CopyWithMode(binary.Path, destBinaryPath, 0o555); err != nil {
			return fmt.Errorf("failed to copy binary: %w", err)
		}
	}

	// setup the apps: directive for each binary
	for name, config := range snap.Apps {
		command := name
		if config.Command != "" {
			command = config.Command
		}

		// TODO: test that the correct binary is used in Command
		// See https://github.com/goreleaser/goreleaser/pull/1449
		appMetadata := AppMetadata{
			Command: strings.TrimSpace(strings.Join([]string{
				command,
				config.Args,
			}, " ")),
			Adapter:          config.Adapter,
			After:            config.After,
			Aliases:          config.Aliases,
			Autostart:        config.Autostart,
			Before:           config.Before,
			BusName:          config.BusName,
			CommandChain:     config.CommandChain,
			CommonID:         config.CommonID,
			Completer:        config.Completer,
			Daemon:           config.Daemon,
			Desktop:          config.Desktop,
			Environment:      config.Environment,
			Extensions:       config.Extensions,
			InstallMode:      config.InstallMode,
			Passthrough:      config.Passthrough,
			Plugs:            config.Plugs,
			PostStopCommand:  config.PostStopCommand,
			RefreshMode:      config.RefreshMode,
			ReloadCommand:    config.ReloadCommand,
			RestartCondition: config.RestartCondition,
			RestartDelay:     config.RestartDelay,
			Slots:            config.Slots,
			Sockets:          config.Sockets,
			StartTimeout:     config.StartTimeout,
			StopCommand:      config.StopCommand,
			StopMode:         config.StopMode,
			StopTimeout:      config.StopTimeout,
			Timer:            config.Timer,
			WatchdogTimeout:  config.WatchdogTimeout,
		}

		if config.Completer != "" {
			destCompleterPath := filepath.Join(primeDir, config.Completer)
			if err := os.MkdirAll(filepath.Dir(destCompleterPath), 0o755); err != nil {
				return fmt.Errorf("failed to create folder: %w", err)
			}
			log.WithField("src", config.Completer).
				WithField("dst", destCompleterPath).
				Debug("copy")

			if err := gio.CopyWithMode(config.Completer, destCompleterPath, 0o644); err != nil {
				return fmt.Errorf("failed to copy completer: %w", err)
			}

			appMetadata.Completer = config.Completer
		}

		metadata.Apps[name] = appMetadata
		metadata.Assumes = snap.Assumes
		metadata.Hooks = snap.Hooks
		metadata.Plugs = snap.Plugs
	}

	out, err := yaml.Marshal(metadata)
	if err != nil {
		return err
	}

	log.WithField("file", file).Debugf("writing metadata file")
	if err = os.WriteFile(file, out, 0o644); err != nil { //nolint: gosec
		return err
	}

	snapFile := filepath.Join(ctx.Config.Dist, folder+".snap")
	log.WithField("snap", snapFile).Info("creating")
	/* #nosec */
	cmd := exec.CommandContext(ctx, "snapcraft", "pack", primeDir, "--output", snapFile)
	if out, err = cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to generate snap package: %w: %s", err, string(out))
	}
	if !snap.Publish {
		return nil
	}
	ctx.Artifacts.Add(&artifact.Artifact{
		Type:    artifact.PublishableSnapcraft,
		Name:    folder + ".snap",
		Path:    snapFile,
		Goos:    binaries[0].Goos,
		Goarch:  binaries[0].Goarch,
		Goarm:   binaries[0].Goarm,
		Goamd64: binaries[0].Goamd64,
		Extra: map[string]interface{}{
			releasesExtra: channels,
		},
	})
	return nil
}

const (
	reviewWaitMsg  = `Waiting for previous upload(s) to complete their review process.`
	humanReviewMsg = `A human will soon review your snap`
	needsReviewMsg = `(NEEDS REVIEW)`
)

func push(ctx *context.Context, snap *artifact.Artifact) error {
	log := log.WithField("snap", snap.Name)
	releases := artifact.ExtraOr(*snap, releasesExtra, []string{})
	/* #nosec */
	cmd := exec.CommandContext(ctx, "snapcraft", "upload", "--release="+strings.Join(releases, ","), snap.Path)
	log.WithField("args", cmd.Args).Info("pushing snap")
	if out, err := cmd.CombinedOutput(); err != nil {
		if strings.Contains(string(out), reviewWaitMsg) || strings.Contains(string(out), humanReviewMsg) || strings.Contains(string(out), needsReviewMsg) {
			log.Warn(reviewWaitMsg)
		} else {
			return fmt.Errorf("failed to push %s package: %w: %s", snap.Path, err, string(out))
		}
	}
	snap.Type = artifact.Snapcraft
	ctx.Artifacts.Add(snap)
	return nil
}

func processChannelsTemplates(ctx *context.Context, snap config.Snapcraft) ([]string, error) {
	//nolint:prealloc
	var channels []string
	for _, channeltemplate := range snap.ChannelTemplates {
		channel, err := tmpl.New(ctx).Apply(channeltemplate)
		if err != nil {
			return nil, fmt.Errorf("failed to execute channel template '%s': %w", err, err)
		}
		if channel == "" {
			continue
		}

		channels = append(channels, channel)
	}

	return channels, nil
}

var archToSnap = map[string]string{
	"386":     "i386",
	"arm":     "armhf",
	"arm6":    "armhf",
	"arm7":    "armhf",
	"ppc64le": "ppc64el",
}

// TODO: write tests for this
func linuxArch(key string) string {
	// XXX: list of all linux arches: `go tool dist list | grep linux`
	arch := strings.TrimPrefix(key, "linux")
	for _, suffix := range []string{
		"hardfloat",
		"softfloat",
		"v1",
		"v2",
		"v3",
		"v4",
	} {
		arch = strings.TrimSuffix(arch, suffix)
	}

	if got, ok := archToSnap[arch]; ok {
		return got
	}

	return arch
}
