package git

import (
	"strings"

	"github.com/goreleaser/goreleaser/internal/git"
)

func cleanGit(args ...string) (output string, err error) {
	output, err = git.Run(args...)
	return strings.Replace(strings.Split(output, "\n")[0], "'", "", -1), err
}
