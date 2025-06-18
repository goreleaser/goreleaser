// Package tmpl provides templating utilities for goreleaser.
package tmpl

import (
	"bytes"
	"fmt"
	"maps"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/pkg/build"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Template holds data that can be applied to a template string.
type Template struct {
	fields Fields
}

// Fields that will be available to the template engine.
type Fields map[string]any

// Template fields names used in build targets and more.
const (
	KeyOS      = "Os"
	KeyArch    = "Arch"
	KeyAmd64   = "Amd64"
	Key386     = "I386"
	KeyArm     = "Arm"
	KeyArm64   = "Arm64"
	KeyMips    = "Mips"
	KeyPpc64   = "Ppc64"
	KeyRiscv64 = "Riscv64"
)

// general keys.
const (
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
	isGitClean      = "IsGitClean"
	gitTreeState    = "GitTreeState"
	major           = "Major"
	minor           = "Minor"
	patch           = "Patch"
	prerelease      = "Prerelease"
	isSnapshot      = "IsSnapshot"
	isSingleTarget  = "IsSingleTarget"
	isNightly       = "IsNightly"
	isDraft         = "IsDraft"
	env             = "Env"
	date            = "Date"
	now             = "Now"
	timestamp       = "Timestamp"
	modulePath      = "ModulePath"
	releaseNotes    = "ReleaseNotes"
	runtimeK        = "Runtime"
)

// artifact-only keys.
const (
	binary       = "Binary"
	artifactName = "ArtifactName"
	artifactExt  = "ArtifactExt"
	artifactPath = "ArtifactPath"
)

// build keys.
const (
	name   = "Name"
	ext    = "Ext"
	path   = "Path"
	target = "Target"
)

// New Template.
func New(ctx *context.Context) *Template {
	sv := ctx.Semver
	rawVersionV := fmt.Sprintf("%d.%d.%d", sv.Major, sv.Minor, sv.Patch)
	treeState := "clean"
	if ctx.Git.Dirty {
		treeState = "dirty"
	}

	fields := map[string]any{}
	maps.Copy(fields, map[string]any{
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
		isGitClean:      !ctx.Git.Dirty,
		gitTreeState:    treeState,
		env:             ctx.Env,
		date:            ctx.Date.UTC().Format(time.RFC3339),
		timestamp:       ctx.Date.UTC().Unix(),
		now:             ctx.Date.UTC(),
		major:           ctx.Semver.Major,
		minor:           ctx.Semver.Minor,
		patch:           ctx.Semver.Patch,
		prerelease:      ctx.Semver.Prerelease,
		isSnapshot:      ctx.Snapshot,
		isSingleTarget:  ctx.SingleTarget,
		isNightly:       false,
		isDraft:         ctx.Config.Release.Draft,
		releaseNotes:    ctx.ReleaseNotes,
		releaseURL:      ctx.ReleaseURL,
		tagSubject:      ctx.Git.TagSubject,
		tagContents:     ctx.Git.TagContents,
		tagBody:         ctx.Git.TagBody,
		runtimeK:        ctx.Runtime,
	})

	return &Template{
		fields: fields,
	}
}

// SetEnv adds a single environment variable into the template env.
func (t *Template) SetEnv(single string) *Template {
	k, v, ok := strings.Cut(single, "=")
	if !ok || k == "" {
		return t
	}
	// TODO: handle delete?
	tt := t.copying()
	envs := tt.fields[env].(context.Env)
	envs[k] = v
	tt.fields[env] = envs
	return tt
}

// WithExtraFields allows to add new more custom fields to the template.
// It will override fields with the same name.
func (t *Template) WithExtraFields(f Fields) *Template {
	tt := t.copying()
	maps.Copy(tt.fields, f)
	return tt
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
	return t.WithExtraFields(Fields{
		env: context.Env(e),
	})
}

// WithArtifact populates Fields from the artifact.
func (t *Template) WithArtifact(a *artifact.Artifact) *Template {
	return t.WithExtraFields(Fields{
		KeyOS:        a.Goos,
		KeyArch:      a.Goarch,
		KeyAmd64:     a.Goamd64,
		Key386:       a.Go386,
		KeyArm:       a.Goarm,
		KeyArm64:     a.Goarm64,
		KeyMips:      a.Gomips,
		KeyPpc64:     a.Goppc64,
		KeyRiscv64:   a.Goriscv64,
		target:       a.Target,
		binary:       artifact.ExtraOr(*a, binary, t.fields[projectName].(string)),
		artifactName: a.Name,
		artifactExt:  a.Ext(),
		artifactPath: a.Path,
	})
}

func (t *Template) WithBuildOptions(opts build.Options) *Template {
	return t.WithExtraFields(buildOptsToFields(opts))
}

func buildOptsToFields(opts build.Options) Fields {
	f := Fields{
		target: opts.Target.String(),
		ext:    opts.Ext,
		name:   opts.Name,
		path:   opts.Path,

		// set them all to empty, which should prevent breaking templates.
		// the .Fields() call will override whichever values are actually
		// available.
		KeyOS:      "",
		KeyArch:    "",
		KeyAmd64:   "",
		Key386:     "",
		KeyArm:     "",
		KeyArm64:   "",
		KeyMips:    "",
		KeyPpc64:   "",
		KeyRiscv64: "",
	}
	for k, v := range opts.Target.Fields() {
		f[k] = v
	}
	return f
}

// Bool Apply the given string, and converts it to a bool.
func (t *Template) Bool(s string) (bool, error) {
	r, err := t.Apply(s)
	return strings.TrimSpace(strings.ToLower(r)) == "true", err
}

// SliceOpt is a [Slice] option.
type SliceOpt func(*sliceOptions)

// NonEmpty filters out empty items.
func NonEmpty() SliceOpt {
	return func(o *sliceOptions) {
		o.filtering = func(s string) bool { return s != "" }
	}
}

// WithPrefix pretend a prefix to every item.
func WithPrefix(prefix string) SliceOpt {
	return func(o *sliceOptions) {
		o.mapping = func(s string) string { return prefix + s }
	}
}

type sliceOptions struct {
	filtering func(string) bool
	mapping   func(string) string
}

// Slice applies to all items in the given input.
func (t *Template) Slice(in []string, opts ...SliceOpt) ([]string, error) {
	var opt sliceOptions
	for _, option := range opts {
		option(&opt)
	}
	var out []string
	for _, s := range in {
		applied, err := t.Apply(s)
		if err != nil {
			return nil, err
		}
		if opt.filtering != nil && !opt.filtering(applied) {
			continue
		}
		if opt.mapping != nil {
			applied = opt.mapping(applied)
		}
		out = append(out, applied)
	}
	return out, nil
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
			"contains":       strings.Contains,
			"tolower":        strings.ToLower,
			"toupper":        strings.ToUpper,
			"trim":           strings.TrimSpace,
			"trimprefix":     strings.TrimPrefix,
			"trimsuffix":     strings.TrimSuffix,
			"title":          cases.Title(language.English).String,
			"dir":            filepath.Dir,
			"base":           filepath.Base,
			"abs":            filepath.Abs,
			"incmajor":       incMajor,
			"incminor":       incMinor,
			"incpatch":       incPatch,
			"filter":         filter(false),
			"reverseFilter":  filter(true),
			"mdv2escape":     mdv2Escape,
			"envOrDefault":   t.envOrDefault,
			"isEnvSet":       t.isEnvSet,
			"map":            makemap,
			"indexOrDefault": indexOrDefault,
			"urlPathEscape":  url.PathEscape,
			"blake2b":        checksum("blake2b"),
			"blake2s":        checksum("blake2s"),
			"crc32":          checksum("crc32"),
			"md5":            checksum("md5"),
			"sha224":         checksum("sha224"),
			"sha384":         checksum("sha384"),
			"sha256":         checksum("sha256"),
			"sha1":           checksum("sha1"),
			"sha512":         checksum("sha512"),
			"sha3_224":       checksum("sha3-224"),
			"sha3_384":       checksum("sha3-384"),
			"sha3_256":       checksum("sha3-256"),
			"sha3_512":       checksum("sha3-512"),
		}).
		Parse(s)
	if err != nil {
		return "", newTmplError(s, err)
	}

	err = tmpl.Execute(&out, t.fields)
	return out.String(), newTmplError(s, err)
}

