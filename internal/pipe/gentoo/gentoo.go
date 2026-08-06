// Package gentoo provides a Pipe that builds and publishes gentoo ebuilds.
package gentoo

import (
	"bytes"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	_ "embed"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	pathlib "path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"

	"github.com/Masterminds/semver/v3"
	"golang.org/x/crypto/blake2b"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/internal/commitauthor"
	"github.com/goreleaser/goreleaser/v2/internal/extrafiles"
	"github.com/goreleaser/goreleaser/v2/internal/ids"
	"github.com/goreleaser/goreleaser/v2/internal/tmpl"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

const (
	ebuildExtra     = "GentooConfig"
	ebuildPathExtra = "GentooPath"
	ebuildMetaCache = "GentooMetaCache"
)

//go:embed templates/ebuild.tmpl
var ebuildTemplate string

//go:embed templates/md5-cache.tmpl
var metaCacheTemplate string

//go:embed templates/metadata.xml.tmpl
var metadataXMLTemplate string

type installData struct {
	Source   string
	Target   string
	Keywords []string
}

type archData struct {
	Keyword string
	File    string
	URI     string
}

type installGroup struct {
	Keywords []string
	Installs []installData
}

type installItemData struct {
	Source string
	Target string
	Dir    string
	Base   string
	Use    []string
}

type parsedGentooVersion struct {
	version  *semver.Version
	revision int
}

func (v *parsedGentooVersion) GreaterThan(other *parsedGentooVersion) bool {
	if v.version.Equal(other.version) {
		return v.revision > other.revision
	}
	return v.version.GreaterThan(other.version)
}

func parseGentooVersion(n, prefix string) *parsedGentooVersion {
	vStr := strings.TrimSuffix(strings.TrimPrefix(n, prefix), ".ebuild")
	var rev int
	if idx := strings.LastIndex(vStr, "-r"); idx != -1 {
		if parsedRev, err := strconv.Atoi(vStr[idx+2:]); err == nil {
			rev = parsedRev
			vStr = vStr[:idx]
		}
	}
	vStr = strings.ReplaceAll(vStr, "_", "-")
	v, err := semver.NewVersion(vStr)
	if err != nil {
		return nil
	}
	return &parsedGentooVersion{
		version:  v,
		revision: rev,
	}
}

func getVersionBucket(v *parsedGentooVersion) string {
	pr := v.version.Prerelease()
	switch {
	case strings.Contains(pr, "rc"):
		return "rc"
	case strings.Contains(pr, "beta"):
		return "beta"
	case strings.Contains(pr, "alpha"):
		return "alpha"
	default:
		return "stable"
	}
}

type ebuildDeleter struct {
	dir            string
	category       string
	metaCache      bool
	files          *[]client.RepoFile
	deletedEbuilds *[]string
}

func (d *ebuildDeleter) Delete(ebuildName string) {
	*d.files = append(*d.files, client.RepoFile{Path: pathlib.Join(d.dir, ebuildName), Delete: true})
	*d.deletedEbuilds = append(*d.deletedEbuilds, ebuildName)
	if !d.metaCache {
		return
	}
	md5Name := strings.TrimSuffix(ebuildName, ".ebuild")
	md5CachePath := pathlib.Join("metadata", "md5-cache", d.category, md5Name)
	*d.files = append(*d.files, client.RepoFile{Path: md5CachePath, Delete: true})
}

// Pipe builds and publishes gentoo ebuilds.
type Pipe struct{}

func (Pipe) String() string        { return "gentoo ebuild" }
func (Pipe) ContinueOnError() bool { return true }
func (Pipe) Skip(ctx *context.Context) bool {
	return len(ctx.Config.Gentoos) == 0
}

func (Pipe) Default(ctx *context.Context) error {
	ids := ids.New("gentoo_overlay")
	for i := range ctx.Config.Gentoos {
		g := &ctx.Config.Gentoos[i]
		g.CommitAuthor = commitauthor.Default(g.CommitAuthor)
		if g.ID == "" {
			g.ID = "default"
		}
		if !g.Bin {
			return errors.New("gentoo.bin must be true")
		}
		if g.CommitMessageTemplate == "" {
			g.CommitMessageTemplate = "{{ .ProjectName }}: bump to {{ .Tag }}"
		}
		if g.Type == "bin" || g.Type == "" {
			if g.Bindir == "" {
				g.Bindir = "/opt/bin"
			}
		} else {
			if g.Bindir == "" {
				g.Bindir = "/usr/bin"
			}
		}
		if g.Type == "" {
			g.Type = "bin"
		}
		if len(g.Keywords) == 0 {
			g.Keywords = config.StringArray{"~amd64"}
		}
		if g.License == "" {
			return errors.New("gentoo.license is required")
		}
		if g.KeepVersions < 0 {
			return errors.New("gentoo.keep_versions must be greater than or equal to 0")
		}
		if g.VersionRetentionStrategy != "" && g.VersionRetentionStrategy != "keep_latest" && g.VersionRetentionStrategy != "keep_prereleases" {
			return fmt.Errorf("gentoo.version_retention_strategy %q is not valid, must be one of [keep_latest, keep_prereleases]", g.VersionRetentionStrategy)
		}
		if g.KeepVersions > 0 && g.VersionRetentionStrategy == "" {
			return errors.New("gentoo.version_retention_strategy must be provided if gentoo.keep_versions > 0")
		}
		if g.Name == "" {
			g.Name = ctx.Config.ProjectName
		}
		if g.Path == "" {
			g.Path = defaultPath(g.Name, g.Category, g.Type)
			if g.Category == "" {
				log.Warnf("no gentoo category configured for %q; defaulting path to %q", g.Name, filepath.ToSlash(g.Path))
			}
		} else if !hasCategory(g.Path) {
			log.Warnf("gentoo.path %q does not include a category/package path; Gentoo ebuild paths usually look like %q", g.Path, filepath.ToSlash(defaultPath(g.Name, g.Category, g.Type)))
		}
		ids.Inc(g.ID)
	}
	return ids.Validate()
}

func (Pipe) Run(ctx *context.Context) error {
	cli, err := client.NewReleaseClient(ctx)
	if err != nil {
		return err
	}
	for _, cfg := range ctx.Config.Gentoos {
		if err := doRun(ctx, cfg, cli); err != nil {
			return err
		}
	}
	return nil
}

var gentooPrereleaseRe = regexp.MustCompile(`-(alpha|beta|pre|rc|p)[.\-]?(\d*)`)

func gentooVersion(v string) string {
	return gentooPrereleaseRe.ReplaceAllString(v, "_${1}${2}")
}

type extraFilesProcessor struct {
	cfg        config.Gentoo
	arches     []*artifact.Artifact
	extraFiles map[string]string
}

func (v *extraFilesProcessor) inArchives(fileName string) bool {
	if len(v.arches) == 0 {
		return false
	}
	for _, art := range v.arches {
		found := false
		if files, ok := art.Extra[artifact.ExtraFiles].([]string); ok {
			for _, f := range files {
				if archiveDestination(*art, f) == normalizeArchivePath(fileName) {
					found = true
					break
				}
			}
		}
		if !found {
			if bins, ok := art.Extra[artifact.ExtraBinaries].([]string); ok {
				for _, b := range bins {
					if archiveDestination(*art, b) == normalizeArchivePath(fileName) {
						found = true
						break
					}
				}
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func archiveDestination(art artifact.Artifact, destination string) string {
	return normalizeArchivePath(filepath.Join(artifact.ExtraOr(art, artifact.ExtraWrappedIn, ""), destination))
}

func normalizeArchivePath(path string) string {
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(path)), "./")
}

func newExtraFilesProcessor(cfg config.Gentoo, arches []*artifact.Artifact, extraFiles map[string]string) *extraFilesProcessor {
	return &extraFilesProcessor{
		cfg:        cfg,
		arches:     arches,
		extraFiles: extraFiles,
	}
}

func (v *extraFilesProcessor) Filter() error {
	for name, src := range v.extraFiles {
		if v.inArchives(name) {
			log.Warnf("file %s is already in all archives, skipping upload to Gentoo files/ directory", name)
			delete(v.extraFiles, name)
			continue
		}
		if err := v.validate(name, src); err != nil {
			return err
		}
	}
	return nil
}

func (v *extraFilesProcessor) validate(name, src string) error {
	info, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("failed to stat extra file %s: %w", name, err)
	}
	if !v.cfg.SkipFilesValidation {
		if info.Size() > 20*1024 {
			return fmt.Errorf("extra file %s is larger than 20KB. Gentoo policy forbids large files in the files/ directory. Please add it to a release asset instead", name)
		}

		f, err := os.Open(src)
		if err != nil {
			return fmt.Errorf("failed to open extra file %s: %w", name, err)
		}
		defer f.Close()
		buf := make([]byte, 512)
		n, err := f.Read(buf)
		if err != nil && err != io.EOF {
			return fmt.Errorf("failed to read extra file %s: %w", name, err)
		}
		if bytes.IndexByte(buf[:n], 0) != -1 {
			return fmt.Errorf("extra file %s appears to be a binary file. Gentoo policy forbids binary files in the files/ directory", name)
		}
	}
	return nil
}

func (v *extraFilesProcessor) processStringArray(arr []string) []string {
	var out []string
	for _, s := range arr {
		if _, ok := v.extraFiles[s]; ok {
			out = append(out, "${FILESDIR}/"+strings.TrimPrefix(s, "files/"))
		} else {
			out = append(out, s)
		}
	}
	return out
}

func (v *extraFilesProcessor) buildInstallItems(cfgItems []config.GentooInstallItem) []installItemData {
	var items []installItemData
	for _, d := range cfgItems {
		src := d.Src
		if _, ok := v.extraFiles[d.Src]; ok {
			src = "${FILESDIR}/" + strings.TrimPrefix(d.Src, "files/")
		}
		items = append(items, installItemData{
			Source: src,
			Target: d.Dst,
			Dir:    pathlib.Dir(filepath.ToSlash(d.Dst)),
			Base:   pathlib.Base(filepath.ToSlash(d.Dst)),
			Use:    d.Use,
		})
	}
	return items
}

func doRun(ctx *context.Context, cfg config.Gentoo, cl client.ReleaseURLTemplater) error {
	tp := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"Version":  gentooVersion(ctx.Version),
		"Name":     cfg.Name,
		"Category": cfg.Category,
	})
	if err := tp.ApplyAll(&cfg.Name, &cfg.Category, &cfg.Path, &cfg.Description, &cfg.Homepage, &cfg.BugsTo, &cfg.License); err != nil {
		return err
	}
	var err error
	cfg.Repository, err = client.TemplateRef(tp.Apply, cfg.Repository)
	if err != nil {
		return err
	}
	if strings.TrimSpace(cfg.Path) == "" {
		return errors.New("gentoo.path is required and must include the category/package ebuild path")
	}
	if !isValidGentooPath(cfg.Path) {
		return fmt.Errorf("gentoo.path %q must be a relative category/package/file.ebuild path", cfg.Path)
	}

	path := filepath.Join(ctx.Config.Dist, "gentoo", cfg.ID, cfg.Path)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	filters := []artifact.Filter{
		artifact.ByGoos("linux"),
		artifact.ByType(artifact.UploadableArchive),
		artifact.OnlyReplacingUnibins,
	}
	if len(cfg.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(cfg.IDs...))
	}

	arches := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	if len(arches) == 0 {
		return errors.New("no linux archives found")
	}

	var archInfos []archData
	keywordSet := map[string]struct{}{}

	uriTemplate, err := cl.ReleaseURLTemplate(ctx)
	if err != nil {
		return err
	}

	for _, art := range arches {
		url, err := tmpl.New(ctx).WithArtifact(art).Apply(uriTemplate)
		if err != nil {
			return err
		}
		kw := gentooArch(art.Goarch)
		archInfos = append(archInfos, archData{
			Keyword: kw,
			File:    art.Name,
			URI:     url,
		})
		keywordSet["~"+kw] = struct{}{}
	}

	keywords := cfg.Keywords
	if len(keywords) == 0 {
		keywords = make(config.StringArray, 0, len(keywordSet))
		for k := range keywordSet {
			keywords = append(keywords, k)
		}
		slices.Sort(keywords)
	}

	keywordMap := make(map[string]bool)
	for _, art := range archInfos {
		keywordMap[art.Keyword] = true
	}
	keywordsList := make([]string, 0, len(keywordMap))
	for kw := range keywordMap {
		keywordsList = append(keywordsList, kw)
	}
	slices.Sort(keywordsList)

	installGroups := []installGroup{
		{
			Keywords: keywordsList,
			Installs: []installData{
				{
					Source:   cfg.Name,
					Target:   cfg.Name,
					Keywords: keywordsList,
				},
			},
		},
	}

	extraInstall, err := tp.Apply(cfg.ExtraInstall)
	if err != nil {
		return err
	}

	extraFiles, err := extrafiles.Find(ctx, cfg.Files)
	if err != nil {
		return err
	}

	ef := newExtraFilesProcessor(cfg, arches, extraFiles)

	if err := ef.Filter(); err != nil {
		return err
	}
	useFlags := gentooUseFlags(cfg)

	data := struct {
		Name          string
		Description   string
		Homepage      string
		License       string
		Keywords      string
		Bindir        string
		ExtraInstall  string
		Archs         []archData
		InstallGroups []installGroup
		UseFlags      []config.GentooUseFlag
		Dobin         []installItemData
		Doconfd       []installItemData
		Dodir         []string
		Dodoc         []string
		Doenvd        []installItemData
		Doexe         []installItemData
		Doheader      []installItemData
		Doinitd       []installItemData
		Doins         []installItemData
		Doman         []string
		Dosbin        []installItemData
		Dosym         []installItemData
		Systemd       []installItemData
	}{
		Name:          cfg.Name,
		Description:   cfg.Description,
		Homepage:      cfg.Homepage,
		License:       cfg.License,
		Keywords:      strings.Join(keywords, " "),
		Bindir:        cfg.Bindir,
		ExtraInstall:  extraInstall,
		Archs:         archInfos,
		InstallGroups: installGroups,
		UseFlags:      useFlags,
		Dobin:         ef.buildInstallItems(cfg.Dobin),
		Doconfd:       ef.buildInstallItems(cfg.Doconfd),
		Dodir:         cfg.Dodir,
		Dodoc:         ef.processStringArray(cfg.Dodoc),
		Doenvd:        ef.buildInstallItems(cfg.Doenvd),
		Doexe:         ef.buildInstallItems(cfg.Doexe),
		Doheader:      ef.buildInstallItems(cfg.Doheader),
		Doinitd:       ef.buildInstallItems(cfg.Doinitd),
		Doins:         ef.buildInstallItems(cfg.Doins),
		Doman:         ef.processStringArray(cfg.Doman),
		Dosbin:        ef.buildInstallItems(cfg.Dosbin),
		Dosym:         ef.buildInstallItems(cfg.Dosym),
		Systemd:       ef.buildInstallItems(cfg.Systemd),
	}

	for _, sym := range data.Dosym {
		if sym.Target == "" {
			return errors.New("gentoo.dosym requires a destination")
		}
	}

	var buf bytes.Buffer
	if err := template.Must(template.New("ebuild").Parse(ebuildTemplate)).Execute(&buf, data); err != nil {
		return err
	}
	content := buf.String()

	log.WithField("ebuild", path).Info("writing")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}

	for name, src := range extraFiles {
		destName, err := gentooExtraFilePath(name)
		if err != nil {
			return err
		}
		dst := filepath.Join(filepath.Dir(path), destName)
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		if err := copyFile(src, dst); err != nil {
			return err
		}
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: destName,
			Path: dst,
			Type: artifact.GentooFile,
			Extra: map[string]any{
				ebuildExtra:     cfg,
				ebuildPathExtra: pathlib.Join(filepath.ToSlash(filepath.Dir(cfg.Path)), filepath.ToSlash(destName)),
			},
		})
	}

	ctx.Artifacts.Add(&artifact.Artifact{
		Name: filepath.Base(path),
		Path: path,
		Type: artifact.GentooEbuild,
		Extra: map[string]any{
			ebuildExtra:     cfg,
			ebuildPathExtra: cfg.Path,
		},
	})

	if cfg.MetaCache {
		pkgVer := strings.TrimSuffix(filepath.Base(cfg.Path), ".ebuild")
		category := strings.Split(filepath.ToSlash(filepath.Clean(cfg.Path)), "/")[0]
		metaCachePath := pathlib.Join("metadata", "md5-cache", category, pkgVer)
		metaCacheDistPath := filepath.Join(ctx.Config.Dist, "gentoo", cfg.ID, metaCachePath)

		metaContent := generateMetaCacheContent(data, content)
		if err := os.MkdirAll(filepath.Dir(metaCacheDistPath), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(metaCacheDistPath, []byte(metaContent), 0o644); err != nil {
			return err
		}
		ctx.Artifacts.Add(&artifact.Artifact{
			Name: pkgVer,
			Path: metaCacheDistPath,
			Type: artifact.GentooFile,
			Extra: map[string]any{
				ebuildExtra:     cfg,
				ebuildPathExtra: metaCachePath,
				ebuildMetaCache: true,
			},
		})
	}

	return nil
}

