// Package skips handles the --skip flag.
package skips

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

// Key is a skip name.
type Key string

// Constants for implemented skips.
const (
	PreBuildHooks  Key = "pre-hooks"
	PostBuildHooks Key = "post-hooks"
	Publish        Key = "publish"
	Announce       Key = "announce"
	Sign           Key = "sign"
	Validate       Key = "validate"
	SBOM           Key = "sbom"
	Ko             Key = "ko"
	Docker         Key = "docker"
	Before         Key = "before"
	Winget         Key = "winget"
	Snapcraft      Key = "snapcraft"
	Scoop          Key = "scoop"
	Homebrew       Key = "homebrew"
	Nix            Key = "nix"
	AUR            Key = "aur"
	AURSource      Key = "aur-source"
	NFPM           Key = "nfpm"
	Chocolatey     Key = "chocolatey"
	Notarize       Key = "notarize"
	Archive        Key = "archive"
)

// String stringifies the skip keys.
func String(ctx *context.Context) string {
	keys := slices.Sorted(maps.Keys(ctx.Skips))
	str := strings.Join(keys, ", ")
	if idx := strings.LastIndex(str, ","); idx > -1 {
		comma := ""
		if len(keys) > 2 {
			comma = ","
		}
		str = str[:idx] + comma + " and" + str[idx+1:]
	}
	return str
}

// Any checks if any of the keys are set.
func Any(ctx *context.Context, keys ...Key) bool {
	for _, key := range keys {
		if ctx.Skips[string(key)] {
			return true
		}
	}
	return false
}

// Set sets the keys in the context.
func Set(ctx *context.Context, keys ...Key) {
	for _, key := range keys {
		ctx.Skips[string(key)] = true
	}
}

var (
	// SetRelease are the allowed skips for the release pipeline.
	SetRelease = set(Release)

	// SetBuild are the allowed skips for the build pipeline.
	SetBuild = set(Build)
)

func set(allowed Keys) func(ctx *context.Context, keys ...string) error {
	return func(ctx *context.Context, keys ...string) error {
		for _, key := range keys {
			if key == "" {
				continue
			}
			if !slices.Contains(allowed, Key(key)) {
				return fmt.Errorf("--skip=%s is not allowed. Valid options for skip are [%s]", key, allowed)
			}
			ctx.Skips[key] = true
		}
		return nil
	}
}

// Keys is a key slice.
type Keys []Key

func (keys Keys) String() string {
	ss := make([]string, len(keys))
	for i, key := range keys {
		ss[i] = string(key)
	}
	slices.Sort(ss)
	return strings.Join(ss, ", ")
}

// Complete returns the keys that match the prefix.
func (keys Keys) Complete(prefix string) []string {
	var result []string
	for _, k := range keys {
		if strings.HasPrefix(string(k), strings.ToLower(prefix)) {
			result = append(result, string(k))
		}
	}
	slices.Sort(result)
	return result
}

// Release is the release pipeline skips.
var Release = Keys{
	Publish,
	Announce,
	Sign,
	Validate,
	SBOM,
	Ko,
	Docker,
	Winget,
	Chocolatey,
	Snapcraft,
	Scoop,
	Homebrew,
	Nix,
	AUR,
	AURSource,
	NFPM,
	Before,
	Notarize,
	Archive,
}

// Build is the build pipeline skips.
var Build = Keys{
	PreBuildHooks,
	PostBuildHooks,
	Validate,
	Before,
}
