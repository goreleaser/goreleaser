package docker

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/pipe"
	"github.com/goreleaser/goreleaser/v2/internal/skips"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"
)

const (
	registryPort    = "5050"
	registry        = "localhost:5050/"
	altRegistryPort = "5051"
	altRegistry     = "localhost:5051/"
)

func start(tb testing.TB) {
	tb.Helper()
	tb.Log("starting registries")
	var wg sync.WaitGroup
	wg.Go(func() { testlib.StartRegistry(tb, "alt_registry", altRegistryPort) })
	wg.Go(func() { testlib.StartRegistry(tb, "registry", registryPort) })
	wg.Wait()
}

func TestRunPipeNoBuild(t *testing.T) {
	table := map[string]struct {
		dockers      []config.Docker
		env          map[string]string
		assertError  errChecker
		extraPrepare func(t *testing.T, ctx *context.Context)
	}{
		"templated-dockerfile-invalid": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img13:v1",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "{{ .Env.Dockerfile }}",
				},
			},
			assertError: shouldTemplateErr,
		},
		"tag_template_error": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img28:{{.Tag}",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			assertError: shouldTemplateErr,
		},
		"build_flag_template_error": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img29:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					BuildFlagTemplates: []string{
						"--label=tag={{.Tag}",
					},
				},
			},
			assertError: shouldTemplateErr,
		},
		"missing_env_on_tag_template": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img30:{{.Env.NOPE}}",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			assertError: shouldTemplateErr,
		},
		"missing_env_on_build_flag_template": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img31:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					BuildFlagTemplates: []string{
						"--label=nope={{.Env.NOPE}}",
					},
				},
			},
			assertError: shouldTemplateErr,
		},
		"no image tags": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						"",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			assertError: shouldErr("no image templates found"),
		},
		"dockerfile_doesnt_exist": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{"img34:latest"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfilezzz",
				},
			},
			assertError: shouldErr(`failed to copy dockerfile`),
		},
		"extra_file_doesnt_exist": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{"img35:latest"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfile",
					Files: []string{
						"testdata/nope.txt",
					},
				},
			},
			assertError: shouldErr(`failed to copy extra file 'testdata/nope.txt'`),
		},
		"binary doesnt exist": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{"img36:latest"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfile",
					IDs:            []string{"nope"},
				},
			},
			assertError: shouldErr(`failed to copy wont-exist`),
			extraPrepare: func(t *testing.T, ctx *context.Context) {
				t.Helper()
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   "wont-exist",
					Path:   "wont-exist",
					Goarch: "amd64",
					Goos:   "linux",
					Type:   artifact.Binary,
					Extra: map[string]any{
						artifact.ExtraID: "nope",
					},
				})
			},
		},
	}

	t.Parallel()
	for name, docker := range table {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			folder := t.TempDir()
			dist := filepath.Join(folder, "dist")
			require.NoError(t, os.MkdirAll(filepath.Join(dist, "mybin", "subdir"), 0o755))
			f, err := os.Create(filepath.Join(dist, "mybin", "mybin"))
			require.NoError(t, err)
			require.NoError(t, f.Close())
			f, err = os.Create(filepath.Join(dist, "mybin", "anotherbin"))
			require.NoError(t, err)
			require.NoError(t, f.Close())
			f, err = os.Create(filepath.Join(dist, "mybin", "subdir", "subbin"))
			require.NoError(t, err)
			require.NoError(t, f.Close())
			f, err = os.Create(filepath.Join(dist, "mynfpm.apk"))
			require.NoError(t, err)
			require.NoError(t, f.Close())
			for _, arch := range []string{"amd64", "386", "arm64"} {
				f, err = os.Create(filepath.Join(dist, fmt.Sprintf("mybin_%s.apk", arch)))
				require.NoError(t, err)
				require.NoError(t, f.Close())
			}

			ctx := testctx.WrapWithCfg(t.Context(),
				config.Project{
					ProjectName: "mybin",
					Dist:        dist,
					Dockers:     slices.Clone(docker.dockers),
				},
				testctx.WithEnv(docker.env),
				testctx.WithVersion("1.0.0"),
				testctx.WithCurrentTag("v1.0.0"),
				testctx.WithCommit("a1b2c3d4"),
				testctx.WithSemver(1, 0, 0, ""))

			for _, os := range []string{"linux", "darwin"} {
				for _, arch := range []string{"amd64", "386", "arm64"} {
					for _, bin := range []string{"mybin", "anotherbin", "subdir/subbin"} {
						ctx.Artifacts.Add(&artifact.Artifact{
							Name:   bin,
							Path:   filepath.Join(dist, "mybin", bin),
							Goarch: arch,
							Goos:   os,
							Type:   artifact.Binary,
							Extra: map[string]any{
								artifact.ExtraID: bin,
							},
						})
					}
				}
			}
			for _, arch := range []string{"amd64", "386", "arm64"} {
				name := fmt.Sprintf("mybin_%s.apk", arch)
				ctx.Artifacts.Add(&artifact.Artifact{
					Name:   name,
					Path:   filepath.Join(dist, name),
					Goarch: arch,
					Goos:   "linux",
					Type:   artifact.LinuxPackage,
					Extra: map[string]any{
						artifact.ExtraID: "mybin",
					},
				})
			}

			if docker.extraPrepare != nil {
				docker.extraPrepare(t, ctx)
			}

			for i := range ctx.Config.Dockers {
				docker := &ctx.Config.Dockers[i]
				docker.Use = useDocker
				docker.PushFlags = []string{}
			}
			require.NoError(t, Pipe{}.Default(ctx))
			docker.assertError(t, Pipe{}.Run(ctx))
		})
	}
}

