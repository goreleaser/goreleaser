// Package gomod provides go modules utilities, such as template variables and the ability to proxy the module from
// proxy.golang.org.
package gomod

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/caarlos0/log"
	"github.com/goreleaser/goreleaser/internal/logext"
	"github.com/goreleaser/goreleaser/internal/tmpl"
	"github.com/goreleaser/goreleaser/pkg/config"
	"github.com/goreleaser/goreleaser/pkg/context"
)

// ErrReplaceWithProxy happens when the configuration has gomod.proxy enabled,
// and the go.mod file contains replace directives.
//
// Replaces does not work with proxying, nor with go installs,
// and are made for development only.
var ErrReplaceWithProxy = errors.New("cannot use the go.mod replace directive with go mod proxy enabled")

type CheckGoModPipe struct{}

func (CheckGoModPipe) String() string { return "checking go.mod" }
func (CheckGoModPipe) Skip(ctx *context.Context) bool {
	return ctx.ModulePath == "" || !ctx.Config.GoMod.Proxy
}

var replaceRe = regexp.MustCompile("^replace .* => .*$")

// Run the ReplaceCheckPipe.
func (CheckGoModPipe) Run(ctx *context.Context) error {
	for i := range ctx.Config.Builds {
		build := &ctx.Config.Builds[i]
		path := filepath.Join(build.UnproxiedDir, "go.mod")
		mod, err := os.ReadFile(path)
		if err != nil {
			log.Errorf("could not check %q", path)
			return nil
		}
		for _, line := range strings.Split(string(mod), "\n") {
			if !replaceRe.MatchString(line) {
				continue
			}
			log.Warnf(
				"your %[2]s file has %[1]s directive in it, and go mod proxying is enabled - "+
					"this does not work, and you need to either disable it or remove the %[1]s directive",
				logext.Keyword("replace"),
				logext.Keyword("go.mod"),
			)
			log.Warnf("the offending line is %s", logext.Keyword(strings.TrimSpace(line)))
			if ctx.Snapshot {
				// only warn on snapshots
				break
			}
			return ErrReplaceWithProxy
		}
	}

	return nil
}

// ProxyPipe for gomod proxy.
type ProxyPipe struct{}

func (ProxyPipe) String() string { return "proxying go module" }

func (ProxyPipe) Skip(ctx *context.Context) bool {
	return ctx.ModulePath == "" || !ctx.Config.GoMod.Proxy || ctx.Snapshot
}

// Run the ProxyPipe.
func (ProxyPipe) Run(ctx *context.Context) error {
	for i := range ctx.Config.Builds {
		build := &ctx.Config.Builds[i]
		if err := proxyBuild(ctx, build); err != nil {
			return err
		}
	}

	return nil
}

const goModTpl = `module {{ .BuildID }}`

// ErrProxy happens when something goes wrong while proxying the current go module.
type ErrProxy struct {
	err     error
	details string
}

func newErrProxy(err error) error {
	return ErrProxy{
		err: err,
	}
}

func newDetailedErrProxy(err error, details string) error {
	return ErrProxy{
		err:     err,
		details: details,
	}
}

func (e ErrProxy) Error() string {
	out := fmt.Sprintf("failed to proxy module: %v", e.err)
	if e.details != "" {
		return fmt.Sprintf("%s: %s", out, e.details)
	}
	return out
}

func (e ErrProxy) Unwrap() error {
	return e.err
}

func proxyBuild(ctx *context.Context, build *config.Build) error {
	mainPackage := path.Join(ctx.ModulePath, build.Main)
	if strings.HasSuffix(build.Main, ".go") {
		pkg := path.Dir(build.Main)
		log.Warnf("guessing package of '%s' to be '%s', if this is incorrect, setup 'build.%s.main' to be the correct package", build.Main, pkg, build.ID)
		mainPackage = path.Join(ctx.ModulePath, pkg)
	}
	template := tmpl.New(ctx).WithExtraFields(tmpl.Fields{
		"Main":    mainPackage,
		"BuildID": build.ID,
	})

	log.Infof("proxying %s@%s to build %s", ctx.ModulePath, ctx.Git.CurrentTag, mainPackage)

	mod, err := template.Apply(goModTpl)
	if err != nil {
		return newErrProxy(err)
	}

	dir := filepath.Join(ctx.Config.Dist, "proxy", build.ID)

	log.Debugf("creating needed files")

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return newErrProxy(err)
	}

	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o666); err != nil {
		return newErrProxy(err)
	}

	if err := copyGoSum("go.sum", filepath.Join(dir, "go.sum")); err != nil {
		return newErrProxy(err)
	}

	log.Debugf("tidying")
	cmd := exec.CommandContext(ctx, ctx.Config.GoMod.GoBinary, "get", ctx.ModulePath+"@"+ctx.Git.CurrentTag)
	cmd.Dir = dir
	cmd.Env = append(ctx.Config.GoMod.Env, os.Environ()...)
	if out, err := cmd.CombinedOutput(); err != nil {
		return newDetailedErrProxy(err, string(out))
	}

	build.UnproxiedMain = build.Main
	build.UnproxiedDir = build.Dir
	build.Main = mainPackage
	build.Dir = dir
	return nil
}

func copyGoSum(src, dst string) error {
	r, err := os.OpenFile(src, os.O_RDONLY, 0o666)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	defer r.Close()

	w, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer w.Close()

	_, err = io.Copy(w, r)
	return err
}