func (Pipe) Publish(ctx *context.Context) error {
	arts := ctx.Artifacts.Filter(artifact.Or(
		artifact.ByType(artifact.GentooEbuild),
		artifact.ByType(artifact.GentooFile),
	)).List()

	type group struct {
		cfg   config.Gentoo
		files []client.RepoFile
	}

	groups := map[string]*group{}

	for _, art := range arts {
		cfg := artifact.MustExtra[config.Gentoo](*art, ebuildExtra)
		skip, err := tmpl.New(ctx).Apply(cfg.SkipUpload)
		if err != nil {
			return err
		}
		if strings.TrimSpace(skip) == "true" {
			log.Debug("gentoo.skip_upload is true")
			continue
		}
		if strings.TrimSpace(skip) == "auto" && ctx.Semver.Prerelease != "" {
			log.Debug("gentoo.skip_upload is auto and version is a prerelease")
			continue
		}
		key := cfg.ID
		g := groups[key]
		if g == nil {
			g = &group{cfg: cfg}
			groups[key] = g
		}
		content, err := os.ReadFile(art.Path)
		if err != nil {
			return err
		}
		g.files = append(g.files, client.RepoFile{
			Content: content,
			Path:    filepath.ToSlash(artifact.MustExtra[string](*art, ebuildPathExtra)),
		})
	}

	if len(groups) == 0 {
		return nil
	}

	cl, err := client.New(ctx)
	if err != nil {
		return err
	}

	for _, g := range groups {
		msg, err := tmpl.New(ctx).Apply(g.cfg.CommitMessageTemplate)
		if err != nil {
			return err
		}
		author, err := commitauthor.Get(ctx, g.cfg.CommitAuthor)
		if err != nil {
			return err
		}
		repo := client.RepoFromRef(g.cfg.Repository)

		repoClient, err := client.NewIfToken(ctx, cl, g.cfg.Repository.Token)
		if err != nil {
			return err
		}

		if g.cfg.Repository.PullRequest.Enabled {
			base := client.Repo{
				Name:   g.cfg.Repository.PullRequest.Base.Name,
				Owner:  g.cfg.Repository.PullRequest.Base.Owner,
				Branch: g.cfg.Repository.PullRequest.Base.Branch,
			}
			fscli, ok := repoClient.(client.ForkSyncer)
			if ok {
				if err := fscli.SyncFork(ctx, repo, base); err != nil {
					log.WithError(err).Warn("could not sync fork")
				}
			}
		}

		var deletedEbuilds []string
		// list existing ebuilds
		if lister, ok := repoClient.(client.DirectoryLister); ok && g.cfg.KeepVersions > 0 && g.cfg.VersionRetentionStrategy != "" {
			dir := filepath.ToSlash(filepath.Dir(g.cfg.Path))
			listRepo := repo
			if g.cfg.Repository.PullRequest.Enabled {
				listRepo.Branch = g.cfg.Repository.PullRequest.Base.Branch
			}
			names, err := lister.ListDir(ctx, listRepo, dir)
			if err != nil {
				return err
			}
			var ebuilds []string
			prefix := filepath.Base(dir) + "-"
			for _, n := range names {
				if strings.HasPrefix(n, prefix) && strings.HasSuffix(n, ".ebuild") {
					ebuilds = append(ebuilds, n)
				}
			}
			sort.Slice(ebuilds, func(i, j int) bool {
				vI := parseGentooVersion(ebuilds[i], prefix)
				vJ := parseGentooVersion(ebuilds[j], prefix)
				if vI != nil && vJ != nil {
					return vI.GreaterThan(vJ)
				}
				if vI != nil {
					return true
				}
				if vJ != nil {
					return false
				}
				return ebuilds[i] > ebuilds[j]
			})

			var newFiles []string
			for _, f := range g.files {
				name := filepath.Base(f.Path)
				if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".ebuild") {
					newFiles = append(newFiles, name)
				}
			}

			deleter := &ebuildDeleter{
				dir:            dir,
				category:       strings.Split(filepath.ToSlash(filepath.Clean(g.cfg.Path)), "/")[0],
				metaCache:      g.cfg.MetaCache,
				files:          &g.files,
				deletedEbuilds: &deletedEbuilds,
			}

			if g.cfg.VersionRetentionStrategy == "keep_prereleases" {
				var allEbuilds []string
				allEbuilds = append(allEbuilds, ebuilds...)
				allEbuilds = append(allEbuilds, newFiles...)

				maxVersions := map[string]*parsedGentooVersion{}
				for _, n := range allEbuilds {
					v := parseGentooVersion(n, prefix)
					if v == nil {
						continue
					}
					b := getVersionBucket(v)
					if maxVersions[b] == nil || v.GreaterThan(maxVersions[b]) {
						maxVersions[b] = v
					}
				}

				groups := map[string][]string{}
				for _, n := range ebuilds {
					v := parseGentooVersion(n, prefix)
					if v == nil {
						groups["stable"] = append(groups["stable"], n)
						continue
					}
					b := getVersionBucket(v)

					violates := false
					switch b {
					case "alpha":
						if (maxVersions["beta"] != nil && !v.GreaterThan(maxVersions["beta"])) ||
							(maxVersions["rc"] != nil && !v.GreaterThan(maxVersions["rc"])) ||
							(maxVersions["stable"] != nil && !v.GreaterThan(maxVersions["stable"])) {
							violates = true
						}
					case "beta":
						if (maxVersions["rc"] != nil && !v.GreaterThan(maxVersions["rc"])) ||
							(maxVersions["stable"] != nil && !v.GreaterThan(maxVersions["stable"])) {
							violates = true
						}
					case "rc":
						if maxVersions["stable"] != nil && !v.GreaterThan(maxVersions["stable"]) {
							violates = true
						}
					}

					if violates {
						deleter.Delete(n)
					} else {
						groups[b] = append(groups[b], n)
					}
				}

				newCounts := countNewEbuilds(ebuilds, newFiles, func(f string) string {
					v := parseGentooVersion(f, prefix)
					if v == nil {
						return "stable"
					}
					return getVersionBucket(v)
				})

				for b, bucketEbuilds := range groups {
					allowedToKeep := max(0, g.cfg.KeepVersions-newCounts[b])
					if len(bucketEbuilds) > allowedToKeep {
						for _, n := range bucketEbuilds[allowedToKeep:] {
							deleter.Delete(n)
						}
					}
				}
			} else if g.cfg.VersionRetentionStrategy == "keep_latest" {
				newUniqueCount := 0
				for _, n := range newFiles {
					if !slices.Contains(ebuilds, n) {
						newUniqueCount++
					}
				}
				allowedToKeep := max(0, g.cfg.KeepVersions-newUniqueCount)
				if len(ebuilds) > allowedToKeep {
					log.WithField("keep_versions", g.cfg.KeepVersions).
						WithField("new_unique", newUniqueCount).
						WithField("allowed_to_keep", allowedToKeep).
						WithField("total_old", len(ebuilds)).
						Debug("keeping latest versions")
					for _, n := range ebuilds[allowedToKeep:] {
						deleter.Delete(n)
					}
				}
			}
		}

		stateRepo := repo
		if g.cfg.Repository.PullRequest.Enabled {
			stateRepo.Branch = g.cfg.Repository.PullRequest.Base.Branch
		}

		settings, err := loadOverlaySettings(ctx, g.cfg, repoClient, stateRepo)
		if err != nil {
			return err
		}

		metaCacheAllowed := true
		if settings.hasCacheFormatsConfigured {
			metaCacheAllowed = slices.Contains(settings.cacheFormats, "md5-dict") || slices.Contains(settings.cacheFormats, "md5-cache")
		}

		if g.cfg.MetaCache && !metaCacheAllowed {
			log.Warnf("gentoo.meta_cache is true for %q, but overlay metadata/layout.conf disables cache-formats", g.cfg.ID)
		}

		var filteredFiles []client.RepoFile
		for _, f := range g.files {
			if strings.HasPrefix(filepath.ToSlash(f.Path), "metadata/md5-cache/") && !f.Delete && (!g.cfg.MetaCache || !metaCacheAllowed) {
				continue
			}
			filteredFiles = append(filteredFiles, f)
		}
		g.files = filteredFiles
		if err := handleGentooManifestAndMetadata(ctx, g.cfg, repoClient, stateRepo, &g.files, deletedEbuilds); err != nil {
			return err
		}

		if g.cfg.Repository.Git.URL != "" {
			if err := client.NewGitUploadClient(repo.Branch).CreateFiles(ctx, author, repo, msg, g.files); err != nil {
				return err
			}
		} else if fc, ok := repoClient.(client.FilesCreator); ok {
			err = fc.CreateFiles(ctx, author, repo, msg, g.files)
			if err != nil {
				return err
			}
		} else {
			var filesToCreate []client.RepoFile
			for _, f := range g.files {
				if f.Delete {
					if d, ok := repoClient.(client.FileDeleter); ok {
						if err := d.DeleteFile(ctx, author, repo, f.Path, msg); err != nil {
							return err
						}
					}
					continue
				}
				filesToCreate = append(filesToCreate, f)
			}
			if len(filesToCreate) > 0 {
				if fc, ok := repoClient.(client.FilesCreator); ok {
					if err := fc.CreateFiles(ctx, author, repo, msg, filesToCreate); err != nil {
						return err
					}
				} else {
					for _, f := range filesToCreate {
						if err := repoClient.CreateFile(ctx, author, repo, f.Content, f.Path, msg); err != nil {
							return err
						}
					}
				}
			}
		}

		if !g.cfg.Repository.PullRequest.Enabled {
			continue
		}

		base := client.Repo{
			Name:   g.cfg.Repository.PullRequest.Base.Name,
			Owner:  g.cfg.Repository.PullRequest.Base.Owner,
			Branch: g.cfg.Repository.PullRequest.Base.Branch,
		}
		prClient, err := client.NewIfToken(ctx, repoClient, g.cfg.Repository.PullRequest.Token)
		if err != nil {
			return err
		}
		pcl, ok := prClient.(client.PullRequestOpener)
		if !ok {
			return errors.New("client does not support pull requests")
		}
		if err := pcl.OpenPullRequest(ctx, base, repo, msg, g.cfg.Repository.PullRequest.Draft); err != nil {
			return err
		}
	}
	return nil
}