// TODO: this test is too big... split in smaller tests? Mainly the manifest ones...
func TestRunPipe(t *testing.T) {
	testlib.CheckDocker(t)
	testlib.SkipIfWindows(t, "registry images only available for windows")

	table := map[string]struct {
		dockers             []config.Docker
		manifests           []config.DockerManifest
		env                 map[string]string
		expect              []string
		assertImageLabels   imageLabelFinder
		assertError         errChecker
		pubAssertError      errChecker
		manifestAssertError errChecker
		extraPrepare        func(t *testing.T, ctx *context.Context)
	}{
		"multiarch": {
			dockers: []config.Docker{
				{
					ImageTemplates:     []string{registry + "goreleaser/img1:test-amd64"},
					Goos:               "linux",
					Goarch:             "amd64",
					Dockerfile:         "testdata/Dockerfile.arch",
					BuildFlagTemplates: []string{"--build-arg", "ARCH=amd64", "--platform", "linux/amd64"},
				},
				{
					ImageTemplates:     []string{registry + "goreleaser/img1:test-arm64v8"},
					Goos:               "linux",
					Goarch:             "arm64",
					Dockerfile:         "testdata/Dockerfile.arch",
					BuildFlagTemplates: []string{"--build-arg", "ARCH=arm64v8", "--platform", "linux/arm64"},
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate: registry + "goreleaser/img1:test",
					ImageTemplates: []string{
						registry + "goreleaser/img1:test-amd64",
						registry + "goreleaser/img1:test-arm64v8",
					},
				},
			},
			expect: []string{
				registry + "goreleaser/img1:test-amd64",
				registry + "goreleaser/img1:test-arm64v8",
			},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
			assertImageLabels:   noLabels,
		},
		"manifest autoskip no prerelease": {
			env: map[string]string{"AUTO": "auto"},
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/img2:test-amd64"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate: registry + "goreleaser/img2:test",
					ImageTemplates: []string{
						registry + "goreleaser/img2:test-amd64",
					},
					SkipPush: "{{ .Env.AUTO }}",
				},
			},
			expect: []string{
				registry + "goreleaser/img2:test-amd64",
			},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
			assertImageLabels:   noLabels,
		},
		"manifest autoskip prerelease": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/img3:test-amd64"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate: registry + "goreleaser/img3:test",
					ImageTemplates: []string{
						registry + "goreleaser/img3:test-amd64",
					},
					SkipPush: "auto",
				},
			},
			expect: []string{
				registry + "goreleaser/img3:test-amd64",
			},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: testlib.AssertSkipped,
			assertImageLabels:   noLabels,
			extraPrepare: func(t *testing.T, ctx *context.Context) {
				t.Helper()
				ctx.Semver.Prerelease = "beta"
			},
		},
		"manifest skip": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/img4:test-amd64"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate: registry + "goreleaser/img4:test",
					ImageTemplates: []string{
						registry + "goreleaser/img4:test-amd64",
					},
					SkipPush: "true",
				},
			},
			expect: []string{
				registry + "goreleaser/img4:test-amd64",
			},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: testlib.AssertSkipped,
			assertImageLabels:   noLabels,
		},
		"multiarch with previous existing manifest": {
			dockers: []config.Docker{
				{
					ImageTemplates:     []string{registry + "goreleaser/img4:2test-amd64"},
					Goos:               "linux",
					Goarch:             "amd64",
					Dockerfile:         "testdata/Dockerfile.arch",
					BuildFlagTemplates: []string{"--build-arg", "ARCH=amd64", "--platform", "linux/amd64"},
				},
				{
					ImageTemplates:     []string{registry + "goreleaser/img4:2test-arm64v8"},
					Goos:               "linux",
					Goarch:             "arm64",
					Dockerfile:         "testdata/Dockerfile.arch",
					BuildFlagTemplates: []string{"--build-arg", "ARCH=arm64v8", "--platform", "linux/arm64"},
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate: registry + "goreleaser/img4:2test",
					ImageTemplates: []string{
						registry + "goreleaser/img4:2test-amd64",
						registry + "goreleaser/img4:2test-arm64v8",
					},
				},
			},
			expect: []string{
				registry + "goreleaser/img4:2test-amd64",
				registry + "goreleaser/img4:2test-arm64v8",
			},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
			assertImageLabels:   noLabels,
			extraPrepare: func(t *testing.T, ctx *context.Context) {
				t.Helper()
				for _, cmd := range []string{
					fmt.Sprintf("docker manifest rm %sgoreleaser/img4:2test || true", registry),
					fmt.Sprintf("docker build --sbom=false --provenance=false -t %sgoreleaser/dummy:v1 --platform linux/amd64 -f testdata/Dockerfile.dummy .", registry),
					fmt.Sprintf("docker push %sgoreleaser/dummy:v1", registry),
					fmt.Sprintf("docker manifest create %sgoreleaser/img4:2test --amend %sgoreleaser/dummy:v1 --insecure", registry, registry),
				} {
					parts := strings.Fields(strings.TrimSuffix(cmd, " || true"))
					out, err := exec.CommandContext(ctx, parts[0], parts[1:]...).CombinedOutput()
					if !strings.HasSuffix(cmd, " || true") {
						require.NoError(t, err, cmd+": "+string(out))
					}
				}
			},
		},
		"multiarch image not found": {
			dockers: []config.Docker{
				{
					ImageTemplates:     []string{registry + "goreleaser/img5:latest-arm64v8"},
					Goos:               "linux",
					Goarch:             "arm64",
					Dockerfile:         "testdata/Dockerfile.arch",
					BuildFlagTemplates: []string{"--build-arg", "ARCH=arm64v8", "--platform", "linux/arm64"},
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate:   registry + "goreleaser/img5:test",
					ImageTemplates: []string{registry + "goreleaser/img5:latest-amd64"},
				},
			},
			expect:              []string{registry + "goreleaser/img5:latest-arm64v8"},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldErr("failed to create localhost:5050/goreleaser/img5:test"),
			assertImageLabels:   noLabels,
		},
		"multiarch manifest template error": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/img6"},
					Goos:           "linux",
					Goarch:         "arm64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate:   registry + "goreleaser/img6:{{ .Goos }",
					ImageTemplates: []string{registry + "goreleaser/img6"},
				},
			},
			expect:              []string{registry + "goreleaser/img6"},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldTemplateErr,
			assertImageLabels:   noLabels,
		},
		"multiarch image template error": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/img7"},
					Goos:           "linux",
					Goarch:         "arm64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate:   registry + "goreleaser/img7",
					ImageTemplates: []string{registry + "goreleaser/img7:{{ .Goos }"},
				},
			},
			expect:              []string{registry + "goreleaser/img7"},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldTemplateErr,
			assertImageLabels:   noLabels,
		},
		"multiarch missing manifest name": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/img8"},
					Goos:           "linux",
					Goarch:         "arm64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate:   "  ",
					ImageTemplates: []string{registry + "goreleaser/img8"},
				},
			},
			expect:              []string{registry + "goreleaser/img8"},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: testlib.AssertSkipped,
			assertImageLabels:   noLabels,
		},
		"multiarch missing images": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/img9"},
					Dockerfile:     "testdata/Dockerfile",
					Goos:           "linux",
					Goarch:         "arm64",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate:   "ignored",
					ImageTemplates: []string{" ", "   ", ""},
				},
			},
			expect:              []string{registry + "goreleaser/img9"},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: testlib.AssertSkipped,
			assertImageLabels:   noLabels,
		},
		"valid": {
			env: map[string]string{
				"FOO": "123",
			},
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img10:{{.Tag}}-{{.Env.FOO}}",
						registry + "goreleaser/img10:v{{.Major}}",
						registry + "goreleaser/img10:v{{.Major}}.{{.Minor}}",
						registry + "goreleaser/img10:commit-{{.Commit}}",
						registry + "goreleaser/img10:latest",
						altRegistry + "goreleaser/img10:{{.Tag}}-{{.Env.FOO}}",
						altRegistry + "goreleaser/img10:v{{.Major}}",
						altRegistry + "goreleaser/img10:v{{.Major}}.{{.Minor}}",
						altRegistry + "goreleaser/img10:commit-{{.Commit}}",
						altRegistry + "goreleaser/img10:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					BuildFlagTemplates: []string{
						"--label=org.label-schema.schema-version=1.0",
						"--label=org.label-schema.version={{.Version}}",
						"--label=org.label-schema.vcs-ref={{.Commit}}",
						"--label=org.label-schema.name={{.ProjectName}}",
						"--build-arg=FRED={{.Tag}}",
					},
					Files: []string{
						"testdata/extra_file.txt",
					},
				},
			},
			expect: []string{
				registry + "goreleaser/img10:v1.0.0-123",
				registry + "goreleaser/img10:v1",
				registry + "goreleaser/img10:v1.0",
				registry + "goreleaser/img10:commit-a1b2c3d4",
				registry + "goreleaser/img10:latest",
				altRegistry + "goreleaser/img10:v1.0.0-123",
				altRegistry + "goreleaser/img10:v1",
				altRegistry + "goreleaser/img10:v1.0",
				altRegistry + "goreleaser/img10:commit-a1b2c3d4",
				altRegistry + "goreleaser/img10:latest",
			},
			assertImageLabels: shouldFindImagesWithLabels(
				"goreleaser/img10",
				"label=org.label-schema.schema-version=1.0",
				"label=org.label-schema.version=1.0.0",
				"label=org.label-schema.vcs-ref=a1b2c3d4",
				"label=org.label-schema.name=mybin",
			),
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"templated-dockerfile": {
			env: map[string]string{
				"Dockerfile": "testdata/Dockerfile",
			},
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img11:v1",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "{{ .Env.Dockerfile }}",
					Files: []string{
						"testdata/extra_file.txt",
					},
				},
			},
			expect: []string{
				registry + "goreleaser/img11:v1",
			},
			assertError:         shouldNotErr,
			assertImageLabels:   noLabels,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"wrong binary name": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img12:v1",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile.wrongbin",
				},
			},
			assertError:       shouldErr("seems like you tried to copy a file that is not available in the build context"),
			assertImageLabels: noLabels,
		},
		"image template with env": {
			env: map[string]string{
				"FOO": "img14",
			},
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/{{.Env.FOO}}:{{.Tag}}",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			expect: []string{
				registry + "goreleaser/img14:v1.0.0",
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"image template uppercase": {
			env: map[string]string{
				"FOO": "img15_UPPERCASE",
			},
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/{{.Env.FOO}}:{{.Tag}}",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			expect:              []string{},
			assertImageLabels:   noLabels,
			assertError:         shouldErr(`failed to build localhost:5050/goreleaser/img15_UPPERCASE:v1.0.0`),
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"empty image tag": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						"",
						registry + "goreleaser/img16:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			expect: []string{
				registry + "goreleaser/img16:latest",
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"valid with ids": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img17:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					IDs:        []string{"mybin"},
				},
			},
			expect: []string{
				registry + "goreleaser/img17:latest",
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"multiple images with same extra file": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img18:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					Files:      []string{"testdata/extra_file.txt"},
				},
				{
					ImageTemplates: []string{
						registry + "goreleaser/img19:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					Files:      []string{"testdata/extra_file.txt"},
				},
			},
			expect: []string{
				registry + "goreleaser/img18:latest",
				registry + "goreleaser/img19:latest",
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"multiple images with same dockerfile": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img20:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
				{
					ImageTemplates: []string{
						registry + "goreleaser/img21:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			assertImageLabels: noLabels,
			expect: []string{
				registry + "goreleaser/img20:latest",
				registry + "goreleaser/img21:latest",
			},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"valid_skip_push": {
			env: map[string]string{"TRUE": "true"},
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img22:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					SkipPush:   "{{.Env.TRUE}}",
				},
			},
			expect: []string{
				registry + "goreleaser/img22:latest",
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      testlib.AssertSkipped,
			manifestAssertError: shouldNotErr,
		},
		"one_img_error_with_skip_push": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img23:true",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile.true",
					SkipPush:   "true",
				},
				{
					ImageTemplates: []string{
						registry + "goreleaser/img23:false",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile.false",
					SkipPush:   "true",
				},
			},
			expect: []string{
				registry + "goreleaser/img23:true",
			},
			assertImageLabels: noLabels,
			assertError:       shouldErr("failed to build localhost:5050/goreleaser/img23:false"),
		},
		"valid_no_latest": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img24:{{.Version}}",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			expect: []string{
				registry + "goreleaser/img24:1.0.0",
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"valid build args": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img25:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					BuildFlagTemplates: []string{
						"--label=foo=bar",
					},
				},
			},
			expect: []string{
				registry + "goreleaser/img25:latest",
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"bad build args": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img26:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					BuildFlagTemplates: []string{
						"--bad-flag",
					},
				},
			},
			assertImageLabels: noLabels,
			assertError:       shouldErr("failed to build localhost:5050/goreleaser/img26:latest"),
		},
		"bad_dockerfile": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img27:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile.bad",
				},
			},
			assertImageLabels: noLabels,
			assertError:       shouldErr("failed to build localhost:5050/goreleaser/img27:latest"),
		},
		"image_has_projectname_template_variable": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/img32_{{.ProjectName}}:{{.Tag}}-{{.Env.FOO}}",
						registry + "goreleaser/img32_{{.ProjectName}}:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					SkipPush:   "true",
				},
			},
			env: map[string]string{
				"FOO": "123",
			},
			expect: []string{
				registry + "goreleaser/img32_mybin:v1.0.0-123",
				registry + "goreleaser/img32_mybin:latest",
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      testlib.AssertSkipped,
			manifestAssertError: shouldNotErr,
		},
		"no_permissions": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{"localhost:1/img33:latest"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			expect: []string{
				"docker.io/nope:latest",
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      shouldErr(`failed to push localhost:1/img33:latest`),
			manifestAssertError: shouldNotErr,
		},
		"multiple_ids": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/img37:latest"},
					Goos:           "darwin",
					Goarch:         "amd64",
					IDs:            []string{"mybin", "anotherbin", "subdir/subbin"},
					Dockerfile:     "testdata/Dockerfile.multiple",
				},
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
			expect: []string{
				registry + "goreleaser/img37:latest",
			},
		},
		"nfpm and multiple binaries": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/img38:latest"},
					Goos:           "linux",
					Goarch:         "amd64",
					IDs:            []string{"mybin", "anotherbin"},
					Dockerfile:     "testdata/Dockerfile.nfpm",
				},
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
			expect: []string{
				registry + "goreleaser/img38:latest",
			},
		},
		"nfpm and multiple binaries on arm64": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/img39:latest"},
					Goos:           "linux",
					Goarch:         "arm64",
					IDs:            []string{"mybin", "anotherbin"},
					Dockerfile:     "testdata/Dockerfile.nfpm",
				},
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
			expect: []string{
				registry + "goreleaser/img39:latest",
			},
		},
	}

	start(t)

	for imager := range imagers {
		t.Run(imager, func(t *testing.T) {
			for name, docker := range table {
				t.Run(name, func(t *testing.T) {
					folder := t.TempDir()
					dist := filepath.Join(folder, "dist")
					require.NoError(t, os.MkdirAll(filepath.Join(dist, "mybin", "subdir"), 0o755))
					f, err := os.Create(filepath.Join(dist, "mybin", "mybin"))
					require.NoError(t, err)
					require.NoError(t, f.Close())
					f, err = os.Create(filepath.Join(dist, "mybin", "anotherbin"))
					require.NoError(t, err)
					require.NoError(t, f.Close())
					f, err = os.Create(filepath.Join(dist, "mybin", "subdir", "subbin"))
					require.NoError(t, err)
					require.NoError(t, f.Close())
					f, err = os.Create(filepath.Join(dist, "mynfpm.apk"))
					require.NoError(t, err)
					require.NoError(t, f.Close())
					for _, arch := range []string{"amd64", "386", "arm64"} {
						f, err = os.Create(filepath.Join(dist, fmt.Sprintf("mybin_%s.apk", arch)))
						require.NoError(t, err)
						require.NoError(t, f.Close())
					}

					ctx := testctx.WrapWithCfg(t.Context(),
						config.Project{
							ProjectName:     "mybin",
							Dist:            dist,
							Dockers:         slices.Clone(docker.dockers),
							DockerManifests: slices.Clone(docker.manifests),
						},
						testctx.WithEnv(docker.env),
						testctx.WithVersion("1.0.0"),
						testctx.WithCurrentTag("v1.0.0"),
						testctx.WithCommit("a1b2c3d4"),
						testctx.WithSemver(1, 0, 0, ""))

					for _, os := range []string{"linux", "darwin"} {
						for _, arch := range []string{"amd64", "386", "arm64"} {
							for _, bin := range []string{"mybin", "anotherbin", "subdir/subbin"} {
								ctx.Artifacts.Add(&artifact.Artifact{
									Name:   bin,
									Path:   filepath.Join(dist, "mybin", bin),
									Goarch: arch,
									Goos:   os,
									Type:   artifact.Binary,
									Extra: map[string]any{
										artifact.ExtraID: bin,
									},
								})
							}
						}
					}
					for _, arch := range []string{"amd64", "386", "arm64"} {
						name := fmt.Sprintf("mybin_%s.apk", arch)
						ctx.Artifacts.Add(&artifact.Artifact{
							Name:   name,
							Path:   filepath.Join(dist, name),
							Goarch: arch,
							Goos:   "linux",
							Type:   artifact.LinuxPackage,
							Extra: map[string]any{
								artifact.ExtraID: "mybin",
							},
						})
					}

					if docker.extraPrepare != nil {
						docker.extraPrepare(t, ctx)
					}

					// this might fail as the image doesnt exist yet, so lets ignore the error
					_ = rmis(t, docker.expect)

					for i := range ctx.Config.Dockers {
						docker := &ctx.Config.Dockers[i]
						docker.Use = imager
						docker.PushFlags = []string{}
					}
					for i := range ctx.Config.DockerManifests {
						manifest := &ctx.Config.DockerManifests[i]
						manifest.Use = useDocker
						manifest.PushFlags = []string{"--insecure"}
						manifest.CreateFlags = []string{"--insecure"}
					}
					require.NoError(t, Pipe{}.Default(ctx))
					require.NoError(t, ManifestPipe{}.Default(ctx))
					err = Pipe{}.Run(ctx)
					docker.assertError(t, err)
					if err == nil {
						docker.pubAssertError(t, Pipe{}.Publish(ctx))
						docker.manifestAssertError(t, ManifestPipe{}.Publish(ctx))
					}

					for _, d := range docker.dockers {
						docker.assertImageLabels(t, d.Use)
					}

					// this might should not fail as the image should have been created when
					// the step ran
					require.NoError(t, rmis(t, docker.expect))

					_ = ctx.Artifacts.Filter(
						artifact.Or(
							artifact.ByType(artifact.DockerImage),
							artifact.ByType(artifact.DockerManifest),
						),
					).Visit(func(a *artifact.Artifact) error {
						digest := artifact.MustExtra[string](*a, artifact.ExtraDigest)
						require.NotEmpty(t, digest, "missing digest for "+a.Name)
						return nil
					})
				})
			}
		})
	}
}

