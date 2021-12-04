package extrafiles

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/fileglob"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// Find resolves extra files globs et al into a map of names/paths or an error.
func Find(ctx *context.Context, files []config.ExtraFile) (map[string]string, error) {
	t := tmpl.New(ctx)
	result := map[string]string{}
	for _, extra := range files {
		glob, err := t.Apply(extra.Glob)
		if err != nil {
			return result, fmt.Errorf("failed to apply template to glob %q: %w", extra.Glob, err)
		}
		if glob == "" {
			log.Warn("ignoring empty glob")
			continue
		}
		files, err := fileglob.Glob(glob)
		if err != nil {
			return result, fmt.Errorf("globbing failed for pattern %s: %w", extra.Glob, err)
		}
		if len(files) > 1 && extra.NameTemplate != "" {
			return result, fmt.Errorf("failed to add extra_file: %q -> %q: glob matches multiple files", extra.Glob, extra.NameTemplate)
		}
		for _, file := range files {
			info, err := os.Stat(file)
			if err == nil && info.IsDir() {
				log.Debugf("ignoring directory %s", file)
				continue
			}
			n, err := t.Apply(extra.NameTemplate)
			if err != nil {
				return result, fmt.Errorf("failed to apply template to name %q: %w", extra.NameTemplate, err)
			}
			name := filepath.Base(file)
			if n != "" {
				name = n
			}
			if old, ok := result[name]; ok {
				log.Warnf("overriding %s with %s for name %s", old, file, name)
			}
			result[name] = file
		}
	}
	return result, nil
}
