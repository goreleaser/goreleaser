// npm helpers for the Node.js SEA builder.
//
// This file implements the auto-bundle step the Node.js SEA builder
// runs before invoking `node --build-sea`: when package.json declares
// a non-empty `scripts.build` entry, the builder shells out to
// `npm run build` in the build directory so the file referenced by
// `main` is the freshly bundled output.
//
// Project-specific install / dependency setup remains the user's
// responsibility — typically done in the `before:` hook with
// `npm ci` or `pnpm install --frozen-lockfile`. This package
// deliberately does not run any install step itself: the auto-bundle
// has to be safe to invoke unconditionally, so it sticks to the one
// command we can guarantee will not mutate the project's lockfile or
// touch the network (`npm run build` against an existing
// `node_modules/`).

package nodesea

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/v2/internal/packagejson"
)

// RunNPMBuild runs `npm run build` in dir when the project's
// package.json declares a non-empty `scripts.build` entry. A missing
// package.json or a missing `scripts.build` entry is not an error: the
// function returns nil and the caller's build proceeds against
// whatever `build.Main` already resolves to.
//
// env, stdout and stderr are passed straight to the spawned process.
// When env is nil, os.Environ() is inherited.
func RunNPMBuild(ctx context.Context, dir string, env []string, stdout, stderr io.Writer) error {
	pkg, err := packagejson.OpenOrEmpty(filepath.Join(dir, "package.json"))
	if err != nil {
		return fmt.Errorf("nodesea: scan package.json: %w", err)
	}
	if !pkg.Scripts.HasBuild() {
		log.WithField("dir", dir).
			Debug("no scripts.build in package.json; skipping auto-bundle")
		return nil
	}
	log.WithField("dir", dir).Info("running npm run build")
	cmd := exec.CommandContext(ctx, "npm", "run", "build")
	cmd.Dir = dir
	cmd.Env = env
	if cmd.Env == nil {
		cmd.Env = os.Environ()
	}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("nodesea: npm run build: %w", err)
	}
	return nil
}