func TestBuildCommand(t *testing.T) {
	images := []string{"goreleaser/test_build_flag", "goreleaser/test_multiple_tags"}
	tests := []struct {
		name   string
		flags  []string
		buildx bool
		expect []string
	}{
		{
			name:   "no flags",
			flags:  []string{},
			expect: []string{"build", ".", "-t", images[0], "-t", images[1], "--provenance=false", "--sbom=false"},
		},
		{
			name:   "single flag",
			flags:  []string{"--label=foo"},
			expect: []string{"build", ".", "-t", images[0], "-t", images[1], "--label=foo", "--provenance=false", "--sbom=false"},
		},
		{
			name:   "multiple flags",
			flags:  []string{"--label=foo", "--build-arg=bar=baz"},
			expect: []string{"build", ".", "-t", images[0], "-t", images[1], "--label=foo", "--build-arg=bar=baz", "--provenance=false", "--sbom=false"},
		},
		{
			name:   "buildx",
			buildx: true,
			flags:  []string{"--label=foo", "--build-arg=bar=baz"},
			expect: []string{"buildx", "build", ".", "--load", "-t", images[0], "-t", images[1], "--label=foo", "--build-arg=bar=baz", "--provenance=false", "--sbom=false"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imager := dockerImager{
				buildx: tt.buildx,
			}
			require.Equal(t, tt.expect, imager.buildCommand(images, tt.flags))
		})
	}
}

