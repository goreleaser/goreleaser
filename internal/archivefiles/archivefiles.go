// Package archivefiles can evaluate a list of config.Files into their final form.
package archivefiles

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/caarlos0/log"
	"github.com/goreleaser/fileglob"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
)

// Eval evaluates the given list of files to their final form.
func Eval(template *tmpl.Template, files []config.File) ([]config.File, error) {
	var result []config.File
	for _, f := range files {
		glob, err := template.Apply(f.Source)
		if err != nil {
			return result, fmt.Errorf("failed to apply template %s: %w", f.Source, err)
		}

		files, err := fileglob.Glob(glob)
		if err != nil {
			return result, fmt.Errorf("globbing failed for pattern %s: %w", glob, err)
		}

		if len(files) == 0 {
			if !f.Default {
				// only log if its not a default glob, as those are usually
				// very generic and are not really warnings for the user.
				log.WithField("glob", f.Source).Warn("no files matched")
			}
			continue
		}

		if err := tmplInfo(template, &f.Info); err != nil {
			return result, err
		}

		// the prefix may not be a complete path or may use glob patterns, in that case use the parent directory
		prefix := glob
		if _, err := os.Stat(prefix); errors.Is(err, fs.ErrNotExist) || fileglob.ContainsMatchers(prefix) {
			prefix = filepath.Dir(longestCommonPrefix(files))
		}

		for _, file := range files {
			dst, err := destinationFor(f, prefix, file)
			if err != nil {
				return nil, err
			}
			result = append(result, config.File{
				Source:      filepath.ToSlash(file),
				Destination: filepath.ToSlash(dst),
				Info:        f.Info,
			})
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Destination < result[j].Destination
	})

	return unique(result), nil
}

func tmplInfo(template *tmpl.Template, info *config.FileInfo) error {
	if err := template.ApplyAll(
		&info.Owner,
		&info.Group,
		&info.MTime,
	); err != nil {
		return err
	}
	if info.MTime != "" {
		var err error
		info.ParsedMTime, err = time.Parse(time.RFC3339Nano, info.MTime)
		if err != nil {
			return fmt.Errorf("failed to parse %s: %w", info.MTime, err)
		}
	}
	return nil
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

func destinationFor(f config.File, prefix, path string) (string, error) {
	if f.StripParent {
		return filepath.Join(f.Destination, filepath.Base(path)), nil
	}

	if f.Destination != "" {
		relpath, err := filepath.Rel(prefix, path)
		if err != nil {
			// since prefix is a prefix of src a relative path should always be found
			return "", err
		}
		return filepath.ToSlash(filepath.Join(f.Destination, relpath)), nil
	}

	return filepath.Join(f.Destination, path), nil
}

// longestCommonPrefix returns the longest prefix of all strings the argument
// slice. If the slice is empty the empty string is returned.
// copied from nfpm
func longestCommonPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	lcp := strs[0]
	for _, str := range strs {
		lcp = strlcp(lcp, str)
	}
	return lcp
}

// copied from nfpm
func strlcp(a, b string) string {
	var min int
	if len(a) > len(b) {
		min = len(b)
	} else {
		min = len(a)
	}
	for i := 0; i < min; i++ {
		if a[i] != b[i] {
			return a[0:i]
		}
	}
	return a[0:min]
}
