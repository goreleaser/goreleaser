package docker

import (
	"flag"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/goreleaser/goreleaser/internal/artifact"
	"github.com/goreleaser/goreleaser/pipeline"
	"github.com/stretchr/testify/assert"
)

var it = flag.Bool("it", false, "push images to docker hub")
var registry = "localhost:5000/"

func TestMain(m *testing.M) {
	flag.Parse()
	if *it {
		registry = "docker.io/"
	}
	os.Exit(m.Run())
}

func start(t *testing.T) {
	if *it {
		return
	}
	if out, err := exec.Command(
		"docker", "run", "-d", "-p", "5000:5000", "--name", "registry", "registry:2",
	).CombinedOutput(); err != nil {
		t.Log("failed to start docker registry", string(out), err)
		t.FailNow()
	}
}

func killAndRm(t *testing.T) {
	if *it {
		return
	}
	t.Log("killing registry")
	_ = exec.Command("docker", "kill", "registry").Run()
	_ = exec.Command("docker", "rm", "registry").Run()
}

func TestRunPipe(t *testing.T) {
	type errChecker func(*testing.T, error)
	var shouldErr = func(msg string) errChecker {
		return func(t *testing.T, err error) {
			assert.Error(t, err)
			assert.Contains(t, err.Error(), msg)
		}
	}
	var shouldNotErr = func(t *testing.T, err error) {
		assert.NoError(t, err)
	}

	var table = map[string]struct {
		docker      config.Docker
		publish     bool
		expect      []string
		assertError errChecker
	}{
		"valid": {
			publish: true,
			docker: config.Docker{
				Image:      registry + "goreleaser/test_run_pipe",
				Goos:       "linux",
				Goarch:     "amd64",
				Dockerfile: "testdata/Dockerfile",
				Binary:     "mybin",
				TagTemplates: []string{
					"{{.Tag}}-{{.Env.FOO}}",
					"v{{.Major}}",
					"v{{.Major}}.{{.Minor}}",
					"latest",
				},
				Files: []string{
					"testdata/extra_file.txt",
				},
			},
			expect: []string{
				registry + "goreleaser/test_run_pipe:v1.0.0-123",
				registry + "goreleaser/test_run_pipe:v1",
				registry + "goreleaser/test_run_pipe:v1.0",
				registry + "goreleaser/test_run_pipe:latest",
			},
			assertError: shouldNotErr,
		},
		"valid_skip_push": {
			publish: true,
			docker: config.Docker{
				Image:      registry + "goreleaser/test_run_pipe",
				Goos:       "linux",
				Goarch:     "amd64",
				Dockerfile: "testdata/Dockerfile",
				Binary:     "mybin",
				SkipPush:   true,
				TagTemplates: []string{
					"{{.Tag}}-{{.Env.FOO}}",
					"v{{.Major}}",
					"v{{.Major}}.{{.Minor}}",
					"latest",
				},
				Files: []string{
					"testdata/extra_file.txt",
				},
			},
			expect: []string{
				registry + "goreleaser/test_run_pipe:v1.0.0-123",
				registry + "goreleaser/test_run_pipe:v1",
				registry + "goreleaser/test_run_pipe:v1.0",
				registry + "goreleaser/test_run_pipe:latest",
			},
			assertError: shouldNotErr,
		},
		"valid_no_latest": {
			publish: true,
			docker: config.Docker{
				Image:      registry + "goreleaser/test_run_pipe",
				Goos:       "linux",
				Goarch:     "amd64",
				Dockerfile: "testdata/Dockerfile",
				Binary:     "mybin",
				TagTemplates: []string{
					"{{.Version}}",
				},
				Files: []string{
					"testdata/extra_file.txt",
				},
			},
			expect: []string{
				registry + "goreleaser/test_run_pipe:1.0.0",
			},
			assertError: shouldNotErr,
		},
		"valid_dont_publish": {
			publish: false,
			docker: config.Docker{
				Image:      registry + "goreleaser/test_run_pipe",
				Goos:       "linux",
				Goarch:     "amd64",
				Dockerfile: "testdata/Dockerfile",
				Binary:     "mybin",
				TagTemplates: []string{
					"{{.Tag}}-{{.Env.FOO}}",
					"latest",
				},
				Files: []string{
					"testdata/extra_file.txt",
				},
			},
			expect: []string{
				registry + "goreleaser/test_run_pipe:v1.0.0-123",
				registry + "goreleaser/test_run_pipe:latest",
			},
			assertError: shouldNotErr,
		},
		"bad_dockerfile": {
			publish: true,
			docker: config.Docker{
				Image:      registry + "goreleaser/test_run_pipe",
				Goos:       "linux",
				Goarch:     "amd64",
				Dockerfile: "testdata/Dockerfile.bad",
				Binary:     "mybin",
				TagTemplates: []string{
					"{{.Version}}",
				},
			},
			assertError: shouldErr("pull access denied for nope, repository does not exist"),
		},
		"template_error": {
			publish: true,
			docker: config.Docker{
				Image:      registry + "goreleaser/test_run_pipe",
				Goos:       "linux",
				Goarch:     "amd64",
				Dockerfile: "testdata/Dockerfile",
				Binary:     "mybin",
				TagTemplates: []string{
					"{{.Tag}",
				},
			},
			assertError: shouldErr(`template: tag:1: unexpected "}" in operand`),
		},
		"missing_env_on_template": {
			publish: true,
			docker: config.Docker{
				Image:      registry + "goreleaser/test_run_pipe",
				Goos:       "linux",
				Goarch:     "amd64",
				Dockerfile: "testdata/Dockerfile",
				Binary:     "mybin",
				TagTemplates: []string{
					"{{.Env.NOPE}}",
				},
			},
			assertError: shouldErr(`template: tag:1:6: executing "tag" at <.Env.NOPE>: map has no entry for key "NOPE"`),
		},
		"no_permissions": {
			publish: true,
			docker: config.Docker{
				Image:      "docker.io/nope",
				Goos:       "linux",
				Goarch:     "amd64",
				Binary:     "mybin",
				Dockerfile: "testdata/Dockerfile",
				TagTemplates: []string{
					"{{.Tag}}",
					"latest",
				},
				Latest: true,
			},
			expect: []string{
				"docker.io/nope:latest",
				"docker.io/nope:v1.0.0",
			},
			assertError: shouldErr(`requested access to the resource is denied`),
		},
		"dockerfile_doesnt_exist": {
			publish: true,
			docker: config.Docker{
				Image:      "whatever",
				Goos:       "linux",
				Goarch:     "amd64",
				Binary:     "mybin",
				Dockerfile: "testdata/Dockerfilezzz",
				TagTemplates: []string{
					"{{.Tag}}",
				},
			},
			assertError: shouldErr(`failed to link dockerfile`),
		},
		"extra_file_doesnt_exist": {
			publish: true,
			docker: config.Docker{
				Image:  "whatever",
				Goos:   "linux",
				Goarch: "amd64",
				Binary: "mybin",
				Files: []string{
					"testdata/nope.txt",
				},
				Dockerfile: "testdata/Dockerfile",
				TagTemplates: []string{
					"{{.Tag}}",
				},
			},
			assertError: shouldErr(`failed to link extra file 'testdata/nope.txt'`),
		},
		"no_matching_binaries": {
			publish: true,
			docker: config.Docker{
				Image:      "whatever",
				Goos:       "darwin",
				Goarch:     "amd64",
				Binary:     "mybinnnn",
				Dockerfile: "testdata/Dockerfile",
			},
			assertError: shouldNotErr,
		},
	}

	killAndRm(t)
	start(t)
	defer killAndRm(t)

	for name, docker := range table {
		t.Run(name, func(tt *testing.T) {
			folder, err := ioutil.TempDir("", "archivetest")
			assert.NoError(tt, err)
			var dist = filepath.Join(folder, "dist")
			assert.NoError(tt, os.Mkdir(dist, 0755))
			assert.NoError(tt, os.Mkdir(filepath.Join(dist, "mybin"), 0755))
			var binPath = filepath.Join(dist, "mybin", "mybin")
			_, err = os.Create(binPath)
			assert.NoError(tt, err)

			var ctx = context.New(config.Project{
				ProjectName: "mybin",
				Dist:        dist,
				Dockers: []config.Docker{
					docker.docker,
				},
			})
			ctx.SkipPublish = !docker.publish
			ctx.Env = map[string]string{
				"FOO": "123",
			}
			ctx.Version = "1.0.0"
			ctx.Git = context.GitInfo{
				CurrentTag: "v1.0.0",
			}
			for _, os := range []string{"linux", "darwin"} {
				for _, arch := range []string{"amd64", "386"} {
					ctx.Artifacts.Add(artifact.Artifact{
						Name:   "mybin",
						Path:   binPath,
						Goarch: arch,
						Goos:   os,
						Type:   artifact.Binary,
						Extra: map[string]string{
							"Binary": "mybin",
						},
					})
				}
			}

			// this might fail as the image doesnt exist yet, so lets ignore the error
			for _, img := range docker.expect {
				_ = exec.Command("docker", "rmi", img).Run()
			}

			docker.assertError(t, Pipe{}.Run(ctx))

			// this might should not fail as the image should have been created when
			// the step ran
			for _, img := range docker.expect {
				tt.Log("removing docker image", img)
				assert.NoError(tt, exec.Command("docker", "rmi", img).Run(), "could not delete image %s", img)
			}

		})
	}
}