func TestDescription(t *testing.T) {
	require.NotEmpty(t, Pipe{}.String())
}

func TestNoDockerWithoutImageName(t *testing.T) {
	testlib.AssertSkipped(t, Pipe{}.Run(testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{
			{
				Goos: "linux",
			},
		},
	})))
}

func TestDefault(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{
			{
				IDs: []string{"aa"},
			},
			{
				Use: useBuildx,
			},
		},
		DockerManifests: []config.DockerManifest{
			{},
			{
				Use: useDocker,
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Dockers, 2)
	docker := ctx.Config.Dockers[0]
	require.Equal(t, "linux", docker.Goos)
	require.Equal(t, "amd64", docker.Goarch)
	require.Equal(t, "6", docker.Goarm)
	require.Equal(t, []string{"aa"}, docker.IDs)
	require.Equal(t, useDocker, docker.Use)
	docker = ctx.Config.Dockers[1]
	require.Equal(t, useBuildx, docker.Use)
	require.Equal(t, uint(10), docker.Retry.Attempts)
	require.Equal(t, 10*time.Second, docker.Retry.Delay)
	require.Equal(t, 5*time.Minute, docker.Retry.MaxDelay)

	require.NoError(t, ManifestPipe{}.Default(ctx))
	require.Len(t, ctx.Config.DockerManifests, 2)
	require.Equal(t, useDocker, ctx.Config.DockerManifests[0].Use)
	require.Equal(t, useDocker, ctx.Config.DockerManifests[1].Use)

	for _, manifest := range ctx.Config.DockerManifests {
		require.Equal(t, uint(10), manifest.Retry.Attempts)
		require.Equal(t, 10*time.Second, manifest.Retry.Delay)
		require.Equal(t, 5*time.Minute, manifest.Retry.MaxDelay)
	}
}

func TestDefaultDuplicateID(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{
			{ID: "foo"},
			{},
			{ID: "bar"},
			{ID: "foo"},
		},
		DockerManifests: []config.DockerManifest{
			{ID: "bar"},
			{},
			{ID: "bar"},
			{ID: "foo"},
		},
	})

	require.EqualError(t, Pipe{}.Default(ctx), "found 2 dockers with the ID 'foo', please fix your config")
	require.EqualError(t, ManifestPipe{}.Default(ctx), "found 2 docker_manifests with the ID 'bar', please fix your config")
}

