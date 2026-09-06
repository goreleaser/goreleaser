package docker

import (
	"slices"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestValidateManifester(t *testing.T) {
	tests := []struct {
		use       string
		wantError string
	}{
		{use: "docker"},
		{use: "buildx", wantError: "docker manifest: invalid use: buildx, valid options are [docker]"},
	}

	for _, tt := range tests {
		t.Run(tt.use, func(t *testing.T) {
			err := validateManifester(tt.use)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}

type recordingManifester struct {
	createCalls atomic.Int32
	pushCalls   atomic.Int32
	images      []string
}

func (m *recordingManifester) Create(_ *context.Context, _ string, images, _ []string) error {
	m.createCalls.Add(1)
	m.images = slices.Clone(images)
	return nil
}

func (m *recordingManifester) Push(*context.Context, string, []string) (string, error) {
	m.pushCalls.Add(1)
	return "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", nil
}

func registerTestManifester(t *testing.T, name string, impl manifester) {
	t.Helper()

	lock.Lock()
	previous, existed := manifesters[name]
	manifesters[name] = impl
	lock.Unlock()
	t.Cleanup(func() {
		lock.Lock()
		defer lock.Unlock()
		if existed {
			manifesters[name] = previous
		} else {
			delete(manifesters, name)
		}
	})
}

func TestManifestPublishSkipsRenderedEmptyImages(t *testing.T) {
	t.Run("all rendered empty", func(t *testing.T) {
		const use = "test-rendered-empty"
		manifester := &recordingManifester{}
		registerTestManifester(t, use, manifester)
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			DockerManifests: []config.DockerManifest{
				{
					Use:            use,
					NameTemplate:   "example.invalid/app:latest",
					ImageTemplates: []string{`{{ if .IsSnapshot }}example.invalid/app:snapshot{{ end }}`},
				},
			},
		})

		require.NoError(t, ManifestPipe{}.Default(ctx))
		err := ManifestPipe{}.Publish(ctx)
		require.ErrorContains(t, err, "manifest has no images")
		require.Equal(t, int32(0), manifester.createCalls.Load())
		require.Equal(t, int32(0), manifester.pushCalls.Load())
	})

	t.Run("mixed rendered images", func(t *testing.T) {
		const use = "test-rendered-mixed"
		manifester := &recordingManifester{}
		registerTestManifester(t, use, manifester)
		ctx := testctx.WrapWithCfg(t.Context(), config.Project{
			DockerManifests: []config.DockerManifest{
				{
					Use:          use,
					NameTemplate: "example.invalid/app:latest",
					ImageTemplates: []string{
						`{{ if .IsSnapshot }}example.invalid/app:snapshot{{ end }}`,
						"example.invalid/app:release",
					},
				},
			},
		})
		ctx.Artifacts.Add(&artifact.Artifact{
			Type:  artifact.DockerImage,
			Name:  "example.invalid/app:release",
			Extra: map[string]any{artifact.ExtraDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		})

		require.NoError(t, ManifestPipe{}.Default(ctx))
		require.NoError(t, ManifestPipe{}.Publish(ctx))
		require.Equal(t, int32(1), manifester.createCalls.Load())
		require.Equal(t, int32(1), manifester.pushCalls.Load())
		require.Equal(t, []string{"example.invalid/app:release@sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}, manifester.images)
	})
}

func Test_manifestImages(t *testing.T) {
	const someImage = "repo/image:tag"

	tests := []struct {
		name        string
		artifacts   []*artifact.Artifact
		templates   []string
		want        []string
		errContains string
		wantErr     bool
	}{
		{
			name:        "no templates",
			want:        []string{},
			wantErr:     true,
			errContains: "manifest has no images",
		},
		{
			name:        "empty template string",
			templates:   []string{""},
			want:        []string{},
			wantErr:     true,
			errContains: "manifest has no images",
		},
		{
			name: "single image with digest",
			artifacts: []*artifact.Artifact{
				{
					Type:  artifact.DockerImage,
					Name:  someImage,
					Extra: map[string]any{artifact.ExtraDigest: "sha256:123"},
				},
			},
			templates: []string{someImage},
			want:      []string{someImage + "@sha256:123"},
		},
		{
			name: "single image without digest",
			artifacts: []*artifact.Artifact{
				{
					Type:  artifact.DockerImage,
					Name:  someImage,
					Extra: map[string]any{},
				},
			},
			templates: []string{someImage},
			want:      []string{someImage},
		},
		{
			name: "template with no matching artifact",
			artifacts: []*artifact.Artifact{
				{
					Type:  artifact.DockerImage,
					Name:  "other/image:tag",
					Extra: map[string]any{artifact.ExtraDigest: "sha"},
				},
			},
			templates: []string{someImage},
			want:      []string{someImage},
		},
		{
			name: "multiple templates with some empty",
			artifacts: []*artifact.Artifact{
				{
					Type:  artifact.DockerImage,
					Name:  "a:1",
					Extra: map[string]any{artifact.ExtraDigest: "d1"},
				},
				{
					Type:  artifact.DockerImage,
					Name:  "b:2",
					Extra: map[string]any{},
				},
			},
			templates: []string{
				"a:1",
				`{{ if contains "aa" "a" }}{{""}}{{ else }}{{ "fail" }}{{ end }}`,
				"b:2",
				"",
			},
			want: []string{"a:1@d1", "b:2"},
		},
		{
			name:        "incorrect template",
			templates:   []string{"{{ if eq (INCORRECT true) }}"},
			wantErr:     true,
			errContains: "template: failed to apply",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{})
			for _, art := range tt.artifacts {
				ctx.Artifacts.Add(art)
			}

			manifest := config.DockerManifest{
				ImageTemplates: tt.templates,
			}

			got, err := manifestImages(ctx, manifest)

			if tt.wantErr {
				require.Error(t, err)
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error to contain %q, got: %v", tt.errContains, err)
				}
			} else {
				require.NoError(t, err)
				if !slices.Equal(got, tt.want) {
					t.Fatalf("unexpected output: want: %v, got: %v", tt.want, got)
				}
			}
		})
	}
}
