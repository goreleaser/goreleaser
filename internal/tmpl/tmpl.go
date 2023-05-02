// Package tmpl provides templating utilities for goreleaser.
package tmpl

import (
	"bytes"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/build"
	"github.com/goreleaser/goreleaser/pkg/context"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Template holds data that can be applied to a template string.
type Template struct {
	fields Fields
}

// Fields that will be available to the template engine.
type Fields map[string]interface{}

const (
	// general keys.
	projectName     = "ProjectName"
	version         = "Version"
	rawVersion      = "RawVersion"
	tag             = "Tag"
	previousTag     = "PreviousTag"
	branch          = "Branch"
	commit          = "Commit"
	shortCommit     = "ShortCommit"
	fullCommit      = "FullCommit"
	commitDate      = "CommitDate"
	commitTimestamp = "CommitTimestamp"
	gitURL          = "GitURL"
	summary         = "Summary"
	tagSubject      = "TagSubject"
	tagContents     = "TagContents"
	tagBody         = "TagBody"
	releaseURL      = "ReleaseURL"
	isGitDirty      = "IsGitDirty"
	major           = "Major"
	minor           = "Minor"
	patch           = "Patch"
	prerelease      = "Prerelease"
	isSnapshot      = "IsSnapshot"
	isDraft         = "IsDraft"
	env             = "Env"
	date            = "Date"
	now             = "Now"
	timestamp       = "Timestamp"
	modulePath      = "ModulePath"
	releaseNotes    = "ReleaseNotes"
	runtimeK        = "Runtime"

	// artifact-only keys.
	osKey        = "Os"
	amd64        = "Amd64"
	arch         = "Arch"
	arm          = "Arm"
	mips         = "Mips"
	binary       = "Binary"
	artifactName = "ArtifactName"
	artifactExt  = "ArtifactExt"
	artifactPath = "ArtifactPath"

	// build keys.
	name   = "Name"
	ext    = "Ext"
	path   = "Path"
	target = "Target"
)

// New Template.
func New(ctx *context.Context) *Template {
	sv := ctx.Semver
	rawVersionV := fmt.Sprintf("%d.%d.%d", sv.Major, sv.Minor, sv.Patch)

	fields := map[string]interface{}{}
	for k, v := range map[string]interface{}{
		projectName:     ctx.Config.ProjectName,
		modulePath:      ctx.ModulePath,
		version:         ctx.Version,
		rawVersion:      rawVersionV,
		summary:         ctx.Git.Summary,
		tag:             ctx.Git.CurrentTag,
		previousTag:     ctx.Git.PreviousTag,
		branch:          ctx.Git.Branch,
		commit:          ctx.Git.Commit,
		shortCommit:     ctx.Git.ShortCommit,
		fullCommit:      ctx.Git.FullCommit,
		commitDate:      ctx.Git.CommitDate.UTC().Format(time.RFC3339),
		commitTimestamp: ctx.Git.CommitDate.UTC().Unix(),
		gitURL:          ctx.Git.URL,
		isGitDirty:      ctx.Git.Dirty,
		env:             ctx.Env,
		date:            ctx.Date.UTC().Format(time.RFC3339),
		timestamp:       ctx.Date.UTC().Unix(),
		now:             ctx.Date.UTC(),
		major:           ctx.Semver.Major,
		minor:           ctx.Semver.Minor,
		patch:           ctx.Semver.Patch,
		prerelease:      ctx.Semver.Prerelease,
		isSnapshot:      ctx.Snapshot,
		isDraft:         ctx.Config.Release.Draft,
		releaseNotes:    ctx.ReleaseNotes,
		releaseURL:      ctx.ReleaseURL,
		tagSubject:      ctx.Git.TagSubject,
		tagContents:     ctx.Git.TagContents,
		tagBody:         ctx.Git.TagBody,
		runtimeK:        ctx.Runtime,
	} {
		fields[k] = v
	}

	return &Template{
		fields: fields,
	}
}

// WithEnvS overrides template's env field with the given KEY=VALUE list of
// environment variables.
func (t *Template) WithEnvS(envs []string) *Template {
	result := map[string]string{}
	for _, env := range envs {
		k, v, ok := strings.Cut(env, "=")
		if !ok || k == "" {
			continue
		}
		result[k] = v
	}
	return t.WithEnv(result)
}

// WithEnv overrides template's env field with the given environment map.
func (t *Template) WithEnv(e map[string]string) *Template {
	t.fields[env] = e
	return t
}

// WithExtraFields allows to add new more custom fields to the template.
// It will override fields with the same name.
func (t *Template) WithExtraFields(f Fields) *Template {
	for k, v := range f {
		t.fields[k] = v
	}
	return t
}

// WithArtifactReplacements populates Fields from the artifact and replacements.
//
// Deprecated: use WithArtifact instead.
func (t *Template) WithArtifactReplacements(a *artifact.Artifact, replacements map[string]string) *Template {
	t.fields[osKey] = replace(replacements, a.Goos)
	t.fields[arch] = replace(replacements, a.Goarch)
	t.fields[arm] = replace(replacements, a.Goarm)
	t.fields[mips] = replace(replacements, a.Gomips)
	t.fields[amd64] = replace(replacements, a.Goamd64)
	t.fields[binary] = artifact.ExtraOr(*a, binary, t.fields[projectName].(string))
	t.fields[artifactName] = a.Name
	t.fields[artifactExt] = artifact.ExtraOr(*a, artifact.ExtraExt, "")
	t.fields[artifactPath] = a.Path
	return t
}

// WithArtifact populates Fields from the artifact.
func (t *Template) WithArtifact(a *artifact.Artifact) *Template {
	t.fields[osKey] = a.Goos
	t.fields[arch] = a.Goarch
	t.fields[arm] = a.Goarm
	t.fields[mips] = a.Gomips
	t.fields[amd64] = a.Goamd64
	t.fields[binary] = artifact.ExtraOr(*a, binary, t.fields[projectName].(string))
	t.fields[artifactName] = a.Name
	t.fields[artifactExt] = artifact.ExtraOr(*a, artifact.ExtraExt, "")
	t.fields[artifactPath] = a.Path
	return t
}

func (t *Template) WithBuildOptions(opts build.Options) *Template {
	return t.WithExtraFields(buildOptsToFields(opts))
}

func buildOptsToFields(opts build.Options) Fields {
	return Fields{
		target: opts.Target,
		ext:    opts.Ext,
		name:   opts.Name,
		path:   opts.Path,
		osKey:  opts.Goos,
		arch:   opts.Goarch,
		arm:    opts.Goarm,
		mips:   opts.Gomips,
	}
}

// Bool Apply the given string, and converts it to a bool.
func (t *Template) Bool(s string) (bool, error) {
	r, err := t.Apply(s)
	return strings.TrimSpace(strings.ToLower(r)) == "true", err
}

// Apply applies the given string against the Fields stored in the template.
func (t *Template) Apply(s string) (string, error) {
	var out bytes.Buffer
	tmpl, err := template.New("tmpl").
		Option("missingkey=error").
		Funcs(template.FuncMap{
			"replace": strings.ReplaceAll,
			"split":   strings.Split,
			"time": func(s string) string {
				return time.Now().UTC().Format(s)
			},
			"tolower":       strings.ToLower,
			"toupper":       strings.ToUpper,
			"trim":          strings.TrimSpace,
			"trimprefix":    strings.TrimPrefix,
			"trimsuffix":    strings.TrimSuffix,
			"title":         cases.Title(language.English).String,
			"dir":           filepath.Dir,
			"base":          filepath.Base,
			"abs":           filepath.Abs,
			"incmajor":      incMajor,
			"incminor":      incMinor,
			"incpatch":      incPatch,
			"filter":        filter(false),
			"reverseFilter": filter(true),
		}).
		Parse(s)
	if err != nil {
		return "", err
	}

	err = tmpl.Execute(&out, t.fields)
	return out.String(), err
}

type ExpectedSingleEnvErr struct{}

func (e ExpectedSingleEnvErr) Error() string {
	return "expected {{ .Env.VAR_NAME }} only (no plain-text or other interpolation)"
}

// ApplySingleEnvOnly enforces template to only contain a single environment variable
// and nothing else.
func (t *Template) ApplySingleEnvOnly(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return "", nil
	}

	// text/template/parse (lexer) could be used here too,
	// but regexp reduces the complexity and should be sufficient,
	// given the context is mostly discouraging users from bad practice
	// of hard-coded credentials, rather than catch all possible cases
	envOnlyRe := regexp.MustCompile(`^{{\s*\.Env\.[^.\s}]+\s*}}$`)
	if !envOnlyRe.Match([]byte(s)) {
		return "", ExpectedSingleEnvErr{}
	}

	var out bytes.Buffer
	tmpl, err := template.New("tmpl").
		Option("missingkey=error").
		Parse(s)
	if err != nil {
		return "", err
	}

	err = tmpl.Execute(&out, t.fields)
	return out.String(), err
}

// deprecated: will be removed soon.
func replace(replacements map[string]string, original string) string {
	result := replacements[original]
	if result == "" {
		return original
	}
	return result
}

func incMajor(v string) string {
	return prefix(v) + semver.MustParse(v).IncMajor().String()
}

func incMinor(v string) string {
	return prefix(v) + semver.MustParse(v).IncMinor().String()
}

func incPatch(v string) string {
	return prefix(v) + semver.MustParse(v).IncPatch().String()
}

func prefix(v string) string {
	if v != "" && v[0] == 'v' {
		return "v"
	}
	return ""
}

func filter(reverse bool) func(content, exp string) string {
	return func(content, exp string) string {
		re := regexp.MustCompilePOSIX(exp)
		var lines []string
		for _, line := range strings.Split(content, "\n") {
			if reverse && re.MatchString(line) {
				continue
			}
			if !reverse && !re.MatchString(line) {
				continue
			}
			lines = append(lines, line)
		}

		return strings.Join(lines, "\n")
	}
}
