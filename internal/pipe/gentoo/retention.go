package gentoo

import (
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/goreleaser/goreleaser/v2/internal/client"
)

var gentooPrereleaseRe = regexp.MustCompile(`-(alpha|beta|pre|rc|p)[.\-]?(\d*)`)

func gentooVersion(v string) string {
	return gentooPrereleaseRe.ReplaceAllString(v, "_${1}${2}")
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
	metaCacheFiles map[string]struct{}
	files          *[]client.RepoFile
	deletedEbuilds *[]string
}

func (d *ebuildDeleter) Delete(ebuildName string) {
	*d.files = append(*d.files, client.RepoFile{Path: path.Join(d.dir, ebuildName), Delete: true})
	*d.deletedEbuilds = append(*d.deletedEbuilds, ebuildName)
	md5Name := strings.TrimSuffix(ebuildName, ".ebuild")
	if _, ok := d.metaCacheFiles[md5Name]; !ok {
		return
	}
	md5CachePath := path.Join("metadata", "md5-cache", d.category, md5Name)
	*d.files = append(*d.files, client.RepoFile{Path: md5CachePath, Delete: true})
}

func countNewEbuilds(ebuilds, newFiles []string, bucket func(string) string) map[string]int {
	counts := map[string]int{}
	for _, file := range newFiles {
		if !slices.Contains(ebuilds, file) {
			counts[bucket(file)]++
		}
	}
	return counts
}
