// Package archivefiles can evaluate a list of config.Files into their final form.
package archivefiles

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/caarlos0/log"
	"github.com/goreleaser/fileglob"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
)

// Eval evaluates the given list of files to their final form.
func Eval(template *tmpl.Template, files []config.File) ([]config.File, error) {
	var result []config.File
	for _, f := range files {
		replaced, err := template.Apply(f.Source)
		if err != nil {
			return result, fmt.Errorf("failed to apply template %s: %w", f.Source, err)
		}

		files, err := fileglob.Glob(replaced)
		if err != nil {
			return result, fmt.Errorf("globbing failed for pattern %s: %w", replaced, err)
		}

		for _, file := range files {
			result = append(result, config.File{
				Source:      file,
				Destination: destinationFor(f, file),
				Info:        f.Info,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Destination < result[j].Destination
	})

	return unique(result), nil
}

// remove duplicates
func unique(in []config.File) []config.File {
	var result []config.File
	exist := map[string]string{}
	for _, f := range in {
		if current := exist[f.Destination]; current != "" {
			log.Warnf(
				"file '%s' already exists in archive as '%s' - '%s' will be ignored",
				f.Destination,
				current,
				f.Source,
			)
			continue
		}
		exist[f.Destination] = f.Source
		result = append(result, f)
	}

	return result
}

func destinationFor(f config.File, path string) string {
	if f.StripParent {
		return filepath.Join(f.Destination, filepath.Base(path))
	}
	return filepath.Join(f.Destination, path)
}
