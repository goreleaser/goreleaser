package gentoo

import (
	"bytes"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/xml"
	"errors"
	"fmt"
	"hash"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/crypto/blake2b"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/client"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

type gentooInnerNode struct {
	XMLName xml.Name
	Content string            `xml:",chardata"`
	Attrs   []xml.Attr        `xml:",any,attr"`
	Nodes   []gentooInnerNode `xml:",any"`
}

type gentooMaintainer struct {
	Type  string `xml:"type,attr,omitempty"`
	Email string `xml:"email"`
	Name  string `xml:"name,omitempty"`
}

type gentooUpstream struct {
	BugsTo string            `xml:"bugs-to,omitempty"`
	Doc    string            `xml:"doc,omitempty"`
	Attrs  []xml.Attr        `xml:",any,attr"`
	Nodes  []gentooInnerNode `xml:",any"`
}

type gentooUseFlag struct {
	XMLName xml.Name   `xml:"flag"`
	Name    string     `xml:"name,attr"`
	Value   string     `xml:",chardata"`
	Attrs   []xml.Attr `xml:",any,attr"`
}

type gentooUse struct {
	XMLName xml.Name          `xml:"use"`
	Flags   []gentooUseFlag   `xml:"flag"`
	Attrs   []xml.Attr        `xml:",any,attr"`
	Nodes   []gentooInnerNode `xml:",any"`
}

type gentooMetadata struct {
	XMLName     xml.Name           `xml:"pkgmetadata"`
	Attrs       []xml.Attr         `xml:",any,attr"`
	Maintainers []gentooMaintainer `xml:"maintainer"`
	Use         *gentooUse         `xml:"use,omitempty"`
	Upstream    *gentooUpstream    `xml:"upstream,omitempty"`
	InnerNodes  []gentooInnerNode  `xml:",any"`
}

func (m *gentooMetadata) AddMaintainers(maintainers []config.GentooMaintainer) error {
	for _, main := range maintainers {
		if main.Email == "" {
			return errors.New("gentoo maintainer email is required")
		}
		exists := false
		for _, em := range m.Maintainers {
			if em.Email == main.Email {
				exists = true
				break
			}
		}
		if !exists {
			m.Maintainers = append(m.Maintainers, gentooMaintainer{
				Type:  "person",
				Email: main.Email,
				Name:  main.Name,
			})
		}
	}
	return nil
}

func (m *gentooMetadata) AddUseFlags(flags []config.GentooUseFlag) {
	if m.Use == nil {
		m.Use = &gentooUse{}
	}
	configuredFlags := make(map[string]string)
	configuredFlags["doc"] = "Install README man page and other docs"
	for _, flag := range flags {
		if flag.Description != "" {
			configuredFlags[strings.TrimLeft(flag.Flag, "+-")] = flag.Description
		}
	}

	var configuredFlagNames []string
	for k := range configuredFlags {
		configuredFlagNames = append(configuredFlagNames, k)
	}
	slices.Sort(configuredFlagNames)

	for _, k := range configuredFlagNames {
		v := configuredFlags[k]
		exists := false
		for _, ef := range m.Use.Flags {
			if ef.Name == k {
				exists = true
				break
			}
		}
		if !exists {
			m.Use.Flags = append(m.Use.Flags, gentooUseFlag{
				Name:  k,
				Value: v,
			})
		}
	}
}

func (m *gentooMetadata) SetUpstream(bugsTo, homepage string) {
	if bugsTo == "" && homepage == "" {
		return
	}
	if m.Upstream == nil {
		m.Upstream = &gentooUpstream{}
	}
	if bugsTo != "" {
		m.Upstream.BugsTo = bugsTo
	}
	if homepage != "" {
		m.Upstream.Doc = homepage
	}
}

func (m *gentooMetadata) Marshal() ([]byte, error) {
	content, err := xml.MarshalIndent(m, "", "\t")
	if err != nil {
		return nil, err
	}
	header := []byte("<?xml version=\"1.0\" encoding=\"UTF-8\"?>\n<!DOCTYPE pkgmetadata SYSTEM \"https://www.gentoo.org/dtd/metadata.dtd\">\n")
	return append(header, append(content, '\n')...), nil
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

func generateManifestLine(recordType, filename, pathStr string, content []byte, manifestHashes []string) (string, error) {
	var r io.Reader
	var size int64

	if content == nil && pathStr != "" {
		info, err := os.Stat(pathStr)
		if err != nil {
			return "", err
		}
		size = info.Size()

		f, err := os.Open(pathStr)
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

	metadataPath := path.Join(dir, "metadata.xml")
	manifestPath := path.Join(dir, "Manifest")

	if len(cfg.Maintainers) > 0 || cfg.BugsTo != "" || cfg.Homepage != "" || len(cfg.UseFlags) > 0 {
		meta := gentooMetadata{}
		if dl, ok := repoClient.(client.FileDownloader); ok {
			content, err := dl.DownloadFile(ctx, repo, metadataPath)
			if err == nil {
				_ = xml.Unmarshal(content, &meta)
			} else if !errors.Is(err, client.ErrNotFound) && !errors.Is(err, client.ErrNotImplemented) {
				return fmt.Errorf("failed to download metadata.xml: %w", err)
			}
		}

		meta.AddUseFlags(cfg.UseFlags)
		if err := meta.AddMaintainers(cfg.Maintainers); err != nil {
			return err
		}
		meta.SetUpstream(cfg.BugsTo, cfg.Homepage)

		content, err := meta.Marshal()
		if err != nil {
			return err
		}
		*files = append(*files, client.RepoFile{
			Content: content,
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

func loadManifestLines(ctx *context.Context, repoClient client.Client, repo client.Repo, manifestPath string) ([]string, error) {
	if dl, ok := repoClient.(client.FileDownloader); ok {
		content, err := dl.DownloadFile(ctx, repo, manifestPath)
		if err == nil {
			return strings.FieldsFunc(string(content), func(r rune) bool { return r == '\n' || r == '\r' }), nil
		}
		if !errors.Is(err, client.ErrNotFound) && !errors.Is(err, client.ErrNotImplemented) {
			return nil, fmt.Errorf("failed to download Manifest: %w", err)
		}
	}
	return nil, nil
}

func manifestFileInfo(filePath, packageDir string) (string, string) {
	pathStr := filepath.ToSlash(filePath)
	filesDir := path.Join(packageDir, "files")
	if pathStr == filesDir || strings.HasPrefix(pathStr, filesDir+"/") {
		return "AUX", strings.TrimPrefix(pathStr, filesDir+"/")
	}
	if strings.HasSuffix(pathStr, ".ebuild") {
		return "EBUILD", path.Base(pathStr)
	}
	return "MISC", path.Base(pathStr)
}