func gentooArch(goarch string) string {
	switch goarch {
	case "386":
		return "x86"
	case "riscv64":
		return "riscv"
	case "ppc64le":
		return "ppc64"
	case "s390x":
		return "s390"
	default:
		return goarch
	}
}

func defaultPath(name, category, typ string) string {
	if category == "" {
		category = "app-misc"
	}
	suffix := ""
	if typ == "bin" {
		suffix = "-bin"
	}
	return filepath.Join(category, name+suffix, fmt.Sprintf("%s%s-{{ .Version }}.ebuild", name, suffix))
}

func hasCategory(path string) bool {
	parts := strings.Split(filepath.ToSlash(filepath.Clean(path)), "/")
	return len(parts) >= 3 && parts[0] != "." && parts[0] != "" && parts[1] != "" && parts[2] != ""
}

func isValidGentooPath(filePath string) bool {
	path := filepath.ToSlash(filePath)
	if pathlib.IsAbs(path) || path != pathlib.Clean(path) {
		return false
	}
	parts := strings.Split(path, "/")
	return len(parts) == 3 &&
		parts[0] != "" && parts[0] != "." && parts[0] != ".." &&
		parts[1] != "" && parts[1] != "." && parts[1] != ".." &&
		strings.HasSuffix(parts[2], ".ebuild")
}

