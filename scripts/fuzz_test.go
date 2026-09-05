package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/goreleaser/goreleaser/v2/internal/testlib"
	"github.com/stretchr/testify/require"
)

func fakeFuzzGo(t *testing.T, exitCode int) string {
	t.Helper()
	testlib.SkipIfWindows(t, "the fuzz wrapper requires Bash")
	dir := t.TempDir()
	calls := filepath.Join(dir, "calls")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go"), []byte(`#!/bin/sh
printf '%s\n' "$*" >> "$GO_CALLS"
case "$*" in
  *-fuzz=*) exit "$FUZZ_EXIT" ;;
esac
`), 0o755))
	t.Setenv("GO_CALLS", calls)
	t.Setenv("FUZZ_EXIT", strconv.Itoa(exitCode))
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	return calls
}

func TestFuzzFailurePropagation(t *testing.T) {
	for _, command := range [][]string{
		{"bash", "scripts/fuzz.sh", "./internal/tmpl", "1ms"},
		{"task", "fuzz:tmpl"},
	} {
		t.Run(command[0], func(t *testing.T) {
			testlib.CheckPath(t, command[0])
			for _, exitCode := range []int{0, 42} {
				t.Run(strconv.Itoa(exitCode), func(t *testing.T) {
					calls := fakeFuzzGo(t, exitCode)
					cmd := exec.CommandContext(t.Context(), command[0], command[1:]...)
					cmd.Dir = ".."
					out, err := cmd.CombinedOutput()
					if exitCode == 0 {
						require.NoError(t, err, "%s", out)
					} else {
						require.Error(t, err, "%s", out)
						var exitErr *exec.ExitError
						require.ErrorAs(t, err, &exitErr)
						if command[0] == "bash" {
							require.Equal(t, exitCode, exitErr.ExitCode(), "%s", out)
						} else {
							require.Contains(t, string(out), "exit status 42")
						}
					}
					recorded, err := os.ReadFile(calls)
					require.NoError(t, err, "%s", out)
					lines := strings.Split(strings.TrimSpace(string(recorded)), "\n")
					if exitCode != 0 {
						require.Len(t, lines, 1, "%s", recorded)
						require.Contains(t, lines[0], "-fuzz=")
						return
					}
					require.Greater(t, len(lines), 1, "%s", recorded)
					require.Equal(t, "test ./internal/tmpl/...", lines[len(lines)-1])
					for _, line := range lines[:len(lines)-1] {
						require.Regexp(t, `-fuzz=\^Fuzz[A-Za-z0-9_]+\$`, line)
						require.Contains(t, line, "./internal/tmpl/...")
					}
				})
			}
		})
	}
}

func TestFuzzTaskDispatch(t *testing.T) {
	testlib.CheckPath(t, "task")
	for _, tc := range []struct {
		task     string
		packages []string
	}{
		{task: "fuzz", packages: []string{"tmpl", "artifact"}},
		{task: "fuzz:tmpl", packages: []string{"tmpl"}},
		{task: "fuzz:artifact", packages: []string{"artifact"}},
	} {
		t.Run(tc.task, func(t *testing.T) {
			calls := fakeFuzzGo(t, 0)
			dry := exec.CommandContext(t.Context(), "task", "--dry", tc.task)
			dry.Dir = ".."
			out, err := dry.CombinedOutput()
			require.NoError(t, err, "%s", out)
			for _, pkg := range tc.packages {
				require.Contains(t, string(out), "scripts/fuzz.sh ./internal/"+pkg+" 30s")
			}

			cmd := exec.CommandContext(t.Context(), "task", tc.task)
			cmd.Dir = ".."
			out, err = cmd.CombinedOutput()
			require.NoError(t, err, "%s", out)
			recorded, err := os.ReadFile(calls)
			require.NoError(t, err, "%s", out)
			for _, pkg := range tc.packages {
				require.Contains(t, string(recorded), "test ./internal/"+pkg+"/...\n")
			}
		})
	}
}
