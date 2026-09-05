package docker

import (
	"bufio"
	stdctx "context"
	"crypto/rand"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/artifact"
	"github.com/goreleaser/goreleaser/v2/internal/testctx"
	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestPushDenied(t *testing.T) {
	testlib.CheckDocker(t)
	testlib.SkipIfWindows(t, "the registry fixture is a Linux container")

	// The native builder needs no downloaded BuildKit container.
	out, err := exec.CommandContext(t.Context(), "docker", "context", "show").CombinedOutput()
	require.NoError(t, err, "%s", out)
	t.Setenv("BUILDX_BUILDER", strings.TrimSpace(string(out)))

	out, err = exec.CommandContext(t.Context(), "docker", "version", "--format", "{{.Server.Arch}}").CombinedOutput()
	require.NoError(t, err, "%s", out)
	arch := strings.TrimSpace(string(out))
	dir := t.TempDir()
	compile := exec.CommandContext(t.Context(), "go", "build", "-o", filepath.Join(dir, "registry"), "testdata/registry/main.go")
	compile.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0")
	out, err = compile.CombinedOutput()
	require.NoError(t, err, "%s", out)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(
		"FROM scratch\nCOPY registry /registry\nENTRYPOINT [\"/registry\"]\n",
	), 0o644))

	registryImage := "goreleaser-test-denied:" + strings.ToLower(rand.Text())
	out, err = exec.CommandContext(t.Context(), "docker", "build", "--load", "-t", registryImage, dir).CombinedOutput()
	require.NoError(t, err, "%s", out)
	t.Cleanup(func() {
		out, err := exec.Command("docker", "rmi", registryImage).CombinedOutput()
		require.NoError(t, err, "%s", out)
	})

	out, err = exec.CommandContext(t.Context(), "docker", "run", "--detach", "--publish", "127.0.0.1::5000", registryImage).CombinedOutput()
	require.NoError(t, err, "%s", out)
	containerID := strings.TrimSpace(string(out))
	t.Cleanup(func() {
		out, err := exec.Command("docker", "rm", "--force", containerID).CombinedOutput()
		require.NoError(t, err, "%s", out)
	})

	logctx, cancel := stdctx.WithCancel(t.Context())
	t.Cleanup(cancel)
	logs := exec.CommandContext(logctx, "docker", "logs", "--follow", containerID)
	stdout, err := logs.StdoutPipe()
	require.NoError(t, err)
	logs.Stderr = logs.Stdout
	require.NoError(t, logs.Start())
	t.Cleanup(func() {
		cancel()
		_ = logs.Wait()
	})
	scanner := bufio.NewScanner(stdout)
	require.True(t, scanner.Scan(), "registry exited before readiness: %v", scanner.Err())
	require.Equal(t, "ready", scanner.Text())

	out, err = exec.CommandContext(t.Context(), "docker", "port", containerID, "5000/tcp").CombinedOutput()
	require.NoError(t, err, "%s", out)
	_, port, err := net.SplitHostPort(strings.TrimSpace(string(out)))
	require.NoError(t, err)

	dockerfile := filepath.Join(dir, "Dockerfile.image")
	require.NoError(t, os.WriteFile(dockerfile, []byte("FROM scratch\nCOPY mybin /mybin\n"), 0o644))
	binary := filepath.Join(dir, "mybin")
	require.NoError(t, os.WriteFile(binary, []byte("binary"), 0o644))
	for _, use := range []string{useDocker, useBuildx} {
		t.Run(use, func(t *testing.T) {
			image := "localhost:" + port + "/goreleaser/denied:" + use
			ctx := testctx.WrapWithCfg(t.Context(), config.Project{
				Dist: t.TempDir(),
				Dockers: []config.Docker{{
					Use:            use,
					Dockerfile:     dockerfile,
					ImageTemplates: []string{image},
				}},
			})
			ctx.Artifacts.Add(&artifact.Artifact{
				Name:    "mybin",
				Path:    binary,
				Type:    artifact.Binary,
				Goos:    "linux",
				Goarch:  "amd64",
				Goamd64: "v1",
				Extra:   artifact.Extras{artifact.ExtraID: "mybin"},
			})
			require.NoError(t, Pipe{}.Default(ctx))
			require.NoError(t, Pipe{}.Run(ctx))
			t.Cleanup(func() {
				out, err := exec.Command("docker", "rmi", image).CombinedOutput()
				require.NoError(t, err, "%s", out)
			})

			err := Pipe{}.Publish(ctx)
			require.ErrorContains(t, err, "failed to push "+image)
			require.ErrorContains(t, err, "test registry denied this push")
		})
	}
}
