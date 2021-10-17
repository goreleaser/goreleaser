package sign

import (
	"os"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

func TestDockerSignDescription(t *testing.T) {
	require.NotEmpty(t, DockerPipe{}.String())
}

func TestDockerSignDefault(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			DockerSigns: []config.Sign{{}},
		},
	}
	err := DockerPipe{}.Default(ctx)
	require.NoError(t, err)
	require.Equal(t, ctx.Config.DockerSigns[0].Cmd, "cosign")
	require.Equal(t, ctx.Config.DockerSigns[0].Signature, "")
	require.Equal(t, ctx.Config.DockerSigns[0].Args, []string{"sign", "-key=cosign.key", "$artifact"})
	require.Equal(t, ctx.Config.DockerSigns[0].Artifacts, "none")
}

func TestDockerSignDisabled(t *testing.T) {
	ctx := context.New(config.Project{})
	ctx.Config.DockerSigns = []config.Sign{
		{Artifacts: "none"},
	}
	err := DockerPipe{}.Publish(ctx)
	require.EqualError(t, err, "artifact signing is disabled")
}

func TestDockerSignInvalidArtifacts(t *testing.T) {
	ctx := context.New(config.Project{})
	ctx.Config.DockerSigns = []config.Sign{
		{Artifacts: "foo"},
	}
	err := DockerPipe{}.Publish(ctx)
	require.EqualError(t, err, "invalid list of artifacts to sign: foo")
}

func TestDockerSignArtifacts(t *testing.T) {
	key := "testdata/cosign/cosign.key"
	cmd := "sh"
	args := []string{"-c", "echo ${artifact} > ${signature} && cosign sign -key=" + key + " -upload=false ${artifact} > ${signature}"}
	password := "password"

	img1 := "ghcr.io/caarlos0/goreleaser-docker-manifest-actions-example:1.2.1-amd64"
	img2 := "ghcr.io/caarlos0/goreleaser-docker-manifest-actions-example:1.2.1-arm64v8"
	man1 := "ghcr.io/caarlos0/goreleaser-docker-manifest-actions-example:1.2.1"

	for name, cfg := range map[string]struct {
		Signs    []config.Sign
		Expected []string
	}{
		"no signature file": {
			Expected: nil, // no sigs
			Signs: []config.Sign{
				{
					Artifacts: "all",
					Stdin:     &password,
					Cmd:       "cosign",
					Args:      []string{"sign", "-key=" + key, "-upload=false", "${artifact}"},
				},
			},
		},
		"sign all": {
			Expected: []string{
				"testdata/cosign/all_img1.sig",
				"testdata/cosign/all_img2.sig",
				"testdata/cosign/all_man1.sig",
			},
			Signs: []config.Sign{
				{
					Artifacts: "all",
					Stdin:     &password,
					Signature: `testdata/cosign/all_${artifactID}.sig`,
					Cmd:       cmd,
					Args:      args,
				},
			},
		},
		"sign all filtering id": {
			Expected: []string{"testdata/cosign/all_filter_by_id_img2.sig"},
			Signs: []config.Sign{
				{
					Artifacts: "all",
					IDs:       []string{"img2"},
					Stdin:     &password,
					Signature: "testdata/cosign/all_filter_by_id_${artifactID}.sig",
					Cmd:       cmd,
					Args:      args,
				},
			},
		},
		"sign images only": {
			Expected: []string{
				"testdata/cosign/images_img1.sig",
				"testdata/cosign/images_img2.sig",
			},
			Signs: []config.Sign{
				{
					Artifacts: "images",
					Stdin:     &password,
					Signature: "testdata/cosign/images_${artifactID}.sig",
					Cmd:       cmd,
					Args:      args,
				},
			},
		},
		"sign manifests only": {
			Expected: []string{"testdata/cosign/manifests_man1.sig"},
			Signs: []config.Sign{
				{
					Artifacts: "manifests",
					Stdin:     &password,
					Signature: "testdata/cosign/manifests_${artifactID}.sig",
					Cmd:       cmd,
					Args:      args,
				},
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.New(config.Project{})
			ctx.Config.DockerSigns = cfg.Signs

			t.Cleanup(func() {
				for _, f := range cfg.Expected {
					require.NoError(t, os.Remove(f))
				}
			})

			ctx.Artifacts.Add(&artifact.Artifact{
				Name: img1,
				Path: img1,
				Type: artifact.DockerImage,
				Extra: map[string]interface{}{
					artifact.ExtraID: "img1",
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name: img2,
				Path: img2,
				Type: artifact.DockerImage,
				Extra: map[string]interface{}{
					artifact.ExtraID: "img2",
				},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name: man1,
				Path: man1,
				Type: artifact.DockerManifest,
				Extra: map[string]interface{}{
					artifact.ExtraID: "man1",
				},
			})

			require.NoError(t, DockerPipe{}.Default(ctx))
			require.NoError(t, DockerPipe{}.Publish(ctx))
			var sigs []string
			for _, sig := range ctx.Artifacts.Filter(artifact.ByType(artifact.Signature)).List() {
				sigs = append(sigs, sig.Name)
			}
			require.Equal(t, cfg.Expected, sigs)
		})
	}
}

func TestDockerSkip(t *testing.T) {
	t.Run("skip", func(t *testing.T) {
		require.True(t, DockerPipe{}.Skip(context.New(config.Project{})))
	})

	t.Run("skip sign", func(t *testing.T) {
		ctx := context.New(config.Project{})
		ctx.SkipSign = true
		require.True(t, DockerPipe{}.Skip(ctx))
	})

	t.Run("dont skip", func(t *testing.T) {
		ctx := context.New(config.Project{
			DockerSigns: []config.Sign{
				{},
			},
		})
		require.False(t, DockerPipe{}.Skip(ctx))
	})
}
