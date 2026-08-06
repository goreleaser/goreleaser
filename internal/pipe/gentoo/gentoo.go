// Package gentoo implements the gentoo ebuild pipe.
package gentoo

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

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
	cl, err := client.New(ctx)
	if err != nil {
		return err
	}
	return runAll(ctx, cl)
}

func runAll(ctx *context.Context, cl client.ReleaseURLTemplater) error {
	for _, cfg := range ctx.Config.Gentoos {
		if err := doRun(ctx, cfg, cl); err != nil {
			return err
		}
	}
	return nil
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

	if cfg.Path == "" || !hasCategory(cfg.Path) {
		return errors.New("gentoo.path is required and must include the category/package ebuild path")
	}
	if strings.HasPrefix(filepath.ToSlash(filepath.Clean(cfg.Path)), "../") || strings.Contains(filepath.ToSlash(filepath.Clean(cfg.Path)), "/../") {
		return fmt.Errorf("gentoo.path %q must be a relative category/package/file.ebuild path", cfg.Path)
	}

	path, err := tp.Apply(cfg.Path)
	if err != nil {
		return err
	}

	path = filepath.Join(ctx.Config.Dist, "gentoo", cfg.ID, path)

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

	data := ebuildData{
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

	if err := data.Validate(); err != nil {
		return err
	}

	content, err := data.RenderEbuild()
	if err != nil {
		return err
	}

	log.WithField("ebuild", path).Info("writing")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}

	if err := ef.InstallExtraFiles(ctx, path); err != nil {
		return err
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
		pkgVer := strings.TrimSuffix(filepath.Base(path), ".ebuild")
		category := strings.Split(filepath.ToSlash(filepath.Clean(cfg.Path)), "/")[0]
		metaCachePath := filepath.ToSlash(filepath.Join("metadata", "md5-cache", category, pkgVer))
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

type publishGroup struct {
	cfg   config.Gentoo
	files []client.RepoFile
}

func collectPublishGroups(ctx *context.Context) ([]*publishGroup, error) {
	arts := ctx.Artifacts.Filter(artifact.Or(
		artifact.ByType(artifact.GentooEbuild),
		artifact.ByType(artifact.GentooFile),
	)).List()

	groupMap := map[string]*publishGroup{}
	var groups []*publishGroup

	for _, art := range arts {
		cfg := artifact.MustExtra[config.Gentoo](*art, ebuildExtra)
		skip, err := tmpl.New(ctx).Apply(cfg.SkipUpload)
		if err != nil {
			return nil, err
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
		g := groupMap[key]
		if g == nil {
			g = &publishGroup{cfg: cfg}
			groupMap[key] = g
			groups = append(groups, g)
		}
		content, err := os.ReadFile(art.Path)
		if err != nil {
			return nil, err
		}
		g.files = append(g.files, client.RepoFile{
			Content: content,
			Path:    filepath.ToSlash(artifact.MustExtra[string](*art, ebuildPathExtra)),
		})
	}
	return groups, nil
}

func (g *publishGroup) applyVersionRetention(ctx *context.Context, repoClient client.Client, repo client.Repo) ([]string, error) {
	lister, ok := repoClient.(client.DirectoryLister)
	if !ok || g.cfg.KeepVersions <= 0 || g.cfg.VersionRetentionStrategy == "" {
		return nil, nil
	}

	dir := filepath.ToSlash(filepath.Dir(g.cfg.Path))
	listRepo := repo
	if g.cfg.Repository.PullRequest.Enabled {
		listRepo.Branch = g.cfg.Repository.PullRequest.Base.Branch
	}
	names, err := lister.ListDir(ctx, listRepo, dir)
	if err != nil {
		return nil, err
	}
	var ebuilds []string
	prefix := filepath.Base(dir) + "-"
	for _, n := range names {
		if strings.HasPrefix(n, prefix) && strings.HasSuffix(n, ".ebuild") {
			ebuilds = append(ebuilds, n)
		}
	}
	slices.SortFunc(ebuilds, func(i, j string) int {
		vI := parseGentooVersion(i, prefix)
		vJ := parseGentooVersion(j, prefix)
		if vI != nil && vJ != nil {
			if vI.GreaterThan(vJ) {
				return -1
			}
			if vJ.GreaterThan(vI) {
				return 1
			}
			return 0
		}
		if vI != nil {
			return -1
		}
		if vJ != nil {
			return 1
		}
		return strings.Compare(j, i)
	})

	var newFiles []string
	for _, f := range g.files {
		name := filepath.Base(f.Path)
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".ebuild") {
			newFiles = append(newFiles, name)
		}
	}

	category := strings.Split(filepath.ToSlash(filepath.Clean(g.cfg.Path)), "/")[0]
	metaCacheFiles := map[string]struct{}{}
	if lister, ok := repoClient.(client.DirectoryLister); ok && g.cfg.MetaCache {
		cacheNames, err := lister.ListDir(ctx, listRepo, filepath.ToSlash(filepath.Join("metadata", "md5-cache", category)))
		if err != nil && !errors.Is(err, client.ErrNotFound) && !errors.Is(err, client.ErrNotImplemented) {
			return nil, err
		}
		for _, name := range cacheNames {
			metaCacheFiles[name] = struct{}{}
		}
	}

	var deletedEbuilds []string
	deleter := &ebuildDeleter{
		dir:            dir,
		category:       category,
		metaCacheFiles: metaCacheFiles,
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
	return deletedEbuilds, nil
}

func (g *publishGroup) publish(ctx *context.Context, cl client.Client) error {
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

	deletedEbuilds, err := g.applyVersionRetention(ctx, repoClient, repo)
	if err != nil {
		return err
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
		return nil
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
	return pcl.OpenPullRequest(ctx, base, repo, msg, g.cfg.Repository.PullRequest.Draft)
}

func (Pipe) Publish(ctx *context.Context) error {
	groups, err := collectPublishGroups(ctx)
	if err != nil || len(groups) == 0 {
		return err
	}

	cl, err := client.New(ctx)
	if err != nil {
		return err
	}

	for _, g := range groups {
		if err := g.publish(ctx, cl); err != nil {
			return err
		}
	}
	return nil
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = out.ReadFrom(in)
	return err
}
