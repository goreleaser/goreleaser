package extrafiles

import (
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/mattn/go-zglob"
	"github.com/pkg/errors"
)

func Find(files []config.ExtraFile) (map[string]string, error) {
	var result = map[string]string{}
	for _, extra := range files {
		if extra.Glob != "" {
			files, err := zglob.Glob(extra.Glob)
			if err != nil {
				return result, errors.Wrapf(err, "globbing failed for pattern %s", extra.Glob)
			}
			for _, file := range files {
				info, err := os.Stat(file)
				if err == nil && info.IsDir() {
					log.Debugf("ignoring directory %s", file)
					continue
				}
				var name = filepath.Base(file)
				if old, ok := result[name]; ok {
					log.Warnf("overriding %s with %s for name %s", old, file, name)
				}
				result[name] = file
			}
		}
	}
	return result, nil
}
