package gentoo

import (
	"bytes"
	"crypto/md5"
	_ "embed"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"text/template"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

//go:embed templates/ebuild.tmpl
var ebuildTemplate string

//go:embed templates/md5-cache.tmpl
var metaCacheTemplate string

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

type ebuildData struct {
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
}

func (d ebuildData) Validate() error {
	for _, sym := range d.Dosym {
		if sym.Target == "" {
			return errors.New("gentoo.dosym requires a destination")
		}
	}
	return nil
}

func (d ebuildData) RenderEbuild() (string, error) {
	var buf bytes.Buffer
	if err := template.Must(template.New("ebuild").Parse(ebuildTemplate)).Execute(&buf, d); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (d ebuildData) SortedUseFlags() []string {
	var useFlags []string
	for _, flag := range d.UseFlags {
		if flag.Flag != "" {
			useFlags = append(useFlags, flag.Flag)
		}
	}
	slices.Sort(useFlags)
	return slices.Compact(useFlags)
}

func (d ebuildData) FormattedSrcURIs() []string {
	var srcURIs []string
	for _, art := range d.Archs {
		if art.Keyword != "" && art.URI != "" {
			srcURIs = append(srcURIs, fmt.Sprintf("%s? ( %s )", art.Keyword, art.URI))
		}
	}
	return srcURIs
}

func (d ebuildData) RenderMetaCache(ebuildContent string) (string, error) {
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
		IUSE:        strings.Join(d.SortedUseFlags(), " "),
		Keywords:    d.Keywords,
		License:     d.License,
		SrcURI:      strings.Join(d.FormattedSrcURIs(), " "),
		MD5:         md5Hex,
	}

	var buf bytes.Buffer
	if err := template.Must(template.New("md5-cache").Parse(metaCacheTemplate)).Execute(&buf, tmplData); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func generateMetaCacheContent(data ebuildData, ebuildContent string) string {
	meta, err := data.RenderMetaCache(ebuildContent)
	if err != nil {
		return ""
	}
	return meta
}

type extraFilesProcessor struct {
	cfg        config.Gentoo
	arches     []*artifact.Artifact
	extraFiles map[string]string
}

func newExtraFilesProcessor(cfg config.Gentoo, arches []*artifact.Artifact, extraFiles map[string]string) *extraFilesProcessor {
	return &extraFilesProcessor{
		cfg:        cfg,
		arches:     arches,
		extraFiles: extraFiles,
	}
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

func normalizeArchivePath(pathStr string) string {
	return strings.TrimPrefix(filepath.ToSlash(filepath.Clean(pathStr)), "./")
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
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to read extra file %s: %w", name, err)
		}
		if bytes.IndexByte(buf[:n], 0) != -1 {
			return fmt.Errorf("extra file %s appears to be a binary file. Gentoo policy forbids binary files in the files/ directory", name)
		}
	}
	return nil
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
			Dir:    path.Dir(filepath.ToSlash(d.Dst)),
			Base:   path.Base(filepath.ToSlash(d.Dst)),
			Use:    d.Use,
		})
	}
	return items
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

func (v *extraFilesProcessor) InstallExtraFiles(ctx *context.Context, ebuildPath string) error {
	for name, src := range v.extraFiles {
		destName, err := gentooExtraFilePath(name)
		if err != nil {
			return err
		}
		dst := filepath.Join(filepath.Dir(ebuildPath), destName)
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
				ebuildExtra:     v.cfg,
				ebuildPathExtra: path.Join(filepath.ToSlash(filepath.Dir(v.cfg.Path)), filepath.ToSlash(destName)),
			},
		})
	}
	return nil
}

func gentooArch(goarch string) string {
	switch goarch {
	case "386":
		return "x86"
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
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

func gentooExtraFilePath(name string) (string, error) {
	pathStr := filepath.ToSlash(name)
	pathStr = strings.TrimPrefix(pathStr, "files/")
	if pathStr == "" || path.IsAbs(pathStr) || pathStr != path.Clean(pathStr) || strings.HasPrefix(pathStr, "../") || pathStr == ".." {
		return "", fmt.Errorf("gentoo extra file name %q must remain within the files directory", name)
	}
	return path.Join("files", pathStr), nil
}

func gentooUseFlags(cfg config.Gentoo) []config.GentooUseFlag {
	docFlag := config.GentooUseFlag{
		Flag:        "doc",
		Description: "Install README man page and other docs",
	}
	useFlagsLen := len(cfg.UseFlags)
	capHint := useFlagsLen
	if capHint < int(^uint(0)>>1) {
		capHint++
	}
	flags := make([]config.GentooUseFlag, 1, capHint)
	for _, flag := range cfg.UseFlags {
		if strings.TrimLeft(flag.Flag, "+-") == "doc" {
			docFlag = flag
			continue
		}
		flags = append(flags, flag)
	}
	flags[0] = docFlag
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
