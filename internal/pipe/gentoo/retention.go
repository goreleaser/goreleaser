package gentoo

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/goreleaser/goreleaser/v2/internal/client"
)

var gentooPrereleaseRe = regexp.MustCompile(`(?i)-(alpha|beta|pre|rc|p)[.\-]?(\d*)`)

func convertToGentooVersion(v string, from string) (string, error) {
	switch from {
	case "gentoo-version":
		converted := gentooPrereleaseRe.ReplaceAllStringFunc(v, func(m string) string {
			match := gentooPrereleaseRe.FindStringSubmatch(m)
			return "_" + strings.ToLower(match[1]) + match[2]
		})
		if parseGentooVersion(converted+".ebuild", "") == nil {
			return "", fmt.Errorf("version %q cannot be naturally represented in Gentoo", v)
		}
		return converted, nil
	default:
		return "", fmt.Errorf("unsupported version representation %v", from)
	}
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
	baseNumStr []string
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

	// Algorithm 3.2 & 3.3: Numeric components comparison
	if len(v.baseNum) > 0 && len(other.baseNum) > 0 {
		if v.baseNum[0] < other.baseNum[0] {
			return -1
		}
		if v.baseNum[0] > other.baseNum[0] {
			return 1
		}
	}

	minLen := min(len(v.baseNum), len(other.baseNum))
	for i := 1; i < minLen; i++ {
		s1 := v.baseNumStr[i]
		s2 := other.baseNumStr[i]
		if strings.HasPrefix(s1, "0") || strings.HasPrefix(s2, "0") {
			s1Trim := strings.TrimRight(s1, "0")
			s2Trim := strings.TrimRight(s2, "0")
			if s1Trim < s2Trim {
				return -1
			}
			if s1Trim > s2Trim {
				return 1
			}
		} else {
			if v.baseNum[i] < other.baseNum[i] {
				return -1
			}
			if v.baseNum[i] > other.baseNum[i] {
				return 1
			}
		}
	}

	if len(v.baseNum) < len(other.baseNum) {
		return -1
	}
	if len(v.baseNum) > len(other.baseNum) {
		return 1
	}

	// Algorithm 3.4: Letter components comparison
	if v.baseLetter < other.baseLetter {
		return -1
	}
	if v.baseLetter > other.baseLetter {
		return 1
	}

	// Algorithm 3.5 & 3.6: Suffixes comparison
	cmpSuffix := compareGentooSuffixes(v.suffixes, other.suffixes)
	if cmpSuffix != 0 {
		return cmpSuffix
	}

	// Algorithm 3.7: Revision comparison
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
		if v.baseNum[i] != other.baseNum[i] || v.baseNumStr[i] != other.baseNumStr[i] {
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
	var baseNumStr []string
	for _, p := range parts {
		if p == "" {
			return nil
		}
		num, err := strconv.Atoi(p)
		if err != nil {
			return nil
		}
		baseNum = append(baseNum, num)
		baseNumStr = append(baseNumStr, p)
	}

	return &parsedGentooVersion{
		raw:        n,
		baseNum:    baseNum,
		baseNumStr: baseNumStr,
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

func determineKeepLatestDeletions(ebuilds, newFiles []string, prefix string, keepVersions int) []string {
	var allEbuilds []string
	allEbuilds = append(allEbuilds, ebuilds...)
	for _, n := range newFiles {
		if !slices.Contains(ebuilds, n) {
			allEbuilds = append(allEbuilds, n)
		}
	}

	slices.SortFunc(allEbuilds, func(i, j string) int {
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

	var toDelete []string
	if len(allEbuilds) > keepVersions {
		keptFiles := allEbuilds[:keepVersions]
		for _, n := range ebuilds {
			if !slices.Contains(keptFiles, n) {
				toDelete = append(toDelete, n)
			}
		}
	}
	return toDelete
}