func gentooExtraFilePath(name string) (string, error) {
	path := filepath.ToSlash(name)
	path = strings.TrimPrefix(path, "files/")
	if path == "" || pathlib.IsAbs(path) || path != pathlib.Clean(path) || strings.HasPrefix(path, "../") || path == ".." {
		return "", fmt.Errorf("gentoo extra file name %q must remain within the files directory", name)
	}
	return pathlib.Join("files", path), nil
}

func gentooUseFlags(cfg config.Gentoo) []config.GentooUseFlag {
	flags := append([]config.GentooUseFlag(nil), cfg.UseFlags...)
	configured := make(map[string]struct{}, len(flags))
	for _, flag := range flags {
		configured[strings.TrimLeft(flag.Flag, "+-")] = struct{}{}
	}

	items := [][]config.GentooInstallItem{
		cfg.Dobin, cfg.Doconfd, cfg.Doenvd, cfg.Doexe, cfg.Doheader, cfg.Doinitd,
		cfg.Doins, cfg.Dosbin, cfg.Dosym, cfg.Systemd,
	}
	var additional []string
	for _, group := range items {
		for _, item := range group {
			for _, condition := range item.Use {
				flag := strings.TrimLeft(condition, "!+-")
				if flag != "" {
					if _, ok := configured[flag]; !ok {
						configured[flag] = struct{}{}
						additional = append(additional, flag)
					}
				}
			}
		}
	}
	slices.Sort(additional)
	for _, flag := range additional {
		flags = append(flags, config.GentooUseFlag{Flag: flag})
	}
	return flags
}

