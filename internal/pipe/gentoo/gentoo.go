// Package gentoo implements the gentoo ebuild pipe.
package gentoo

import (
	"bytes"
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
	ids := ids.New("gentoo_overlays")
	for i := range ctx.Config.Gentoos {
		g := &ctx.Config.Gentoos[i]
		g.CommitAuthor = commitauthor.Default(g.CommitAuthor)
		if g.ID == "" {
			g.ID = "default"
		}
		if !g.Bin {
			return errors.New("bin must be true")
		}
		if g.CommitMessageTemplate == "" {
			g.CommitMessageTemplate = "{{ .ProjectName }}: bump to {{ .Tag }}"
		}
		if g.Type == "" {
			g.Type = "bin"
		}
		if g.Type != "bin" {
			return fmt.Errorf("invalid gentoo type %q: currently only \"bin\" is supported", g.Type)
		}
		if g.Type == "bin" && g.Bindir == "" {
			g.Bindir = "/opt/bin"
		} else if g.Bindir == "" {
			g.Bindir = "/usr/bin"
		}
		if g.License == "" {
			return errors.New("license is required")
		}
		if strings.TrimSpace(g.Description) == "" {
			g.Description = ctx.Config.ProjectName
		}
		if strings.TrimSpace(g.Description) == "" {
			return errors.New("description is required")
		}
		if g.ConflictResolution == "" {
			g.ConflictResolution = config.ConflictResolutionRevision
		}
		if g.ConflictResolution != config.ConflictResolutionFail &&
			g.ConflictResolution != config.ConflictResolutionOverwrite &&
			g.ConflictResolution != config.ConflictResolutionRevision {
			return fmt.Errorf("conflict_resolution %q is not valid, must be one of [Fail, Overwrite, Revision]", g.ConflictResolution)
		}
		if g.KeepVersions < 0 {
			return errors.New("keep_versions must be greater than or equal to 0")
		}
		if g.VersionRetentionStrategy != "" && g.VersionRetentionStrategy != config.VersionRetentionStrategyKeepLatest && g.VersionRetentionStrategy != config.VersionRetentionStrategyKeepPrereleases {
			return fmt.Errorf("version_retention_strategy %q is not valid, must be one of [keep_latest, keep_prereleases]", g.VersionRetentionStrategy)
		}
		if g.KeepVersions > 0 && g.VersionRetentionStrategy == "" {
			return errors.New("version_retention_strategy must be provided if keep_versions > 0")
		}
		if g.Name == "" {
			g.Name = ctx.Config.ProjectName
		}
		if g.Category == "" {
			g.Category = "app-misc"
		}
		ids.Inc(g.ID)
	}
	return ids.Validate()
}