func TestDefaultInvalidUse(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{
			{
				Use: "something",
			},
		},
		DockerManifests: []config.DockerManifest{
			{
				Use: "something",
			},
		},
	})

	err := Pipe{}.Default(ctx)
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), `docker: invalid use: something, valid options are`))

	err = ManifestPipe{}.Default(ctx)
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), `docker manifest: invalid use: something, valid options are`))
}

func TestDefaultDockerfile(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Builds: []config.Build{
			{},
		},
		Dockers: []config.Docker{
			{},
			{},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Dockers, 2)
	require.Equal(t, "Dockerfile", ctx.Config.Dockers[0].Dockerfile)
	require.Equal(t, "Dockerfile", ctx.Config.Dockers[1].Dockerfile)
}

func TestDraftRelease(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Release: config.Release{
			Draft: true,
		},
	})

	require.False(t, pipe.IsSkip(Pipe{}.Publish(ctx)))
}

func TestDefaultNoDockers(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Empty(t, ctx.Config.Dockers)
}

func TestDefaultFilesDot(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: "/tmp/distt",
		Dockers: []config.Docker{
			{
				Files: []string{"./lala", "./lolsob", "."},
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
}

func TestDefaultFilesDis(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dist: "/tmp/dist",
		Dockers: []config.Docker{
			{
				Files: []string{"./fooo", "/tmp/dist/asdasd/asd", "./bar"},
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
}

func TestDefaultSet(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{
			{
				IDs:        []string{"foo"},
				Goos:       "windows",
				Goarch:     "i386",
				Dockerfile: "Dockerfile.foo",
			},
		},
	})

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Dockers, 1)
	docker := ctx.Config.Dockers[0]
	require.Equal(t, "windows", docker.Goos)
	require.Equal(t, "i386", docker.Goarch)
	require.Equal(t, []string{"foo"}, docker.IDs)
	require.Equal(t, "Dockerfile.foo", docker.Dockerfile)
}

func Test_processImageTemplates(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(),
		config.Project{
			Builds: []config.Build{
				{
					ID: "default",
				},
			},
			Dockers: []config.Docker{
				{
					Dockerfile: "Dockerfile.foo",
					ImageTemplates: []string{
						"user/image:{{.Tag}}",
						"gcr.io/image:{{.Tag}}-{{.Env.FOO}}",
						"gcr.io/image:v{{.Major}}.{{.Minor}}",
					},
					SkipPush: "true",
				},
			},
			Env: []string{"FOO=123"},
		},
		testctx.WithVersion("1.0.0"),
		testctx.WithCurrentTag("v1.0.0"),
		testctx.WithCommit("a1b2c3d4"),
		testctx.WithSemver(1, 0, 0, ""))

	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Dockers, 1)

	docker := ctx.Config.Dockers[0]
	require.Equal(t, "Dockerfile.foo", docker.Dockerfile)

	images, err := processImageTemplates(ctx, docker)
	require.NoError(t, err)
	require.Equal(t, []string{
		"user/image:v1.0.0",
		"gcr.io/image:v1.0.0-123",
		"gcr.io/image:v1.0",
	}, images)
}

