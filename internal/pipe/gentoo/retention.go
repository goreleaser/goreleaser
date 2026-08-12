package gentoo

import (
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/goreleaser/goreleaser/v2/internal/client"
)

var gentooPrereleaseRe = regexp.MustCompile(`-(alpha|beta|pre|rc|p)[.\-]?(\d*)`)

func gentooVersion(v string) string {
	return gentooPrereleaseRe.ReplaceAllString(v, "_${1}${2}")
}

type suffixKind int

const (
	suffixAlpha suffixKind = 1
	suffixBeta  suffixKind = 2
	suffixPre   suffixKind = 3
	suffixRc    suffixKind = 4
	suffixP     suffixKind = 5
)

type gentooSuffix struct {
	kind suffixKind
	val  int
}

type parsedGentooVersion struct {
	raw        string
	baseNum    []int
	baseLetter rune
	suffixes   []gentooSuffix
	revision   int
}

func (v *parsedGentooVersion) Compare(other *parsedGentooVersion) int {
	if v == nil && other == nil {
		return 0
	}
	if v == nil {
		return -1
	}
	if other == nil {
		return 1
	}

	maxLen := max(len(v.baseNum), len(other.baseNum))
	for i := range maxLen {
		var n1, n2 int
		if i < len(v.baseNum) {
			n1 = v.baseNum[i]
		}
		if i < len(other.baseNum) {
			n2 = other.baseNum[i]
		}
		if n1 < n2 {
			return -1
		}
		if n1 > n2 {
			return 1
		}
	}

	if v.baseLetter < other.baseLetter {
		return -1
	}
	if v.baseLetter > other.baseLetter {
		return 1
	}

	cmpSuffix := compareGentooSuffixes(v.suffixes, other.suffixes)
	if cmpSuffix != 0 {
		return cmpSuffix
	}

	if v.revision < other.revision {
		return -1
	}
	if v.revision > other.revision {
		return 1
	}

	return 0
}

func (v *parsedGentooVersion) GreaterThan(other *parsedGentooVersion) bool {
	return v.Compare(other) > 0
}

func (v *parsedGentooVersion) baseEqual(other *parsedGentooVersion) bool {
	if v == nil || other == nil {
		return v == other
	}
	if len(v.baseNum) != len(other.baseNum) || v.baseLetter != other.baseLetter || len(v.suffixes) != len(other.suffixes) {
		return false
	}
	for i := range v.baseNum {
		if v.baseNum[i] != other.baseNum[i] {
			return false
		}
	}
	for i := range v.suffixes {
		if v.suffixes[i] != other.suffixes[i] {
			return false
		}
	}
	return true
}

func compareGentooSuffixes(s1, s2 []gentooSuffix) int {
	if len(s1) == 0 && len(s2) == 0 {
		return 0
	}
	if len(s1) == 0 {
		if s2[0].kind == suffixP {
			return -1 // release < _p
		}
		return 1 // release > _alpha, _beta, _pre, _rc
	}
	if len(s2) == 0 {
		if s1[0].kind == suffixP {
			return 1 // _p > release
		}
		return -1 // _alpha, _beta, _pre, _rc < release
	}

	maxLen := max(len(s1), len(s2))
	for i := range maxLen {
		if i >= len(s1) {
			if s2[i].kind == suffixP {
				return -1
			}
			return 1
		}
		if i >= len(s2) {
			if s1[i].kind == suffixP {
				return 1
			}
			return -1
		}
		if s1[i].kind < s2[i].kind {
			return -1
		}
		if s1[i].kind > s2[i].kind {
			return 1
		}
		if s1[i].val < s2[i].val {
			return -1
		}
		if s1[i].val > s2[i].val {
			return 1
		}
	}
	return 0
}

var gentooSuffixTokenRe = regexp.MustCompile(`_(alpha|beta|pre|rc|p)(\d*)$`)

func parseGentooVersion(n, prefix string) *parsedGentooVersion {
	vStr := strings.TrimSuffix(strings.TrimPrefix(n, prefix), ".ebuild")
	if vStr == "" || vStr == n {
		return nil
	}

	var rev int
	if idx := strings.LastIndex(vStr, "-r"); idx != -1 {
		if parsedRev, err := strconv.Atoi(vStr[idx+2:]); err == nil {
			rev = parsedRev
			vStr = vStr[:idx]
		}
	}

	var suffixes []gentooSuffix
	for {
		loc := gentooSuffixTokenRe.FindStringSubmatchIndex(vStr)
		if loc == nil {
			break
		}
		kindStr := vStr[loc[2]:loc[3]]
		valStr := vStr[loc[4]:loc[5]]
		val := 0
		if valStr != "" {
			var err error
			val, err = strconv.Atoi(valStr)
			if err != nil {
				return nil
			}
		}

		var kind suffixKind
		switch kindStr {
		case "alpha":
			kind = suffixAlpha
		case "beta":
			kind = suffixBeta
		case "pre":
			kind = suffixPre
		case "rc":
			kind = suffixRc
		case "p":
			kind = suffixP
		default:
			return nil
		}

		suffixes = append([]gentooSuffix{{kind: kind, val: val}}, suffixes...)
		vStr = vStr[:loc[0]]
	}

	if vStr == "" {
		return nil
	}

	var letter rune
	lastChar := vStr[len(vStr)-1]
	if lastChar >= 'a' && lastChar <= 'z' {
		letter = rune(lastChar)
		vStr = vStr[:len(vStr)-1]
	}

	if vStr == "" {
		return nil
	}

	parts := strings.Split(vStr, ".")
	var baseNum []int
	for _, p := range parts {
		if p == "" {
			return nil
		}
		num, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		baseNum = append(baseNum, num)
	}

	return &parsedGentooVersion{
		raw:        n,
		baseNum:    baseNum,
		baseLetter: letter,
		suffixes:   suffixes,
		revision:   rev,
	}
}

func getVersionBucket(v *parsedGentooVersion) string {
	if v == nil || len(v.suffixes) == 0 {
		return "stable"
	}
	last := v.suffixes[len(v.suffixes)-1]
	switch last.kind {
	case suffixAlpha:
		return "alpha"
	case suffixBeta:
		return "beta"
	case suffixPre:
		return "pre"
	case suffixRc:
		return "rc"
	default:
		return "stable"
	}
}

type ebuildDeleter struct {
	dir            string
	metaCacheDir   string
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
	md5CachePath := path.Join(d.metaCacheDir, md5Name)
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
