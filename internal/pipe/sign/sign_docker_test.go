package sign

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/gio"
	"github.com/goreleaser/goreleaser/internal/testlib"
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
	require.Equal(t, ctx.Config.DockerSigns[0].Args, []string{"sign", "--key=cosign.key", "$artifact"})
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
	testlib.CheckPath(t, "cosign")
	key := "cosign.key"
	cmd := "sh"
	args := []string{"-c", "echo ${artifact} > ${signature} && cosign sign --key=" + key + " --upload=false ${artifact} > ${signature}"}
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
					Args:      []string{"sign", "--key=" + key, "--upload=false", "${artifact}"},
				},
			},
		},
		"only certificate": {
			Expected: []string{
				"ghcrio-caarlos0-goreleaser-docker-manifest-actions-example-121-amd64.pem",
				"ghcrio-caarlos0-goreleaser-docker-manifest-actions-example-121-arm64v8.pem",
				"ghcrio-caarlos0-goreleaser-docker-manifest-actions-example-121.pem",
			},
			Signs: []config.Sign{
				{
					Artifacts:   "all",
					Stdin:       &password,
					Cmd:         "cosign",
					Certificate: `{{ replace (replace (replace .Env.artifact "/" "-") ":" "-") "." "" }}.pem`,
					Args:        []string{"sign", "--output-certificate=${certificate}", "--key=" + key, "--upload=false", "${artifact}"},
				},
			},
		},
		"sign all": {
			Expected: []string{
				"all_img1.sig",
				"all_img2.sig",
				"all_man1.sig",
			},
			Signs: []config.Sign{
				{
					Artifacts: "all",
					Stdin:     &password,
					Signature: `all_${artifactID}.sig`,
					Cmd:       cmd,
					Args:      args,
				},
			},
		},
		"sign all filtering id": {
			Expected: []string{"all_filter_by_id_img2.sig"},
			Signs: []config.Sign{
				{
					Artifacts: "all",
					IDs:       []string{"img2"},
					Stdin:     &password,
					Signature: "all_filter_by_id_${artifactID}.sig",
					Cmd:       cmd,
					Args:      args,
				},
			},
		},
		"sign images only": {
			Expected: []string{
				"images_img1.sig",
				"images_img2.sig",
			},
			Signs: []config.Sign{
				{
					Artifacts: "images",
					Stdin:     &password,
					Signature: "images_${artifactID}.sig",
					Cmd:       cmd,
					Args:      args,
				},
			},
		},
		"sign manifests only": {
			Expected: []string{"manifests_man1.sig"},
			Signs: []config.Sign{
				{
					Artifacts: "manifests",
					Stdin:     &password,
					Signature: "manifests_${artifactID}.sig",
					Cmd:       cmd,
					Args:      args,
				},
			},
		},
		// TODO: keyless test?
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.New(config.Project{})
			ctx.Config.DockerSigns = cfg.Signs
			wd, err := os.Getwd()
			require.NoError(t, err)
			tmp := testlib.Mktmp(t)
			require.NoError(t, gio.Copy(filepath.Join(wd, "testdata/cosign/"), tmp))
			ctx.Config.Dist = "dist"
			require.NoError(t, os.Mkdir("dist", 0o755))

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
			for _, sig := range ctx.Artifacts.Filter(
				artifact.Or(
					artifact.ByType(artifact.Signature),
					artifact.ByType(artifact.Certificate),
				),
			).List() {
				sigs = append(sigs, sig.Name)
				require.Truef(t, strings.HasPrefix(sig.Path, ctx.Config.Dist), "signature %q is not in dist dir %q", sig.Path, ctx.Config.Dist)
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