func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, 0o644)
}

func generateManifestLine(recordType, filename, path string, content []byte, manifestHashes []string) (string, error) {
	var r io.Reader
	var size int64

	if content == nil && path != "" {
		info, err := os.Stat(path)
		if err != nil {
			return "", err
		}
		size = info.Size()

		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()
		r = f
	} else {
		size = int64(len(content))
		r = bytes.NewReader(content)
	}

	var writers []io.Writer
	var b2b hash.Hash
	var s512 hash.Hash
	var s256 hash.Hash

	for _, algo := range manifestHashes {
		algo = strings.ToUpper(algo)
		switch algo {
		case "BLAKE2B":
			b2b, _ = blake2b.New512(nil)
			writers = append(writers, b2b)
		case "SHA512":
			s512 = sha512.New()
			writers = append(writers, s512)
		case "SHA256":
			s256 = sha256.New()
			writers = append(writers, s256)
		default:
			return "", fmt.Errorf("unsupported manifest hash algorithm: %s", algo)
		}
	}

	if len(writers) > 0 {
		if _, err := io.Copy(io.MultiWriter(writers...), r); err != nil {
			return "", err
		}
	}

	line := fmt.Sprintf("%s %s %d", recordType, filename, size)
	for _, algo := range manifestHashes {
		algo = strings.ToUpper(algo)
		switch algo {
		case "BLAKE2B":
			if b2b != nil {
				line = fmt.Sprintf("%s BLAKE2B %x", line, b2b.Sum(nil))
			}
		case "SHA512":
			if s512 != nil {
				line = fmt.Sprintf("%s SHA512 %x", line, s512.Sum(nil))
			}
		case "SHA256":
			if s256 != nil {
				line = fmt.Sprintf("%s SHA256 %x", line, s256.Sum(nil))
			}
		}
	}

	return line, nil
}

