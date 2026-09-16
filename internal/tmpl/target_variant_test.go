package tmpl

import (
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/stretchr/testify/require"
)

func TestTargetVariant(t *testing.T) {
	ctx := testctx.Wrap(t.Context())
	for name, tc := range map[string]struct {
		artifact artifact.Artifact
		expect   string
	}{
		"no variant":        {artifact.Artifact{Goos: "linux", Goarch: "amd64"}, ""},
		"default goamd64":   {artifact.Artifact{Goamd64: "v1"}, ""},
		"default goarm64":   {artifact.Artifact{Goarm64: "v8.0"}, ""},
		"default go386":     {artifact.Artifact{Go386: "sse2"}, ""},
		"default goppc64":   {artifact.Artifact{Goppc64: "power8"}, ""},
		"default goriscv64": {artifact.Artifact{Goriscv64: "rva20u64"}, ""},
		"goamd64":           {artifact.Artifact{Goamd64: "v3"}, "v3"},
		"goarm64":           {artifact.Artifact{Goarm64: "v9.0"}, "v9.0"},
		"go386":             {artifact.Artifact{Go386: "softfloat"}, "softfloat"},
		"goppc64":           {artifact.Artifact{Goppc64: "power10"}, "power10"},
		"goriscv64":         {artifact.Artifact{Goriscv64: "rva22u64"}, "rva22u64"},
		// goarm and gomips always render, even at their default, as they did
		// before this function existed.
		"goarm":            {artifact.Artifact{Goarm: "7"}, "v7"},
		"gomips":           {artifact.Artifact{Gomips: "hardfloat"}, "_hardfloat"},
		"goarm64 features": {artifact.Artifact{Goarm64: "v9.0,lse,crypto"}, "v9.0-lse-crypto"},
		"default goarm64 with features": {
			artifact.Artifact{Goarm64: "v8.0,lse"},
			"v8.0-lse",
		},
		"abi": {
			artifact.Artifact{Extra: map[string]any{KeyAbi: "musl"}},
			"_musl",
		},
		"all of them": {
			artifact.Artifact{
				Goarm:   "7",
				Gomips:  "softfloat",
				Goamd64: "v3",
				Extra:   map[string]any{KeyAbi: "gnu"},
			},
			"v7_softfloatv3_gnu",
		},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := New(ctx).WithArtifact(&tc.artifact).Apply("{{ targetVariant . }}")
			require.NoError(t, err)
			require.Equal(t, tc.expect, got)
			require.NotContains(t, got, ",")
		})
	}
}

// The variant is meant to be used on names, so it must not error out when
// applied to a template that has no artifact in scope.
func TestTargetVariantWithoutArtifact(t *testing.T) {
	got, err := New(testctx.Wrap(t.Context())).Apply("{{ targetVariant . }}")
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestTargetVariantInvalidArgument(t *testing.T) {
	_, err := New(testctx.Wrap(t.Context())).Apply("{{ targetVariant .Env }}")
	require.ErrorContains(t, err, "use it as '{{ targetVariant . }}'")
}
