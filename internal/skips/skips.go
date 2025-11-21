// Package skips handles the skip logic.
package skips

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/goreleaser/goreleaser/v2/pkg/context"
)

type Key string

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
	Makeself       Key = "makeself"
	Chocolatey     Key = "chocolatey"
	Notarize       Key = "notarize"
	Archive        Key = "archive"
	MCP            Key = "mcp"
)

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

func Any(ctx *context.Context, keys ...Key) bool {
	for _, key := range keys {
		if ctx.Skips[string(key)] {
			return true
		}
	}
	return false
}

func Set(ctx *context.Context, keys ...Key) {
	for _, key := range keys {
		ctx.Skips[string(key)] = true
	}
}

var (
	SetRelease = set(Release)
	SetBuild   = set(Build)
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

type Keys []Key

func (keys Keys) String() string {
	ss := make([]string, len(keys))
	for i, key := range keys {
		ss[i] = string(key)
	}
	slices.Sort(ss)
	return strings.Join(ss, ", ")
}

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
	Makeself,
	Before,
	Notarize,
	Archive,
	MCP,
}

var Build = Keys{
	PreBuildHooks,
	PostBuildHooks,
	Validate,
	Before,
}