func TestSkip(t *testing.T) {
	t.Run("image", func(t *testing.T) {
		t.Run("skip", func(t *testing.T) {
			require.True(t, Pipe{}.Skip(testctx.Wrap(t.Context())))
		})

		t.Run("skip docker", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Dockers: []config.Docker{{}},
			}, testctx.Skip(skips.Docker))

			require.True(t, Pipe{}.Skip(ctx))
		})

		t.Run("dont skip", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Dockers: []config.Docker{{}},
			})

			require.False(t, Pipe{}.Skip(ctx))
		})
	})

	t.Run("manifest", func(t *testing.T) {
		t.Run("skip", func(t *testing.T) {
			require.True(t, ManifestPipe{}.Skip(testctx.Wrap(t.Context())))
		})

		t.Run("skip docker", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				DockerManifests: []config.DockerManifest{{}},
			}, testctx.Skip(skips.Docker))

			require.True(t, ManifestPipe{}.Skip(ctx))
		})

		t.Run("dont skip", func(t *testing.T) {
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				DockerManifests: []config.DockerManifest{{}},
			})

			require.False(t, ManifestPipe{}.Skip(ctx))
		})
	})
}

func TestWithDigest(t *testing.T) {
	artifacts := artifact.New()
	artifacts.Add(&artifact.Artifact{
		Name: "localhost:5050/owner/img:t1",
		Type: artifact.DockerImage,
		Extra: artifact.Extras{
			artifact.ExtraDigest: "sha256:d1",
		},
	})
	artifacts.Add(&artifact.Artifact{
		Name: "localhost:5050/owner/img:t2",
		Type: artifact.DockerImage,
		Extra: artifact.Extras{
			artifact.ExtraDigest: "sha256:d2",
		},
	})
	artifacts.Add(&artifact.Artifact{
		Name: "localhost:5050/owner/img:t3",
		Type: artifact.DockerImage,
	})

	for _, use := range []string{useDocker, useBuildx} {
		t.Run(use, func(t *testing.T) {
			t.Run("good", func(t *testing.T) {
				require.Equal(t, "localhost:5050/owner/img:t1@sha256:d1", withDigest("localhost:5050/owner/img:t1", artifacts.List()))
			})

			t.Run("no digest", func(t *testing.T) {
				require.Equal(t, "localhost:5050/owner/img:t3", withDigest("localhost:5050/owner/img:t3", artifacts.List()))
			})

			t.Run("no match", func(t *testing.T) {
				require.Equal(t, "localhost:5050/owner/img:t4", withDigest("localhost:5050/owner/img:t4", artifacts.List()))
			})
		})
	}
}

