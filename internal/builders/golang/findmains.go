package golang

import (
	"cmp"
	"errors"
	"fmt"
	"go/ast"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/caarlos0/log"
	"golang.org/x/tools/go/packages"
)

var errNoMains = errors.New("no main functions found")

func findMains(dir string, patterns ...string) (map[string]string, error) {
	dirabs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("could not find '%s' absolute path: %w", dir, err)
	}

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedDeps |
			packages.NeedFiles |
			packages.NeedModule,
		Tests: false,
		Dir:   dirabs,
	}

	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("could not load packages: %w", err)
	}

	result := make(map[string]string) // binaryName → dir
	for _, pkg := range pkgs {
		if pkg.Name != "main" {
			continue
		}

		if !hasMainFunc(pkg) {
			continue
		}

		binaryName := computeBinaryName(pkg)
		if binaryName == "" {
			log.WithField("package", pkg.PkgPath).Warn("didn't find a binary name for package")
			continue
		}

		relPkgDir := pkg.Dir
		if rel, err := filepath.Rel(dirabs, relPkgDir); err == nil {
			relPkgDir = cmp.Or(rel, ".")
			if relPkgDir != "." {
				relPkgDir = "./" + relPkgDir
			}
		}

		if old, exists := result[binaryName]; exists && old != dir {
			log.Warnf("duplicate binary name %q for packages %q and %q", binaryName, old, relPkgDir)
			continue
		}

		result[binaryName] = relPkgDir
	}

	if len(result) == 0 {
		return nil, errNoMains
	}

	return result, nil
}

func hasMainFunc(pkg *packages.Package) bool {
	return slices.ContainsFunc(pkg.Syntax, hasMain)
}

func hasMain(file *ast.File) bool {
	for _, decl := range file.Decls {
		fn, isFn := decl.(*ast.FuncDecl)
		if !isFn {
			continue
		}
		if fn.Name.Name == "main" && fn.Recv == nil {
			return true
		}
	}
	return false
}

func computeBinaryName(pkg *packages.Package) string {
	if pkg.Module == nil {
		return path.Base(pkg.Dir)
	}

	modulePath := pkg.Module.Path
	moduleDir := pkg.Module.Dir

	isRoot := pkg.Dir == moduleDir ||
		strings.TrimRight(pkg.Dir, string(filepath.Separator)) == moduleDir

	if !isRoot {
		return path.Base(pkg.Dir)
	}

	base := path.Base(modulePath)

	if base[0] == 'v' {
		if len(base) > 1 && isDigits(base[1:]) {
			base = path.Base(strings.TrimSuffix(modulePath, "/"+base))
		}
	}

	return base
}

func isDigits(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}