// ApplyAll applies all the given strings against the Fields stored in the
// template. Application stops as soon as an error is encountered.
func (t *Template) ApplyAll(sps ...*string) error {
	for _, sp := range sps {
		s := *sp
		result, err := t.Apply(s)
		if err != nil {
			return newTmplError(s, err)
		}
		*sp = result
	}
	return nil
}

// ApplySlice applies the template to all items in a slice.
func (t *Template) ApplySlice(in *[]string) error {
	for i, s := range *in {
		ss, err := t.Apply(s)
		if err != nil {
			return newTmplError(s, err)
		}
		(*in)[i] = ss
	}
	return nil
}

func (t *Template) isEnvSet(name string) bool {
	s, ok := t.fields[env].(context.Env)[name]
	return ok && s != ""
}

func (t *Template) envOrDefault(name, value string) string {
	s, ok := t.fields[env].(context.Env)[name]
	if !ok {
		return value
	}
	return s
}

func (t *Template) copying() *Template {
	tpl := &Template{
		fields: Fields{},
	}
	maps.Copy(tpl.fields, t.fields)
	return tpl
}

type ExpectedSingleEnvErr struct{}

func (e ExpectedSingleEnvErr) Error() string {
	return "expected {{ .Env.VAR_NAME }} only (no plain-text or other interpolation)"
}

