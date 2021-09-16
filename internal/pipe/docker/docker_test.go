package docker

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/internal/pipe"
	"github.com/goreleaser/goreleaser/internal/testlib"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
	"github.com/stretchr/testify/require"
)

var (
	it          = flag.Bool("it", false, "push images to docker hub")
	debug       = flag.Bool("debug", false, "enable debug logs")
	registry    = "localhost:5000/"
	altRegistry = "localhost:5050/"
)

func TestMain(m *testing.M) {
	flag.Parse()
	if *it {
		registry = "docker.io/"
	}
	if *debug {
		log.SetLevel(log.DebugLevel)
	}
	os.Exit(m.Run())
}

func start(t *testing.T) {
	t.Helper()
	if *it {
		return
	}
	t.Log("starting registries")
	for _, line := range []string{
		"run -d -p 5000:5000 --name registry registry:2",
		"run -d -p 5050:5000 --name alt_registry registry:2",
	} {
		if out, err := exec.Command("docker", strings.Fields(line)...).CombinedOutput(); err != nil {
			t.Log("failed to start docker registry", string(out), err)
			t.FailNow()
		}
	}
}

func killAndRm(t *testing.T) {
	t.Helper()
	if *it {
		return
	}
	t.Log("killing registries")
	for _, registry := range []string{"registry", "alt_registry"} {
		_ = exec.Command("docker", "rm", "--force", registry).Run()
	}
}