func TestDependencies(t *testing.T) {
	ctx := testctx.WrapWithCfg(t.Context(), config.Project{
		Dockers: []config.Docker{
			{Use: useBuildx},
			{Use: useDocker},
			{Use: "nope"},
		},
		DockerManifests: []config.DockerManifest{
			{Use: useBuildx},
			{Use: useDocker},
			{Use: "nope"},
		},
	})

	require.Equal(t, []string{"docker", "docker"}, Pipe{}.Dependencies(ctx))
	require.Equal(t, []string{"docker", "docker"}, ManifestPipe{}.Dependencies(ctx))
}

func TestIsFileNotFoundError(t *testing.T) {
	t.Run("executable not in path", func(t *testing.T) {
		require.False(t, isFileNotFoundError(`error getting credentials - err: exec: "docker-credential-desktop": executable file not found in $PATH, out:`))
	})

	t.Run("file not found", func(t *testing.T) {
		require.True(t, isFileNotFoundError(`./foo: file not found`))
		require.True(t, isFileNotFoundError(`./foo: not found: not found`))
	})
}

func TestValidateImager(t *testing.T) {
	tests := []struct {
		use       string
		wantError string
	}{
		{use: "docker"},
		{use: "buildx"},
		{use: "notFound", wantError: "docker: invalid use: notFound, valid options are [buildx docker]"},
	}

	for _, tt := range tests {
		t.Run(tt.use, func(t *testing.T) {
			err := validateImager(tt.use)
			if tt.wantError != "" {
				require.EqualError(t, err, tt.wantError)
				return
			}
			require.NoError(t, err)
		})
	}
}