func TestDescription(t *testing.T) {
	assert.NotEmpty(t, Pipe{}.String())
}

func TestNoDockers(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{}))))
}

func TestNoDockerWithoutImageName(t *testing.T) {
	assert.True(t, pipeline.IsSkip(Pipe{}.Run(context.New(config.Project{
		Dockers: []config.Docker{
			{
				Goos: "linux",
			},
		},
	}))))
}

func TestDockerNotInPath(t *testing.T) {
	var path = os.Getenv("PATH")
	defer func() {
		assert.NoError(t, os.Setenv("PATH", path))
	}()
	assert.NoError(t, os.Setenv("PATH", ""))
	var ctx = &context.Context{
		Version: "1.0.0",
		Config: config.Project{
			Dockers: []config.Docker{
				{
					Image: "a/b",
				},
			},
		},
	}
	assert.EqualError(t, Pipe{}.Run(ctx), ErrNoDocker.Error())
}

func TestDefault(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Builds: []config.Build{
				{
					Binary: "foo",
				},
			},
			Dockers: []config.Docker{
				{
					Latest: true,
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Len(t, ctx.Config.Dockers, 1)
	var docker = ctx.Config.Dockers[0]
	assert.Equal(t, "linux", docker.Goos)
	assert.Equal(t, "amd64", docker.Goarch)
	assert.Equal(t, ctx.Config.Builds[0].Binary, docker.Binary)
	assert.Equal(t, "Dockerfile", docker.Dockerfile)
	assert.Empty(t, docker.OldTagTemplate)
	assert.Equal(t, []string{"{{ .Version }}", "latest"}, docker.TagTemplates)

}

func TestDefaultNoDockers(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Dockers: []config.Docker{},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Empty(t, ctx.Config.Dockers)
}

func TestDefaultSet(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Dockers: []config.Docker{
				{
					Goos:       "windows",
					Goarch:     "i386",
					Binary:     "bar",
					Dockerfile: "Dockerfile.foo",
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Len(t, ctx.Config.Dockers, 1)
	var docker = ctx.Config.Dockers[0]
	assert.Equal(t, "windows", docker.Goos)
	assert.Equal(t, "i386", docker.Goarch)
	assert.Equal(t, "bar", docker.Binary)
	assert.Empty(t, docker.OldTagTemplate)
	assert.Equal(t, []string{"{{ .Version }}"}, docker.TagTemplates)
	assert.Equal(t, "Dockerfile.foo", docker.Dockerfile)
}

func TestDefaultWithOldTagTemplateSet(t *testing.T) {
	var ctx = &context.Context{
		Config: config.Project{
			Dockers: []config.Docker{
				{
					Dockerfile:     "Dockerfile.foo",
					OldTagTemplate: "{{.Tag}}",
					Latest:         true,
					Binary:         "foo",
				},
			},
		},
	}
	assert.NoError(t, Pipe{}.Default(ctx))
	assert.Len(t, ctx.Config.Dockers, 1)
	var docker = ctx.Config.Dockers[0]
	assert.Equal(t, []string{"{{.Tag}}", "latest"}, docker.TagTemplates)
	assert.Equal(t, "Dockerfile.foo", docker.Dockerfile)
}

func TestLinkFile(t *testing.T) {
	const srcFile = "/tmp/test"
	const dstFile = "/tmp/linked"
	err := ioutil.WriteFile(srcFile, []byte("foo"), 0644)
	if err != nil {
		t.Log("Cannot setup test file")
		t.Fail()
	}
	err = link(srcFile, dstFile)
	if err != nil {
		t.Log("Failed to link: ", err)
		t.Fail()
	}
	if inode(srcFile) != inode(dstFile) {
		t.Log("Inodes do not match, destination file is not a link")
		t.Fail()
	}
	// cleanup
	os.Remove(srcFile)
	os.Remove(dstFile)
}

func TestLinkDirectory(t *testing.T) {
	const srcDir = "/tmp/testdir"
	const testFile = "test"
	const dstDir = "/tmp/linkedDir"

	os.Mkdir(srcDir, 0755)
	err := ioutil.WriteFile(srcDir+"/"+testFile, []byte("foo"), 0644)
	if err != nil {
		t.Log("Cannot setup test file")
		t.Fail()
	}
	err = link(srcDir, dstDir)
	if err != nil {
		t.Log("Failed to link: ", err)
		t.Fail()
	}
	if inode(srcDir+"/"+testFile) != inode(dstDir+"/"+testFile) {
		t.Log("Inodes do not match, destination file is not a link")
		t.Fail()
	}

	// cleanup
	os.RemoveAll(srcDir)
	os.RemoveAll(dstDir)
}

func TestLinkTwoLevelDirectory(t *testing.T) {
	const srcDir = "/tmp/testdir"
	const srcLevel2 = srcDir + "/level2"
	const testFile = "test"
	const dstDir = "/tmp/linkedDir"

	os.Mkdir(srcDir, 0755)
	os.Mkdir(srcLevel2, 0755)
	err := ioutil.WriteFile(srcDir+"/"+testFile, []byte("foo"), 0644)
	if err != nil {
		t.Log("Cannot setup test file")
		t.Fail()
	}
	err = ioutil.WriteFile(srcLevel2+"/"+testFile, []byte("foo"), 0644)
	if err != nil {
		t.Log("Cannot setup test file")
		t.Fail()
	}
	err = link(srcDir, dstDir)
	if err != nil {
		t.Log("Failed to link: ", err)
		t.Fail()
	}
	if inode(srcDir+"/"+testFile) != inode(dstDir+"/"+testFile) {
		t.Log("Inodes do not match")
		t.Fail()
	}
	if inode(srcLevel2+"/"+testFile) != inode(dstDir+"/level2/"+testFile) {
		t.Log("Inodes do not match")
		t.Fail()
	}
	// cleanup
	os.RemoveAll(srcDir)
	os.RemoveAll(dstDir)
}

func inode(file string) uint64 {
	fileInfo, err := os.Stat(file)
	if err != nil {
		return 0
	}
	stat := fileInfo.Sys().(*syscall.Stat_t)
	return stat.Ino
}