// TODO: this test is too big... split in smaller tests? Mainly the manifest ones...
func TestRunPipe(t *testing.T) {
	type errChecker func(*testing.T, error)
	shouldErr := func(msg string) errChecker {
		return func(t *testing.T, err error) {
			t.Helper()
			require.Error(t, err)
			require.Contains(t, err.Error(), msg)
		}
	}
	shouldNotErr := func(t *testing.T, err error) {
		t.Helper()
		require.NoError(t, err)
	}
	type imageLabelFinder func(*testing.T, string)
	shouldFindImagesWithLabels := func(image string, filters ...string) func(*testing.T, string) {
		return func(t *testing.T, use string) {
			t.Helper()
			for _, filter := range filters {
				cmd := exec.Command("docker", "images", "-q", "--filter", "reference=*/"+image, "--filter", filter)
				t.Log("running", cmd)
				output, err := cmd.CombinedOutput()
				require.NoError(t, err, string(output))
				uniqueIDs := map[string]string{}
				for _, id := range strings.Split(strings.TrimSpace(string(output)), "\n") {
					uniqueIDs[id] = id
				}
				require.Equal(t, 1, len(uniqueIDs))
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
					BuildFlagTemplates: []string{"--build-arg", "ARCH=amd64"},
				},
				{
					ImageTemplates:     []string{registry + "goreleaser/test_multiarch:test-arm64v8"},
					Goos:               "linux",
					Goarch:             "arm64",
					Dockerfile:         "testdata/Dockerfile.arch",
					BuildFlagTemplates: []string{"--build-arg", "ARCH=arm64v8"},
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
					SkipPush: "auto",
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
					BuildFlagTemplates: []string{"--build-arg", "ARCH=amd64"},
				},
				{
					ImageTemplates:     []string{registry + "goreleaser/test_multiarch:2test-arm64v8"},
					Goos:               "linux",
					Goarch:             "arm64",
					Dockerfile:         "testdata/Dockerfile.arch",
					BuildFlagTemplates: []string{"--build-arg", "ARCH=arm64v8"},
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
				_ = exec.Command(fmt.Sprintf("docker manifest rm %sgoreleaser/test_multiarch:2test ", registry)).Run()
				for _, cmd := range []string{
					fmt.Sprintf("docker build -t %sgoreleaser/dummy:v1 --platform linux/amd64 -f testdata/Dockerfile.dummy .", registry),
					fmt.Sprintf("docker push %sgoreleaser/dummy:v1", registry),
					fmt.Sprintf("docker manifest create %sgoreleaser/test_multiarch:2test --amend %sgoreleaser/dummy:v1 --insecure", registry, registry),
				} {
					t.Log("running", cmd)
					parts := strings.Fields(cmd)
					out, err := exec.CommandContext(ctx, parts[0], parts[1:]...).CombinedOutput()
					require.NoError(t, err, cmd+": "+string(out))
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
					BuildFlagTemplates: []string{"--build-arg", "ARCH=arm64v8"},
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
			manifestAssertError: shouldErr("failed to create localhost:5000/goreleaser/test_multiarch_fail:test"),
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
			manifestAssertError: shouldErr(`template: tmpl:1: unexpected "}" in operand`),
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
			manifestAssertError: shouldErr(`template: tmpl:1: unexpected "}" in operand`),
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
			assertError:         shouldErr(`failed to build localhost:5000/goreleaser/test_run_pipe_template_UPPERCASE:v1.0.0`),
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
			dockers: []config.Docker{
				{
					ImageTemplates: []string{
						registry + "goreleaser/test_run_pipe:latest",
					},
					Goos:       "linux",
					Goarch:     "amd64",
					Dockerfile: "testdata/Dockerfile",
					SkipPush:   "true",
				},
			},
			expect: []string{
				registry + "goreleaser/test_run_pipe:latest",
			},
			assertImageLabels: noLabels,
			assertError:       testlib.AssertSkipped,
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
			assertError:       shouldErr("failed to build localhost:5000/goreleaser/one_img_error_with_skip_push:false"),
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
			assertError:       shouldErr("failed to build localhost:5000/goreleaser/test_build_args:latest"),
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
			assertError:       shouldErr("failed to build localhost:5000/goreleaser/bad_dockerfile:latest"),
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
			assertError:       shouldErr(`template: tmpl:1: unexpected "}" in operand`),
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
			assertError:       shouldErr(`template: tmpl:1: unexpected "}" in operand`),
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
			assertError:       shouldErr(`template: tmpl:1:46: executing "tmpl" at <.Env.NOPE>: map has no entry for key "NOPE"`),
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
			assertError:       shouldErr(`template: tmpl:1:19: executing "tmpl" at <.Env.NOPE>: map has no entry for key "NOPE"`),
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
			assertImageLabels: noLabels,
			assertError:       testlib.AssertSkipped,
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
					Extra: map[string]interface{}{
						"ID": "nope",
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
					IDs:            []string{"mybin", "anotherbin"},
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
					ImageTemplates: []string{registry + "goreleaser/nfpm:latest"},
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
				registry + "goreleaser/nfpm:latest",
			},
		},
		"nfpm and multiple binaries on arm64": {
			dockers: []config.Docker{
				{
					ImageTemplates: []string{registry + "goreleaser/nfpm_arm:latest"},
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
				registry + "goreleaser/nfpm_arm:latest",
			},
		},
	}

	killAndRm(t)
	start(t)
	defer killAndRm(t)

	for name, docker := range table {
		for imager := range imagers {
			if imager == useBuildPacks { // buildpack tests are different
				continue
			}
			t.Run(name+" on "+imager, func(t *testing.T) {
				folder := t.TempDir()
				dist := filepath.Join(folder, "dist")
				require.NoError(t, os.Mkdir(dist, 0o755))
				require.NoError(t, os.Mkdir(filepath.Join(dist, "mybin"), 0o755))
				f, err := os.Create(filepath.Join(dist, "mybin", "mybin"))
				require.NoError(t, err)
				require.NoError(t, f.Close())
				f, err = os.Create(filepath.Join(dist, "mybin", "anotherbin"))
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

				ctx := context.New(config.Project{
					ProjectName:     "mybin",
					Dist:            dist,
					Dockers:         docker.dockers,
					DockerManifests: docker.manifests,
				})
				ctx.Parallelism = 1
				ctx.Env = docker.env
				ctx.Version = "1.0.0"
				ctx.Git = context.GitInfo{
					CurrentTag: "v1.0.0",
					Commit:     "a1b2c3d4",
				}
				ctx.Semver = context.Semver{
					Major: 1,
					Minor: 0,
					Patch: 0,
				}
				for _, os := range []string{"linux", "darwin"} {
					for _, arch := range []string{"amd64", "386", "arm64"} {
						for _, bin := range []string{"mybin", "anotherbin"} {
							ctx.Artifacts.Add(&artifact.Artifact{
								Name:   bin,
								Path:   filepath.Join(dist, "mybin", bin),
								Goarch: arch,
								Goos:   os,
								Type:   artifact.Binary,
								Extra: map[string]interface{}{
									"ID": bin,
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
						Extra: map[string]interface{}{
							"ID": "mybin",
						},
					})
				}

				if docker.extraPrepare != nil {
					docker.extraPrepare(t, ctx)
				}

				rmi := func(img string) error {
					return exec.Command("docker", "rmi", "--force", img).Run()
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
					t.Log("removing docker image", img)
					require.NoError(t, rmi(img), "could not delete image %s", img)
				}
			})
		}
	}
}

func TestRunPipeWhileUsingBuildpacks(t *testing.T) {
	type errChecker func(*testing.T, error)
	shouldNotErr := func(t *testing.T, err error) {
		t.Helper()
		require.NoError(t, err)
	}

	table := map[string]struct {
		dockers             []config.Docker
		manifests           []config.DockerManifest
		env                 map[string]string
		expect              []string
		assertError         errChecker
		pubAssertError      errChecker
		manifestAssertError errChecker
	}{
		"golang": {
			dockers: []config.Docker{
				{
					Use:            useBuildPacks,
					ImageTemplates: []string{registry + "goreleaser/buildpacks:{{ .Tag }}"},
				},
			},
			expect: []string{
				registry + "goreleaser/buildpacks:v1.0.0",
			},
			assertError:         shouldNotErr,
			pubAssertError:      shouldNotErr,
			manifestAssertError: shouldNotErr,
		},
	}

	killAndRm(t)
	start(t)
	defer killAndRm(t)

	for name, docker := range table {
		t.Run(name+" on "+useBuildPacks, func(t *testing.T) {
			folder := t.TempDir()
			wd := filepath.Join(folder, "wd")
			dist := filepath.Join(wd, "dist")
			require.NoError(t, os.Mkdir(wd, 0o755))
			require.NoError(t, os.Mkdir(dist, 0o755))
			require.NoError(t, os.Chdir(wd))
			require.NoError(t, os.WriteFile(filepath.Join(wd, "go.mod"), []byte("module test-mod\n\ngo 1.17"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(wd, "main.go"), []byte("package main\n\nfunc main(){}"), 0o600))

			ctx := context.New(config.Project{
				ProjectName: "mybin",
				Dist:        dist,
				Dockers:     docker.dockers,
			})
			ctx.Parallelism = 1
			ctx.Env = docker.env
			ctx.Version = "1.0.0"
			ctx.Git = context.GitInfo{
				CurrentTag: "v1.0.0",
				Commit:     "a1b2c3d4",
			}
			ctx.Semver = context.Semver{
				Major: 1,
				Minor: 0,
				Patch: 0,
			}

			rmi := func(img string) error {
				return exec.Command("docker", "rmi", "--force", img).Run()
			}

			err := Pipe{}.Run(ctx)
			docker.assertError(t, err)
			if err == nil {
				docker.pubAssertError(t, Pipe{}.Publish(ctx))
				docker.manifestAssertError(t, ManifestPipe{}.Publish(ctx))
			}

			// this might should not fail as the image should have been created when
			// the step ran
			for _, img := range docker.expect {
				t.Log("removing docker image", img)
				require.NoError(t, rmi(img), "could not delete image %s", img)
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
			expect: []string{"build", ".", "-t", images[0], "-t", images[1]},
		},
		{
			name:   "single flag",
			flags:  []string{"--label=foo"},
			expect: []string{"build", ".", "-t", images[0], "-t", images[1], "--label=foo"},
		},
		{
			name:   "multiple flags",
			flags:  []string{"--label=foo", "--build-arg=bar=baz"},
			expect: []string{"build", ".", "-t", images[0], "-t", images[1], "--label=foo", "--build-arg=bar=baz"},
		},
		{
			name:   "buildx",
			buildx: true,
			flags:  []string{"--label=foo", "--build-arg=bar=baz"},
			expect: []string{"buildx", "build", ".", "--load", "-t", images[0], "-t", images[1], "--label=foo", "--build-arg=bar=baz"},
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

func TestNoDockers(t *testing.T) {
	require.True(t, pipe.IsSkip(Pipe{}.Run(context.New(config.Project{}))))
}

func TestNoDockerWithoutImageName(t *testing.T) {
	require.True(t, pipe.IsSkip(Pipe{}.Run(context.New(config.Project{
		Dockers: []config.Docker{
			{
				Goos: "linux",
			},
		},
	}))))
}

func TestDefault(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
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
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Dockers, 2)
	docker := ctx.Config.Dockers[0]
	require.Equal(t, "linux", docker.Goos)
	require.Equal(t, "amd64", docker.Goarch)
	require.Equal(t, []string{"aa"}, docker.IDs)
	require.Equal(t, useDocker, docker.Use)
	docker = ctx.Config.Dockers[1]
	require.Equal(t, useBuildx, docker.Use)

	require.NoError(t, ManifestPipe{}.Default(ctx))
	require.Len(t, ctx.Config.DockerManifests, 2)
	require.Equal(t, useDocker, ctx.Config.DockerManifests[0].Use)
	require.Equal(t, useDocker, ctx.Config.DockerManifests[1].Use)
}

func TestDefaultDuplicateID(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Dockers: []config.Docker{
				{ID: "foo"},
				{ /* empty */ },
				{ID: "bar"},
				{ID: "foo"},
			},
			DockerManifests: []config.DockerManifest{
				{ID: "bar"},
				{ /* empty */ },
				{ID: "bar"},
				{ID: "foo"},
			},
		},
	}
	require.EqualError(t, Pipe{}.Default(ctx), "found 2 dockers with the ID 'foo', please fix your config")
	require.EqualError(t, ManifestPipe{}.Default(ctx), "found 2 docker_manifests with the ID 'bar', please fix your config")
}

func TestDefaultInvalidUse(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
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
		},
	}
	err := Pipe{}.Default(ctx)
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), `docker: invalid use: something, valid options are`))

	err = ManifestPipe{}.Default(ctx)
	require.Error(t, err)
	require.True(t, strings.HasPrefix(err.Error(), `docker manifest: invalid use: something, valid options are`))
}

func TestDefaultDockerfile(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Builds: []config.Build{
				{},
			},
			Dockers: []config.Docker{
				{},
				{},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Dockers, 2)
	require.Equal(t, "Dockerfile", ctx.Config.Dockers[0].Dockerfile)
	require.Equal(t, "Dockerfile", ctx.Config.Dockers[1].Dockerfile)
}

func TestDraftRelease(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Release: config.Release{
				Draft: true,
			},
		},
	}

	require.False(t, pipe.IsSkip(Pipe{}.Publish(ctx)))
}

func TestDefaultNoDockers(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Dockers: []config.Docker{},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Empty(t, ctx.Config.Dockers)
}

func TestDefaultFilesDot(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Dist: "/tmp/distt",
			Dockers: []config.Docker{
				{
					Files: []string{"./lala", "./lolsob", "."},
				},
			},
		},
	}
	require.EqualError(t, Pipe{}.Default(ctx), `invalid docker.files: can't be . or inside dist folder: .`)
}