var envOnlyRe = regexp.MustCompile(`^{{\s*\.Env\.[^.\s}]+\s*}}$`)

// ApplySingleEnvOnly enforces template to only contain a single environment variable
// and nothing else.
func (t *Template) ApplySingleEnvOnly(s string) (string, error) {
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return "", nil
	}

	var out bytes.Buffer
	tmpl, err := template.New("tmpl").
		Option("missingkey=error").
		Parse(s)
	if err != nil {
		return "", newTmplError(s, err)
	}

	// text/template/parse (lexer) could be used here too,
	// but regexp reduces the complexity and should be sufficient,
	// given the context is mostly discouraging users from bad practice
	// of hard-coded credentials, rather than catch all possible cases
	if !envOnlyRe.MatchString(s) {
		return "", ExpectedSingleEnvErr{}
	}

	err = tmpl.Execute(&out, t.fields)
	return out.String(), err
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
		for line := range strings.SplitSeq(content, "\n") {
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

var mdv2EscapeReplacer = strings.NewReplacer(
	"_", "\\_",
	"*", "\\*",
	"[", "\\[",
	"]", "\\]",
	"(", "\\(",
	")", "\\)",
	"~", "\\~",
	"`", "\\`",
	">", "\\>",
	"#", "\\#",
	"+", "\\+",
	"-", "\\-",
	"=", "\\=",
	"|", "\\|",
	"{", "\\{",
	"}", "\\}",
	".", "\\.",
	"!", "\\!",
)

func mdv2Escape(s string) string {
	return mdv2EscapeReplacer.Replace(s)
}

func makemap(kvs ...string) (map[string]string, error) {
	if len(kvs)%2 != 0 {
		return nil, fmt.Errorf("map expects even number of arguments, got %d", len(kvs))
	}
	m := make(map[string]string)
	for i := 0; i < len(kvs); i += 2 {
		m[kvs[i]] = kvs[i+1]
	}
	return m, nil
}

func indexOrDefault(m map[string]string, name, value string) string {
	s, ok := m[name]
	if ok {
		return s
	}
	return value
}

func checksum(algorithm string) func(string) (string, error) {
	return func(file string) (string, error) {
		artifact := artifact.Artifact{
			Path: file,
		}

		return artifact.Checksum(algorithm)
	}
}