func handleGentooManifestAndMetadata(ctx *context.Context, cfg config.Gentoo, repoClient client.Client, repo client.Repo, files *[]client.RepoFile, deletedEbuilds []string) error {
	dir := filepath.ToSlash(filepath.Dir(cfg.Path))

	metadataPath := pathlib.Join(dir, "metadata.xml")
	manifestPath := pathlib.Join(dir, "Manifest")

	if len(cfg.Maintainers) > 0 || cfg.BugsTo != "" || cfg.Homepage != "" || len(cfg.UseFlags) > 0 {
		type innerNode struct {
			XMLName xml.Name
			Content string     `xml:",innerxml"`
			Attrs   []xml.Attr `xml:",any,attr"`
		}
		type gentooMaintainer struct {
			XMLName xml.Name    `xml:"maintainer"`
			Type    string      `xml:"type,attr,omitempty"`
			Email   string      `xml:"email,omitempty"`
			Name    string      `xml:"name,omitempty"`
			Attrs   []xml.Attr  `xml:",any,attr"`
			Nodes   []innerNode `xml:",any"`
		}
		type gentooUpstream struct {
			XMLName xml.Name    `xml:"upstream"`
			BugsTo  string      `xml:"bugs-to,omitempty"`
			Doc     string      `xml:"doc,omitempty"`
			Attrs   []xml.Attr  `xml:",any,attr"`
			Nodes   []innerNode `xml:",any"`
		}
		type gentooUseFlag struct {
			XMLName xml.Name   `xml:"flag"`
			Name    string     `xml:"name,attr"`
			Value   string     `xml:",chardata"`
			Attrs   []xml.Attr `xml:",any,attr"`
		}
		type gentooUse struct {
			XMLName xml.Name        `xml:"use"`
			Flags   []gentooUseFlag `xml:"flag"`
			Attrs   []xml.Attr      `xml:",any,attr"`
			Nodes   []innerNode     `xml:",any"`
		}
		type gentooMetadata struct {
			XMLName     xml.Name           `xml:"pkgmetadata"`
			Attrs       []xml.Attr         `xml:",any,attr"`
			Maintainers []gentooMaintainer `xml:"maintainer"`
			Use         *gentooUse         `xml:"use,omitempty"`
			Upstream    *gentooUpstream    `xml:"upstream,omitempty"`
			InnerNodes  []innerNode        `xml:",any"`
		}

		meta := gentooMetadata{}
		if dl, ok := repoClient.(client.FileDownloader); ok {
			content, err := dl.DownloadFile(ctx, repo, metadataPath)
			if err == nil {
				_ = xml.Unmarshal(content, &meta)
			} else if !errors.Is(err, client.ErrNotFound) && !errors.Is(err, client.ErrNotImplemented) {
				return fmt.Errorf("failed to download metadata.xml: %w", err)
			}
		}
		if meta.Use == nil {
			meta.Use = &gentooUse{}
		}

		configuredFlags := make(map[string]string)
		configuredFlags["doc"] = "Install README man page and other docs"
		for _, flag := range cfg.UseFlags {
			if flag.Description != "" {
				configuredFlags[strings.TrimLeft(flag.Flag, "+-")] = flag.Description
			}
		}

		var configuredFlagNames []string
		for k := range configuredFlags {
			configuredFlagNames = append(configuredFlagNames, k)
		}
		sort.Strings(configuredFlagNames)

		for _, k := range configuredFlagNames {
			v := configuredFlags[k]
			exists := false
			for _, ef := range meta.Use.Flags {
				if ef.Name == k {
					exists = true
					break
				}
			}
			if !exists {
				meta.Use.Flags = append(meta.Use.Flags, gentooUseFlag{
					Name:  k,
					Value: v,
				})
			}
		}

		for _, m := range cfg.Maintainers {
			if m.Email == "" {
				return errors.New("gentoo maintainer email is required")
			}
			exists := false
			for _, em := range meta.Maintainers {
				if em.Email == m.Email {
					exists = true
					break
				}
			}
			if !exists {
				meta.Maintainers = append(meta.Maintainers, gentooMaintainer{
					Type:  "person",
					Email: m.Email,
					Name:  m.Name,
				})
			}
		}
		if cfg.BugsTo != "" || cfg.Homepage != "" {
			if meta.Upstream == nil {
				meta.Upstream = &gentooUpstream{}
			}
			if cfg.BugsTo != "" {
				meta.Upstream.BugsTo = cfg.BugsTo
			}
			if cfg.Homepage != "" {
				meta.Upstream.Doc = cfg.Homepage
			}
		}

		useFlags := gentooUseFlags(cfg)
		tmplData := struct {
			config.Gentoo
			UseFlags []config.GentooUseFlag
		}{
			Gentoo:   cfg,
			UseFlags: make([]config.GentooUseFlag, len(useFlags)),
		}
		for i, f := range useFlags {
			tmplData.UseFlags[i] = config.GentooUseFlag{
				Flag:        strings.TrimLeft(f.Flag, "+-"),
				Description: f.Description,
			}
		}

		var buf bytes.Buffer
		if err := template.Must(template.New("metadata.xml").Parse(metadataXMLTemplate)).Execute(&buf, tmplData); err != nil {
			return err
		}
		*files = append(*files, client.RepoFile{
			Content: buf.Bytes(),
			Path:    metadataPath,
		})
	}

	settings, err := loadOverlaySettings(ctx, cfg, repoClient, repo)
	if err != nil {
		return err
	}
	manifestHashes, thinManifests := settings.hashes, settings.thin
	manifestLines, err := loadManifestLines(ctx, repoClient, repo, manifestPath)
	if err != nil {
		return err
	}

	var deletedVersions []string
	prefix := filepath.Base(dir) + "-"
	for _, e := range deletedEbuilds {
		v := strings.TrimSuffix(strings.TrimPrefix(e, prefix), ".ebuild")
		deletedVersions = append(deletedVersions, v)
	}

	newManifestFiles := map[string]struct{}{}
	if !thinManifests {
		for _, f := range *files {
			if !f.Delete {
				recordType, filename := manifestFileInfo(f.Path, dir)
				newManifestFiles[recordType+":"+filename] = struct{}{}
			}
		}
	}
	filters := []artifact.Filter{
		artifact.ByGoos("linux"),
		artifact.ByType(artifact.UploadableArchive),
		artifact.OnlyReplacingUnibins,
	}
	if len(cfg.IDs) > 0 {
		filters = append(filters, artifact.ByIDs(cfg.IDs...))
	}
	arches := ctx.Artifacts.Filter(artifact.And(filters...)).List()
	currentDists := make(map[string]struct{}, len(arches))
	for _, art := range arches {
		currentDists[art.Name] = struct{}{}
	}

	var newManifestLines []string
	for _, line := range manifestLines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			newManifestLines = append(newManifestLines, line)
			continue
		}

		recordType := fields[0]
		filename := fields[1]

		switch recordType {
		case "DIST":
			_, removed := currentDists[filename]
			for _, dv := range deletedVersions {
				if idx := strings.Index(filename, dv); idx != -1 {
					isMatch := true
					if idx > 0 && filename[idx-1] != '_' && filename[idx-1] != '-' {
						isMatch = false
					}
					endIdx := idx + len(dv)
					if endIdx < len(filename) {
						next := filename[endIdx]
						if next == '.' {
							if endIdx+1 < len(filename) && filename[endIdx+1] >= '0' && filename[endIdx+1] <= '9' {
								isMatch = false
							}
						} else if next != '_' && next != '-' {
							isMatch = false
						}
					}
					if isMatch {
						removed = true
						break
					}
				}
			}
			if !removed {
				newManifestLines = append(newManifestLines, line)
			}
		case "EBUILD", "AUX", "MISC":
			if thinManifests {
				continue
			}
			removed := false
			for _, dv := range deletedVersions {
				if recordType == "EBUILD" && filename == filepath.Base(dir)+"-"+dv+".ebuild" {
					removed = true
					break
				}
			}
			if !removed {
				_, removed = newManifestFiles[recordType+":"+filename]
			}
			if !removed {
				newManifestLines = append(newManifestLines, line)
			}
		default:
			newManifestLines = append(newManifestLines, line)
		}
	}

	for _, art := range arches {
		line, err := generateManifestLine("DIST", art.Name, art.Path, nil, manifestHashes)
		if err != nil {
			return err
		}
		newManifestLines = append(newManifestLines, line)
	}

	if !thinManifests {
		for _, f := range *files {
			if f.Delete {
				continue
			}

			recordType, filename := manifestFileInfo(f.Path, dir)

			line, err := generateManifestLine(recordType, filename, f.Path, f.Content, manifestHashes)
			if err != nil {
				return err
			}
			newManifestLines = append(newManifestLines, line)
		}
	}

	if len(newManifestLines) > 0 {
		slices.Sort(newManifestLines)
		newManifestLines = slices.Compact(newManifestLines)
		*files = append(*files, client.RepoFile{
			Content: []byte(strings.Join(newManifestLines, "\n") + "\n"),
			Path:    manifestPath,
		})
	}
	return nil
}

