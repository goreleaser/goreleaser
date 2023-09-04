package skips

import (
	"fmt"
	"strings"

	"github.com/goreleaser/goreleaser/pkg/context"
	"golang.org/x/exp/slices"
)

type Key string

const (
	BeforeBuildHooks Key = "before-hooks"
	PostBuildHooks   Key = "post-hooks"
	Publish          Key = "publish"
	Announce         Key = "announce"
	Sign             Key = "sign"
	Validate         Key = "validate"
	SBOM             Key = "sbom"
	Ko               Key = "ko"
	Docker           Key = "docker"
	Before           Key = "before"
)

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
	return strings.Join(ss, ", ")
}

var Release = Keys{
	Publish,
	Announce,
	Sign,
	Validate,
	SBOM,
	Ko,
	Docker,
	Before,
}

var Build = Keys{
	BeforeBuildHooks,
	PostBuildHooks,
	Validate,
	Before,
}