func shouldNotErr(t *testing.T, err error) {
	t.Helper()
	require.NoError(t, err)
}

func shouldTemplateErr(t *testing.T, err error) {
	t.Helper()
	testlib.RequireTemplateError(t, err)
}

type imageLabelFinder func(*testing.T, string)

func shouldFindImagesWithLabels(image string, filters ...string) func(*testing.T, string) {
	return func(t *testing.T, _ string) {
		t.Helper()
		for _, filter := range filters {
			cmd := exec.CommandContext(t.Context(), "docker", "images", "-q", "--filter", "reference=*/"+image, "--filter", filter)
			// t.Log("running", cmd)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, string(output))
			uniqueIDs := map[string]string{}
			for id := range strings.SplitSeq(strings.TrimSpace(string(output)), "\n") {
				uniqueIDs[id] = id
			}
			require.Len(t, uniqueIDs, 1)
		}
	}
}

func noLabels(t *testing.T, _ string) {
	t.Helper()
}

type errChecker func(*testing.T, error)

func shouldErr(msg string) errChecker {
	return func(t *testing.T, err error) {
		t.Helper()
		if ge, ok := errors.AsType[gerrors.ErrDetailed](err); ok {
			for _, s := range ge.Messages() {
				if strings.Contains(s, msg) {
					return
				}
			}
			for _, a := range ge.Details() {
				s, ok := a.(string)
				if !ok {
					continue
				}
				if strings.Contains(s, msg) {
					return
				}
			}
		} else {
			require.ErrorContains(t, err, msg)
		}
	}
}

func rmis(tb testing.TB, imgs []string) error {
	tb.Helper()
	var wg errgroup.Group
	for _, img := range imgs {
		wg.Go(func() error {
			tb.Log("removing docker image", img)
			return exec.CommandContext(tb.Context(), "docker", "rmi", "--force", img).Run()
		})
	}
	return wg.Wait()
}