type overlaySettings struct {
	hashes                    []string
	thin                      bool
	cacheFormats              []string
	hasCacheFormatsConfigured bool
}

func loadOverlaySettings(ctx *context.Context, cfg config.Gentoo, repoClient client.Client, repo client.Repo) (overlaySettings, error) {
	settings := overlaySettings{
		hashes: []string{"BLAKE2B", "SHA512"},
		thin:   false,
	}
	if len(cfg.ManifestHashes) > 0 {
		settings.hashes = cfg.ManifestHashes
	}
	if cfg.ThinManifests != nil {
		settings.thin = *cfg.ThinManifests
	}

	dl, ok := repoClient.(client.FileDownloader)
	if !ok {
		return settings, nil
	}
	content, err := dl.DownloadFile(ctx, repo, "metadata/layout.conf")
	if errors.Is(err, client.ErrNotFound) || errors.Is(err, client.ErrNotImplemented) {
		return settings, nil
	}
	if err != nil {
		return settings, fmt.Errorf("failed to download layout.conf: %w", err)
	}
	for lineB := range bytes.SplitSeq(content, []byte{'\n'}) {
		key, value, ok := strings.Cut(strings.TrimSpace(string(lineB)), "=")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "manifest-hashes":
			if len(cfg.ManifestHashes) == 0 {
				settings.hashes = strings.Fields(value)
			}
		case "thin-manifests":
			if cfg.ThinManifests == nil {
				settings.thin = strings.TrimSpace(value) == "true"
			}
		case "cache-formats":
			settings.hasCacheFormatsConfigured = true
			settings.cacheFormats = strings.Fields(value)
		}
	}
	return settings, nil
}

