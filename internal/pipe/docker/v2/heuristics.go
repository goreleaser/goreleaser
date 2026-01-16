package docker

import (
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/logext"
)

var projectRootMarkers = map[string]struct{}{
	"go.mod":           {},
	"Cargo.toml":       {},
	"build.zig":        {},
	"build.zig.zon":    {},
	"gyro.zig":         {},
	"zls.build.json":   {},
	"bunfig.toml":      {},
	"deno.json":        {},
	"pyproject.toml":   {},
	"poetry.lock":      {},
	"uv.lock":          {},
	"requirements.txt": {},
	"Pipfile":          {},
}

func findRootProjectExtraFiles(extraFiles []string) []string {
	if os.Getenv("GORELEASER_NO_SLOW_DOCKER_WARN") != "" {
		return nil
	}
	found := map[string]struct{}{}
	for _, file := range extraFiles {
		base := filepath.Base(file)
		if _, ok := projectRootMarkers[base]; ok {
			found[base] = struct{}{}
		}
	}
	return slices.Collect(maps.Keys(found))
}

func emitExtraFilesWarning(markers []string) {
	details := logext.Warning("Your extra_files contain project root markers that suggest you might be building inside Docker.") +
		"\n\n" +
		"Found: " + strings.Join(markers, ", ") + "\n\n" +
		"GoReleaser already builds your binaries for all target platforms.\n" +
		"You likely don't need these files in your Docker image.\n\n" +
		"If you do need them (e.g., for runtime configuration), you can ignore this warning.\n" +
		"Otherwise, remove them from extra_files and copy pre-built binaries instead:\n" +
		logext.Keyword("  COPY $TARGETPLATFORM/mybinary /usr/bin/") + "\n\n" +
		"Learn more at " + logext.URL("https://goreleaser.com/customization/dockers_v2")

	log.WithField("details", details).
		Warn("extra_files may contain unnecessary build files")
}