func (Pipe) Run(ctx *context.Context) error {
	cl, err := client.NewReleaseClient(ctx)
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
		"GentooVersion": gentooVersion(ctx.Version),
		"Version":       gentooVersion(ctx.Version),
		"Name":          cfg.Name,
		"Category":      cfg.Category,
	})
	if err := tp.ApplyAll(&cfg.Name, &cfg.Category, &cfg.OverlayPath, &cfg.Description, &cfg.Homepage, &cfg.BugsTo, &cfg.License); err != nil {
		return err
	}
	var err error
	cfg.Repository, err = client.TemplateRef(tp.Apply, cfg.Repository)
	if err != nil {
		return err
	}

	relPath := ebuildRelPath(ctx, cfg)
	if strings.HasPrefix(filepath.ToSlash(filepath.Clean(relPath)), "../") || strings.Contains(filepath.ToSlash(filepath.Clean(relPath)), "/../") {
		return fmt.Errorf("path %q must be a relative category/package/file.ebuild path", relPath)
	}

	path := filepath.Join(ctx.Config.Dist, "gentoo", cfg.ID, relPath)

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

	keywordSet := map[string]struct{}{}

	uriTemplate, err := cl.ReleaseURLTemplate(ctx)
	if err != nil {
		return err
	}

	archMap := make(map[string][]archItem)
	seenArchID := make(map[string]map[string]*artifact.Artifact)
	var keywordsOrder []string

	for _, art := range arches {
		url, err := tmpl.New(ctx).WithArtifact(art).Apply(uriTemplate)
		if err != nil {
			return err
		}
		kw, err := gentooArch(art.Goarch)
		if err != nil {
			return err
		}
		id := artifact.ExtraOr(*art, artifact.ExtraID, "default")
		if seenArchID[kw] == nil {
			seenArchID[kw] = make(map[string]*artifact.Artifact)
			keywordsOrder = append(keywordsOrder, kw)
		}
		if prev, exists := seenArchID[kw][id]; exists {
			return fmt.Errorf("multiple linux archives map to Gentoo architecture %q for ID %q (%s and %s); please filter artifacts", kw, id, prev.Name, art.Name)
		}
		seenArchID[kw][id] = art
		fileName := art.Name
		versionStr := gentooVersion(ctx.Version)
		if !strings.Contains(fileName, versionStr) && !strings.Contains(fileName, ctx.Version) {
			fileName = fmt.Sprintf("%s-%s-%s", cfg.Name, versionStr, fileName)
		}
		archMap[kw] = append(archMap[kw], archItem{
			File: fileName,
			URI:  url,
		})
		keywordSet["~"+kw] = struct{}{}
	}

	var archInfos []archData
	slices.Sort(keywordsOrder)
	keywordsOrder = slices.Compact(keywordsOrder)
	for _, kw := range keywordsOrder {
		uris := archMap[kw]
		slices.SortFunc(uris, func(a, b archItem) int {
			return strings.Compare(a.File, b.File)
		})
		archInfos = append(archInfos, archData{
			Keyword: kw,
			URIs:    uris,
		})
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

	suppressedIDs := collectSuppressedIDs(cfg)
	installByKw := make(map[string][]installData)

	for _, art := range arches {
		artID := artifact.ExtraOr(*art, artifact.ExtraID, "default")
		if suppressedIDs[artID] {
			continue
		}
		kw, _ := gentooArch(art.Goarch)
		bins := artifact.ExtraOr(*art, artifact.ExtraBinaries, []string{})
		wrappedIn := artifact.ExtraOr(*art, artifact.ExtraWrappedIn, "")
		if len(bins) == 0 {
			bins = []string{cfg.Name}
		}
		for _, b := range bins {
			sourcePath := b
			if wrappedIn != "" {
				sourcePath = filepath.ToSlash(filepath.Join(wrappedIn, b))
			}
			targetName := filepath.Base(b)
			installByKw[kw] = append(installByKw[kw], installData{
				Source:   sourcePath,
				Target:   targetName,
				Keywords: []string{kw},
			})
		}
	}

	for kw := range installByKw {
		slices.SortFunc(installByKw[kw], func(a, b installData) int {
			if c := strings.Compare(a.Source, b.Source); c != 0 {
				return c
			}
			return strings.Compare(a.Target, b.Target)
		})
	}

	var installGroups []installGroup
	if len(installByKw) > 0 {
		groupMap := make(map[string][]string)
		installItemsMap := make(map[string][]installData)

		for _, kw := range keywordsList {
			installs := installByKw[kw]
			if len(installs) == 0 {
				continue
			}
			var keyParts []string
			for _, inst := range installs {
				keyParts = append(keyParts, inst.Source+":"+inst.Target)
			}
			groupKey := strings.Join(keyParts, ";")
			groupMap[groupKey] = append(groupMap[groupKey], kw)
			installItemsMap[groupKey] = installs
		}

		groupKeys := make([]string, 0, len(groupMap))
		for groupKey := range groupMap {
			groupKeys = append(groupKeys, groupKey)
		}
		slices.Sort(groupKeys)

		for _, groupKey := range groupKeys {
			kws := groupMap[groupKey]
			slices.Sort(kws)
			installs := installItemsMap[groupKey]
			for i := range installs {
				installs[i].Keywords = kws
			}
			installGroups = append(installGroups, installGroup{
				Keywords: kws,
				Installs: installs,
			})
		}
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

	dobin, err := ef.buildInstallItems("dobin", cfg.Dobin, "")
	if err != nil {
		return err
	}
	doconfd, err := ef.buildInstallItems("doconfd", cfg.Doconfd, "")
	if err != nil {
		return err
	}
	doenvd, err := ef.buildInstallItems("doenvd", cfg.Doenvd, "")
	if err != nil {
		return err
	}
	doexe, err := ef.buildInstallItems("doexe", cfg.Doexe, cfg.Bindir)
	if err != nil {
		return err
	}
	doheader, err := ef.buildInstallItems("doheader", cfg.Doheader, "")
	if err != nil {
		return err
	}
	doinitd, err := ef.buildInstallItems("doinitd", cfg.Doinitd, "")
	if err != nil {
		return err
	}
	doins, err := ef.buildInstallItems("doins", cfg.Doins, "/")
	if err != nil {
		return err
	}
	dosbin, err := ef.buildInstallItems("dosbin", cfg.Dosbin, "")
	if err != nil {
		return err
	}
	dosym, err := ef.buildInstallItems("dosym", cfg.Dosym, "")
	if err != nil {
		return err
	}
	systemd, err := ef.buildInstallItems("systemd", cfg.Systemd, "")
	if err != nil {
		return err
	}

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
		Dodir:         cfg.Dodir,
		Dodoc:         ef.processStringArray(cfg.Dodoc),
		Installers:    append(append(append(append(append(append(append(append(append([]installItemData{}, dobin...), doconfd...), doenvd...), doexe...), doheader...), doinitd...), doins...), dosbin...), dosym...),
		Doman:         ef.processStringArray(cfg.Doman),
		Systemd:       systemd,
	}

	var eclasses []string
	for _, e := range cfg.Eclasses {
		if !slices.Contains(eclasses, e) {
			eclasses = append(eclasses, e)
		}
	}
	data.Eclasses = eclasses

	if !slices.Contains(eclasses, "systemd") && len(data.Systemd) > 0 {
		for _, item := range data.Systemd {
			item.Target = "1"
			item.Dir = "/usr/lib/systemd/system"
			item.DirSwitchCmd = "insinto"
			item.InstallerCmd = "doins"
			item.InstallRenameCmd = "newins"
			data.Installers = append(data.Installers, item)
		}
		data.Systemd = nil
	} else if len(data.Systemd) > 0 {
		data.Installers = append(data.Installers, data.Systemd...)
		data.Systemd = nil
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
			ebuildPathExtra: relPath,
		},
	})

	if cfg.MetaCache {
		pkgVer := strings.TrimSuffix(filepath.Base(path), ".ebuild")
		if data.HasEclasses() {
			log.Warnf("gentoo: meta_cache is enabled for %q, but ebuild %q inherits eclasses; skipping metadata cache generation", cfg.ID, pkgVer)
		} else {
			metaCachePath := filepath.ToSlash(filepath.Join("metadata", "md5-cache", cfg.Category, pkgVer))
			if cfg.OverlayPath != "" {
				metaCachePath = filepath.ToSlash(filepath.Join(cfg.OverlayPath, metaCachePath))
			}
			metaCacheDistPath := filepath.Join(ctx.Config.Dist, "gentoo", cfg.ID, metaCachePath)

			metaContent := generateMetaCacheContent(data, content)
			if metaContent != "" {
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
		}
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

func (g *publishGroup) applyVersionRetention(ctx *context.Context, repoClient any, repo client.Repo) ([]string, error) {
	dir := packageDir(g.cfg)
	stateRepo := repo
	if g.cfg.Repository.PullRequest.Enabled {
		stateRepo.Branch = g.cfg.Repository.PullRequest.Base.Branch
	}

	var ebuilds []string
	prefix := filepath.Base(dir) + "-"

	lister, ok := repoClient.(client.DirectoryLister)
	if ok {
		names, err := lister.ListDir(ctx, stateRepo, dir)
		if err != nil && !errors.Is(err, client.ErrNotImplemented) {
			return nil, err
		}
		for _, n := range names {
			if strings.HasPrefix(n, prefix) && strings.HasSuffix(n, ".ebuild") {
				ebuilds = append(ebuilds, n)
			}
		}
	}

	if len(ebuilds) == 0 {
		settings, err := loadOverlaySettings(ctx, g.cfg, repoClient, stateRepo)
		if err == nil && !settings.thin {
			manifestPath := filepath.ToSlash(filepath.Join(dir, "Manifest"))
			manifestLines, err := loadManifestLines(ctx, repoClient, stateRepo, manifestPath)
			if err == nil {
				for _, line := range manifestLines {
					fields := strings.Fields(line)
					if len(fields) >= 2 && fields[0] == "EBUILD" {
						ebuilds = append(ebuilds, fields[1])
					}
				}
			}
		}
	}

	if len(ebuilds) > 0 {
		switch g.cfg.ConflictResolution {
		case config.ConflictResolutionRevision:
			dl, ok := repoClient.(client.FileDownloader)
			if ok {
				g.updateVersions(ctx, dl, stateRepo, dir, prefix, ebuilds)
			}
		case config.ConflictResolutionOverwrite:
			// overwrites by default, no specific action required
		case config.ConflictResolutionFail:
			var newFiles []string
			for _, f := range g.files {
				name := filepath.Base(f.Path)
				if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, ".ebuild") {
					newFiles = append(newFiles, name)
				}
			}
			for _, nf := range newFiles {
				if slices.Contains(ebuilds, nf) {
					return nil, fmt.Errorf("ebuild %s already exists in %s", nf, dir)
				}
			}
		}
	}

	if !ok || g.cfg.KeepVersions <= 0 || g.cfg.VersionRetentionStrategy == "" {
		return nil, nil
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

	metaCacheDir := filepath.ToSlash(filepath.Join("metadata", "md5-cache", g.cfg.Category))
	if g.cfg.OverlayPath != "" {
		metaCacheDir = filepath.ToSlash(filepath.Join(g.cfg.OverlayPath, metaCacheDir))
	}
	metaCacheFiles := map[string]struct{}{}
	if g.cfg.MetaCache {
		cacheNames, err := lister.ListDir(ctx, stateRepo, metaCacheDir)
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
		metaCacheDir:   metaCacheDir,
		metaCacheFiles: metaCacheFiles,
		files:          &g.files,
		deletedEbuilds: &deletedEbuilds,
	}

	if g.cfg.VersionRetentionStrategy == config.VersionRetentionStrategyKeepPrereleases {
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
					(maxVersions["pre"] != nil && !v.GreaterThan(maxVersions["pre"])) ||
					(maxVersions["rc"] != nil && !v.GreaterThan(maxVersions["rc"])) ||
					(maxVersions["stable"] != nil && !v.GreaterThan(maxVersions["stable"])) {
					violates = true
				}
			case "beta":
				if (maxVersions["pre"] != nil && !v.GreaterThan(maxVersions["pre"])) ||
					(maxVersions["rc"] != nil && !v.GreaterThan(maxVersions["rc"])) ||
					(maxVersions["stable"] != nil && !v.GreaterThan(maxVersions["stable"])) {
					violates = true
				}
			case "pre":
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
	} else if g.cfg.VersionRetentionStrategy == config.VersionRetentionStrategyKeepLatest {
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

	var stateClient any = repoClient
	if g.cfg.Repository.Git.URL != "" {
		stateClient = client.NewGitUploadClient(repo.Branch)
	}

	stateRepo := repo
	if g.cfg.Repository.PullRequest.Enabled {
		stateRepo.Branch = g.cfg.Repository.PullRequest.Base.Branch
	}

	deletedEbuilds, err := g.applyVersionRetention(ctx, stateClient, repo)
	if err != nil {
		return err
	}

	settings, err := loadOverlaySettings(ctx, g.cfg, stateClient, stateRepo)
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

	metaCachePrefix := "metadata/md5-cache/"
	if g.cfg.OverlayPath != "" {
		metaCachePrefix = filepath.ToSlash(filepath.Join(g.cfg.OverlayPath, "metadata", "md5-cache")) + "/"
	}

	var filteredFiles []client.RepoFile
	for _, f := range g.files {
		if strings.HasPrefix(filepath.ToSlash(f.Path), metaCachePrefix) && !f.Delete && (!g.cfg.MetaCache || !metaCacheAllowed) {
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
					if err := d.DeleteFile(ctx, author, repo, f.Path, msg); err != nil && !errors.Is(err, client.ErrNotImplemented) {
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

func collectSuppressedIDs(cfg config.Gentoo) map[string]bool {
	suppressed := make(map[string]bool)
	add := func(items []config.GentooInstallItem) {
		for _, item := range items {
			if item.SrcID != "" {
				suppressed[item.SrcID] = true
			}
		}
	}
	add(cfg.Dobin)
	add(cfg.Doconfd)
	add(cfg.Doenvd)
	add(cfg.Doexe)
	add(cfg.Doheader)
	add(cfg.Doinitd)
	add(cfg.Doins)
	add(cfg.Dosbin)
	add(cfg.Dosym)
	add(cfg.Systemd)
	return suppressed
}

func packageDir(cfg config.Gentoo) string {
	pkgName := cfg.Name
	if cfg.Type == "bin" && !strings.HasSuffix(pkgName, "-bin") {
		pkgName += "-bin"
	}
	dir := filepath.ToSlash(filepath.Join(cfg.Category, pkgName))
	if cfg.OverlayPath != "" {
		dir = filepath.ToSlash(filepath.Join(cfg.OverlayPath, dir))
	}
	return dir
}

func ebuildRelPath(ctx *context.Context, cfg config.Gentoo) string {
	pkgName := cfg.Name
	if cfg.Type == "bin" && !strings.HasSuffix(pkgName, "-bin") {
		pkgName += "-bin"
	}
	dir := packageDir(cfg)
	return filepath.ToSlash(filepath.Join(dir, fmt.Sprintf("%s-%s.ebuild", pkgName, gentooVersion(ctx.Version))))
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

func stripComments(content []byte) []byte {
	var result []byte
	for line := range bytes.SplitSeq(content, []byte{'\n'}) {
		trimmed := bytes.TrimSpace(line)
		if len(trimmed) > 0 && trimmed[0] != '#' {
			result = append(result, line...)
			result = append(result, '\n')
		}
	}
	return result
}

func (g *publishGroup) updateVersions(ctx *context.Context, dl client.FileDownloader, stateRepo client.Repo, dir, prefix string, ebuilds []string) {
	for i := range g.files {
		if !strings.HasSuffix(g.files[i].Path, ".ebuild") || g.files[i].Delete {
			continue
		}

		fName := filepath.Base(g.files[i].Path)
		v := parseGentooVersion(fName, prefix)
		if v == nil {
			continue
		}

		maxR := -1
		var maxREbuild string
		for _, e := range ebuilds {
			ev := parseGentooVersion(e, prefix)
			if ev != nil && ev.baseEqual(v) && ev.revision > maxR {
				maxR = ev.revision
				maxREbuild = e
			}
		}

		if maxR == -1 || maxREbuild == "" {
			continue
		}

		existingEbuildContent, err := dl.DownloadFile(ctx, stateRepo, filepath.ToSlash(filepath.Join(dir, maxREbuild)))
		if err != nil {
			continue
		}

		strippedExisting := stripComments(existingEbuildContent)
		strippedNew := stripComments(g.files[i].Content)

		isDifferent := !bytes.Equal(strippedExisting, strippedNew)

		if !isDifferent {
			for _, f := range g.files {
				if f.Path == g.files[i].Path || f.Delete {
					continue
				}
				existingContent, dErr := dl.DownloadFile(ctx, stateRepo, f.Path)
				if dErr != nil || !bytes.Equal(existingContent, f.Content) {
					isDifferent = true
					break
				}
			}
		}

		if !isDifferent {
			log.WithField("file", fName).Debug("existing ebuild matches new ebuild content, not creating a new revision")
			g.files[i].Path = filepath.ToSlash(filepath.Join(dir, maxREbuild))
			continue
		}

		newRev := maxR + 1
		vStr := strings.TrimSuffix(strings.TrimPrefix(fName, prefix), ".ebuild")
		newEbuildName := fmt.Sprintf("%s%s-r%d.ebuild", prefix, vStr, newRev)
		newEbuildPath := filepath.ToSlash(filepath.Join(dir, newEbuildName))
		log.WithField("file", fName).WithField("new_file", newEbuildName).Info("ebuild content changed, bumping revision")
		g.files[i].Path = newEbuildPath

		oldMetaCachePrefix := filepath.ToSlash(filepath.Join("metadata", "md5-cache", g.cfg.Category, fmt.Sprintf("%s%s", prefix, vStr)))
		newMetaCachePath := filepath.ToSlash(filepath.Join("metadata", "md5-cache", g.cfg.Category, fmt.Sprintf("%s%s-r%d", prefix, vStr, newRev)))
		if g.cfg.OverlayPath != "" {
			oldMetaCachePrefix = filepath.ToSlash(filepath.Join(g.cfg.OverlayPath, oldMetaCachePrefix))
			newMetaCachePath = filepath.ToSlash(filepath.Join(g.cfg.OverlayPath, newMetaCachePath))
		}
		for j := range g.files {
			if g.files[j].Path == oldMetaCachePrefix {
				g.files[j].Path = newMetaCachePath
			}
		}
	}
}