func loadManifestLines(ctx *context.Context, repoClient client.Client, repo client.Repo, manifestPath string) ([]string, error) {
	dl, ok := repoClient.(client.FileDownloader)
	if !ok {
		return nil, nil
	}
	content, err := dl.DownloadFile(ctx, repo, manifestPath)
	if errors.Is(err, client.ErrNotFound) || errors.Is(err, client.ErrNotImplemented) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to download Manifest: %w", err)
	}
	return strings.FieldsFunc(string(content), func(r rune) bool { return r == '\n' || r == '\r' }), nil
}

func manifestFileInfo(filePath, packageDir string) (string, string) {
	path := filepath.ToSlash(filePath)
	filesDir := pathlib.Join(packageDir, "files")
	if path == filesDir || strings.HasPrefix(path, filesDir+"/") {
		return "AUX", strings.TrimPrefix(path, filesDir+"/")
	}
	if strings.HasSuffix(path, ".ebuild") {
		return "EBUILD", pathlib.Base(path)
	}
	return "MISC", pathlib.Base(path)
}

func countNewEbuilds(existing, newFiles []string, bucket func(string) string) map[string]int {
	counts := map[string]int{}
	for _, file := range newFiles {
		if !slices.Contains(existing, file) {
			counts[bucket(file)]++
		}
	}
	return counts
}

func generateMetaCacheContent(data any, ebuildContent string) string {
	d, ok := data.(struct {
		Name          string
		Description   string
		Homepage      string
		License       string
		Keywords      string
		Bindir        string
		ExtraInstall  string
		Archs         []archData
		InstallGroups []installGroup
		UseFlags      []config.GentooUseFlag
		Dobin         []installItemData
		Doconfd       []installItemData
		Dodir         []string
		Dodoc         []string
		Doenvd        []installItemData
		Doexe         []installItemData
		Doheader      []installItemData
		Doinitd       []installItemData
		Doins         []installItemData
		Doman         []string
		Dosbin        []installItemData
		Dosym         []installItemData
		Systemd       []installItemData
	})
	if !ok {
		return ""
	}
	var useFlags []string
	for _, flag := range d.UseFlags {
		if flag.Flag != "" {
			useFlags = append(useFlags, flag.Flag)
		}
	}
	slices.Sort(useFlags)
	useFlags = slices.Compact(useFlags)

	var srcURIs []string
	for _, art := range d.Archs {
		if art.Keyword != "" && art.URI != "" {
			srcURIs = append(srcURIs, fmt.Sprintf("%s? ( %s )", art.Keyword, art.URI))
		}
	}

	h := md5.Sum([]byte(ebuildContent))
	md5Hex := hex.EncodeToString(h[:])

	tmplData := struct {
		Description string
		Homepage    string
		IUSE        string
		Keywords    string
		License     string
		SrcURI      string
		MD5         string
	}{
		Description: d.Description,
		Homepage:    d.Homepage,
		IUSE:        strings.Join(useFlags, " "),
		Keywords:    d.Keywords,
		License:     d.License,
		SrcURI:      strings.Join(srcURIs, " "),
		MD5:         md5Hex,
	}

	var buf bytes.Buffer
	if err := template.Must(template.New("md5-cache").Parse(metaCacheTemplate)).Execute(&buf, tmplData); err != nil {
		return ""
	}
	return buf.String()
}
