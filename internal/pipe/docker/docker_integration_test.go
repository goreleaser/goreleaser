//go:build integration

package docker

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/gerrors"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/stretchr/testify/require"
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
	testlib.StartRegistry(tb, "registry", registryPort)
	testlib.StartRegistry(tb, "alt_registry", altRegistryPort)
}

// TODO: this test is too big... split in smaller tests? Mainly the manifest ones...
func TestIntegrationRunPipe(t *testing.T) {
	testlib.CheckDocker(t)
	testlib.SkipIfWindows(t, "registry images only available for windows")
	type errChecker func(*testing.T, error)
	shouldErr := func(msg string) errChecker {
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
	shouldNotErr := func(t *testing.T, err error) {
		t.Helper()
		require.NoError(t, err)
	}
	shouldTemplateErr := func(t *testing.T, err error) {
		t.Helper()
		testlib.RequireTemplateError(t, err)
	}
	type imageLabelFinder func(*testing.T, string)
	shouldFindImagesWithLabels := func(image string, filters ...string) func(*testing.T, string) {
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
	noLabels := func(t *testing.T, _ string) {
		t.Helper()
	}

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
					ImageTemplates:     []string{registry + "goreleaser/test_multiarch:test-amd64"},
					Goos:               "linux",
					Goarch:             "amd64",
					Dockerfile:         "testdata/Dockerfile.arch",
					BuildFlagTemplates: []string{"--build-arg", "ARCH=amd64", "--platform", "linux/amd64"},
				},
				{
					ImageTemplates:     []string{registry + "goreleaser/test_multiarch:test-arm64v8"},
					Goos:               "linux",
					Goarch:             "arm64",
					Dockerfile:         "testdata/Dockerfile.arch",
					BuildFlagTemplates: []string{"--build-arg", "ARCH=arm64v8", "--platform", "linux/arm64"},
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate: registry + "goreleaser/test_multiarch:test",
					ImageTemplates: []string{
						registry + "goreleaser/test_multiarch:test-amd64",
						registry + "goreleaser/test_multiarch:test-arm64v8",
					},
				},
			},
			expect: []string{
				registry + "goreleaser/test_multiarch:test-amd64",
				registry + "goreleaser/test_multiarch:test-arm64v8",
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
					ImageTemplates: []string{registry + "goreleaser/test_manifestskip:test-amd64"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate: registry + "goreleaser/test_manifestskip:test",
					ImageTemplates: []string{
						registry + "goreleaser/test_manifestskip:test-amd64",
					},
					SkipPush: "{{ .Env.AUTO }}",
				},
			},
			expect: []string{
				registry + "goreleaser/test_manifestskip:test-amd64",
			},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
			assertImageLabels:   noLabels,
		},
		"manifest autoskip prerelease": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/test_manifestskip-prerelease:test-amd64"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate: registry + "goreleaser/test_manifestskip-prerelease:test",
					ImageTemplates: []string{
						registry + "goreleaser/test_manifestskip-prerelease:test-amd64",
					},
					SkipPush: "auto",
				},
			},
			expect: []string{
				registry + "goreleaser/test_manifestskip-prerelease:test-amd64",
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
					ImageTemplates: []string{registry + "goreleaser/test_manifestskip-true:test-amd64"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate: registry + "goreleaser/test_manifestskip-true:test",
					ImageTemplates: []string{
						registry + "goreleaser/test_manifestskip-true:test-amd64",
					},
					SkipPush: "true",
				},
			},
			expect: []string{
				registry + "goreleaser/test_manifestskip-true:test-amd64",
			},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: testlib.AssertSkipped,
			assertImageLabels:   noLabels,
		},
		"multiarch with previous existing manifest": {
			dockers: []config.Docker{
				{
					ImageTemplates:     []string{registry + "goreleaser/test_multiarch:2test-amd64"},
					Goos:               "linux",
					Goarch:             "amd64",
					Dockerfile:         "testdata/Dockerfile.arch",
					BuildFlagTemplates: []string{"--build-arg", "ARCH=amd64", "--platform", "linux/amd64"},
				},
				{
					ImageTemplates:     []string{registry + "goreleaser/test_multiarch:2test-arm64v8"},
					Goos:               "linux",
					Goarch:             "arm64",
					Dockerfile:         "testdata/Dockerfile.arch",
					BuildFlagTemplates: []string{"--build-arg", "ARCH=arm64v8", "--platform", "linux/arm64"},
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate: registry + "goreleaser/test_multiarch:2test",
					ImageTemplates: []string{
						registry + "goreleaser/test_multiarch:2test-amd64",
						registry + "goreleaser/test_multiarch:2test-arm64v8",
					},
				},
			},
			expect: []string{
				registry + "goreleaser/test_multiarch:2test-amd64",
				registry + "goreleaser/test_multiarch:2test-arm64v8",
			},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
			assertImageLabels:   noLabels,
			extraPrepare: func(t *testing.T, ctx *context.Context) {
				t.Helper()
				for _, cmd := range []string{
					fmt.Sprintf("docker manifest rm %sgoreleaser/test_multiarch:2test || true", registry),
					fmt.Sprintf("docker build --sbom=false --provenance=false -t %sgoreleaser/dummy:v1 --platform linux/amd64 -f testdata/Dockerfile.dummy .", registry),
					fmt.Sprintf("docker push %sgoreleaser/dummy:v1", registry),
					fmt.Sprintf("docker manifest create %sgoreleaser/test_multiarch:2test --amend %sgoreleaser/dummy:v1 --insecure", registry, registry),
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
					ImageTemplates:     []string{registry + "goreleaser/test_multiarch_fail:latest-arm64v8"},
					Goos:               "linux",
					Goarch:             "arm64",
					Dockerfile:         "testdata/Dockerfile.arch",
					BuildFlagTemplates: []string{"--build-arg", "ARCH=arm64v8", "--platform", "linux/arm64"},
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate:   registry + "goreleaser/test_multiarch_fail:test",
					ImageTemplates: []string{registry + "goreleaser/test_multiarch_fail:latest-amd64"},
				},
			},
			expect:              []string{registry + "goreleaser/test_multiarch_fail:latest-arm64v8"},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldErr("failed to create localhost:5050/goreleaser/test_multiarch_fail:test"),
			assertImageLabels:   noLabels,
		},
		"multiarch manifest template error": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/test_multiarch_manifest_tmpl_error"},
					Goos:           "linux",
					Goarch:         "arm64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate:   registry + "goreleaser/test_multiarch_manifest_tmpl_error:{{ .Goos }",
					ImageTemplates: []string{registry + "goreleaser/test_multiarch_manifest_tmpl_error"},
				},
			},
			expect:              []string{registry + "goreleaser/test_multiarch_manifest_tmpl_error"},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldTemplateErr,
			assertImageLabels:   noLabels,
		},
		"multiarch image template error": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/test_multiarch_img_tmpl_error"},
					Goos:           "linux",
					Goarch:         "arm64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate:   registry + "goreleaser/test_multiarch_img_tmpl_error",
					ImageTemplates: []string{registry + "goreleaser/test_multiarch_img_tmpl_error:{{ .Goos }"},
				},
			},
			expect:              []string{registry + "goreleaser/test_multiarch_img_tmpl_error"},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldTemplateErr,
			assertImageLabels:   noLabels,
		},
		"multiarch missing manifest name": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/test_multiarch_no_manifest_name"},
					Goos:           "linux",
					Goarch:         "arm64",
					Dockerfile:     "testdata/Dockerfile",
				},
			},
			manifests: []config.DockerManifest{
				{
					NameTemplate:   "  ",
					ImageTemplates: []string{registry + "goreleaser/test_multiarch_no_manifest_name"},
				},
			},
			expect:              []string{registry + "goreleaser/test_multiarch_no_manifest_name"},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: testlib.AssertSkipped,
			assertImageLabels:   noLabels,
		},
		"multiarch missing images": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/test_multiarch_no_manifest_images"},
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
			expect:              []string{registry + "goreleaser/test_multiarch_no_manifest_images"},
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
						registry + "goreleaser/test_run_pipe:{{.Tag}}-{{.Env.FOO}}",
						registry + "goreleaser/test_run_pipe:v{{.Major}}",
						registry + "goreleaser/test_run_pipe:v{{.Major}}.{{.Minor}}",
						registry + "goreleaser/test_run_pipe:commit-{{.Commit}}",
						registry + "goreleaser/test_run_pipe:latest",
						altRegistry + "goreleaser/test_run_pipe:{{.Tag}}-{{.Env.FOO}}",
						altRegistry + "goreleaser/test_run_pipe:v{{.Major}}",
						altRegistry + "goreleaser/test_run_pipe:v{{.Major}}.{{.Minor}}",
						altRegistry + "goreleaser/test_run_pipe:commit-{{.Commit}}",
						altRegistry + "goreleaser/test_run_pipe:latest",
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
				registry + "goreleaser/test_run_pipe:v1.0.0-123",
				registry + "goreleaser/test_run_pipe:v1",
				registry + "goreleaser/test_run_pipe:v1.0",
				registry + "goreleaser/test_run_pipe:commit-a1b2c3d4",
				registry + "goreleaser/test_run_pipe:latest",
				altRegistry + "goreleaser/test_run_pipe:v1.0.0-123",
				altRegistry + "goreleaser/test_run_pipe:v1",
				altRegistry + "goreleaser/test_run_pipe:v1.0",
				altRegistry + "goreleaser/test_run_pipe:commit-a1b2c3d4",
				altRegistry + "goreleaser/test_run_pipe:latest",
			},
			assertImageLabels: shouldFindImagesWithLabels(
				"goreleaser/test_run_pipe",
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
						registry + "goreleaser/templated_dockerfile:v1",
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
				registry + "goreleaser/templated_dockerfile:v1",
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
						registry + "goreleaser/wrong_bin_name:v1",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile.wrongbin",
				},
			},
			assertError:       shouldErr("seems like you tried to copy a file that is not available in the build context"),
			assertImageLabels: noLabels,
		},
		"templated-dockerfile-invalid": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/invalid-templated-dockerfile:v1",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "{{ .Env.Dockerfile }}",
				},
			},
			expect:              []string{},
			assertError:         shouldTemplateErr,
			assertImageLabels:   noLabels,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"image template with env": {
			env: map[string]string{
				"FOO": "test_run_pipe_template",
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
				registry + "goreleaser/test_run_pipe_template:v1.0.0",
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"image template uppercase": {
			env: map[string]string{
				"FOO": "test_run_pipe_template_UPPERCASE",
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
			assertError:         shouldErr(`failed to build localhost:5050/goreleaser/test_run_pipe_template_UPPERCASE:v1.0.0`),
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"empty image tag": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						"",
						registry + "goreleaser/empty_tag:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			expect: []string{
				registry + "goreleaser/empty_tag:latest",
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
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
			expect:              []string{},
			assertImageLabels:   noLabels,
			assertError:         shouldErr("no image templates found"),
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
		"valid with ids": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/test_run_pipe_build:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					IDs:        []string{"mybin"},
				},
			},
			expect: []string{
				registry + "goreleaser/test_run_pipe_build:latest",
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
						registry + "goreleaser/multiplefiles1:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					Files:      []string{"testdata/extra_file.txt"},
				},
				{
					ImageTemplates: []string{
						registry + "goreleaser/multiplefiles2:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					Files:      []string{"testdata/extra_file.txt"},
				},
			},
			expect: []string{
				registry + "goreleaser/multiplefiles1:latest",
				registry + "goreleaser/multiplefiles2:latest",
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
						registry + "goreleaser/test_run_pipe:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
				{
					ImageTemplates: []string{
						registry + "goreleaser/test_run_pipe2:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			assertImageLabels: noLabels,
			expect: []string{
				registry + "goreleaser/test_run_pipe:latest",
				registry + "goreleaser/test_run_pipe2:latest",
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
						registry + "goreleaser/test_run_pipe:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					SkipPush:   "{{.Env.TRUE}}",
				},
			},
			expect: []string{
				registry + "goreleaser/test_run_pipe:latest",
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
						registry + "goreleaser/one_img_error_with_skip_push:true",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile.true",
					SkipPush:   "true",
				},
				{
					ImageTemplates: []string{
						registry + "goreleaser/one_img_error_with_skip_push:false",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile.false",
					SkipPush:   "true",
				},
			},
			expect: []string{
				registry + "goreleaser/one_img_error_with_skip_push:true",
			},
			assertImageLabels: noLabels,
			assertError:       shouldErr("failed to build localhost:5050/goreleaser/one_img_error_with_skip_push:false"),
		},
		"valid_no_latest": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/test_run_pipe:{{.Version}}",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			expect: []string{
				registry + "goreleaser/test_run_pipe:1.0.0",
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
						registry + "goreleaser/test_build_args:latest",
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
				registry + "goreleaser/test_build_args:latest",
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
						registry + "goreleaser/test_build_args:latest",
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
			assertError:       shouldErr("failed to build localhost:5050/goreleaser/test_build_args:latest"),
		},
		"bad_dockerfile": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/bad_dockerfile:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile.bad",
				},
			},
			assertImageLabels: noLabels,
			assertError:       shouldErr("failed to build localhost:5050/goreleaser/bad_dockerfile:latest"),
		},
		"tag_template_error": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/test_run_pipe:{{.Tag}",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			assertImageLabels: noLabels,
			assertError:       shouldTemplateErr,
		},
		"build_flag_template_error": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/test_run_pipe:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					BuildFlagTemplates: []string{
						"--label=tag={{.Tag}",
					},
				},
			},
			assertImageLabels: noLabels,
			assertError:       shouldTemplateErr,
		},
		"missing_env_on_tag_template": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/test_run_pipe:{{.Env.NOPE}}",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
				},
			},
			assertImageLabels: noLabels,
			assertError:       shouldTemplateErr,
		},
		"missing_env_on_build_flag_template": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/test_run_pipe:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					BuildFlagTemplates: []string{
						"--label=nope={{.Env.NOPE}}",
					},
				},
			},
			assertImageLabels: noLabels,
			assertError:       shouldTemplateErr,
		},
		"image_has_projectname_template_variable": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/{{.ProjectName}}:{{.Tag}}-{{.Env.FOO}}",
						registry + "goreleaser/{{.ProjectName}}:latest",
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
				registry + "goreleaser/mybin:v1.0.0-123",
				registry + "goreleaser/mybin:latest",
			},
			assertImageLabels:   noLabels,
			assertError:         shouldNotErr,
			pubAssertError:      testlib.AssertSkipped,
			manifestAssertError: shouldNotErr,
		},
		"no_permissions": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{"docker.io/nope:latest"},
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
			pubAssertError:      shouldErr(`failed to push docker.io/nope:latest`),
			manifestAssertError: shouldNotErr,
		},
		"dockerfile_doesnt_exist": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{"whatever:latest"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfilezzz",
				},
			},
			assertImageLabels: noLabels,
			assertError:       shouldErr(`failed to copy dockerfile`),
		},
		"extra_file_doesnt_exist": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{"whatever:latest"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfile",
					Files: []string{
						"testdata/nope.txt",
					},
				},
			},
			assertImageLabels: noLabels,
			assertError:       shouldErr(`failed to copy extra file 'testdata/nope.txt'`),
		},
		"binary doesnt exist": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{"whatever:latest"},
					Goos:           "linux",
					Goarch:         "amd64",
					Dockerfile:     "testdata/Dockerfile",
					IDs:            []string{"nope"},
				},
			},
			assertImageLabels: noLabels,
			assertError:       shouldErr(`failed to copy wont-exist`),
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
		"multiple_ids": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/multiple:latest"},
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
				registry + "goreleaser/multiple:latest",
			},
		},
		"nfpm and multiple binaries": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/test_nfpm:latest"},
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
				registry + "goreleaser/test_nfpm:latest",
			},
		},
		"nfpm and multiple binaries on arm64": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/test_nfpm_arm:latest"},
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
				registry + "goreleaser/test_nfpm_arm:latest",
			},
		},
	}

	start(t)

	for name, docker := range table {
		for imager := range imagers {
			t.Run(name+" on "+imager, func(t *testing.T) {
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
						Dockers:         docker.dockers,
						DockerManifests: docker.manifests,
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

				rmi := func(img string) error {
					return exec.CommandContext(t.Context(), "docker", "rmi", "--force", img).Run()
				}

				// this might fail as the image doesnt exist yet, so lets ignore the error
				for _, img := range docker.expect {
					_ = rmi(img)
				}

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
				for _, img := range docker.expect {
					// t.Log("removing docker image", img)
					require.NoError(t, rmi(img), "could not delete image %s", img)
				}

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
	}
}