func TestDefaultFilesDis(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Dist: "/tmp/dist",
			Dockers: []config.Docker{
				{
					Files: []string{"./fooo", "/tmp/dist/asdasd/asd", "./bar"},
				},
			},
		},
	}
	require.EqualError(t, Pipe{}.Default(ctx), `invalid docker.files: can't be . or inside dist folder: /tmp/dist/asdasd/asd`)
}

func TestDefaultSet(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
			Dockers: []config.Docker{
				{
					IDs:        []string{"foo"},
					Goos:       "windows",
					Goarch:     "i386",
					Dockerfile: "Dockerfile.foo",
				},
			},
		},
	}
	require.NoError(t, Pipe{}.Default(ctx))
	require.Len(t, ctx.Config.Dockers, 1)
	docker := ctx.Config.Dockers[0]
	require.Equal(t, "windows", docker.Goos)
	require.Equal(t, "i386", docker.Goarch)
	require.Equal(t, []string{"foo"}, docker.IDs)
	require.Equal(t, "Dockerfile.foo", docker.Dockerfile)
}

func Test_processImageTemplates(t *testing.T) {
	ctx := &context.Context{
		Config: config.Project{
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
		},
	}
	ctx.SkipPublish = true
	ctx.Env = map[string]string{
		"FOO": "123",
	}
	ctx.Version = "1.0.0"
	ctx.Git = context.GitInfo{
		CurrentTag: "v1.0.0",
		Commit:     "a1b2c3d4",
	}
	ctx.Semver = context.Semver{
		Major: 1,
		Minor: 0,
		Patch: 0,
	}

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
