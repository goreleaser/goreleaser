package skips

import "github.com/goreleaser/goreleaser/pkg/context"

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

func SetS(ctx *context.Context, keys ...string) {
	for _, key := range keys {
		ctx.Skips[key] = true
	}
}

var Release = []Key{
	Publish,
	Announce,
	Sign,
	Validate,
	SBOM,
	Ko,
	Docker,
	Before,
}

var Build = []Key{
	BeforeBuildHooks,
	PostBuildHooks,
	Validate,
	Before,
}
