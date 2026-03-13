// Package gomain helps find all the `func main`'s in a given dir following
// some patterns.
package gomain

import (
	"cmp"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode"

	"github.com/caarlos0/log"
	"golang.org/x/tools/go/packages"
)

// ErrNoMains happens when, after scanning the module, we still didn't find any
// 'func main'.
var ErrNoMains = errors.New("directory does not contain any main function\nLearn more at https://goreleaser.com/errors/no-main\n") //nolint:revive,staticcheck

// ErrNoMain happens when no 'func main' is found at a specific path.
type ErrNoMain struct {
	bin string
}

func (e ErrNoMain) Error() string {
	return fmt.Sprintf("build for %s does not contain a main function\nLearn more at https://goreleaser.com/errors/no-main\n", e.bin)
}

// Check checks if the given 'main' (either file or directory) contains a 'func
// main', returning an error otherwise.
//
// Deprecated: this is here for backward compatibility, but it is recommended
// to use [All] instead, which is more robust and works on modules.
func Check(main, binary string) error {
	stat, ferr := os.Stat(main)
	if ferr != nil {
		return fmt.Errorf("couldn't find main file: %w", ferr)
	}
	if stat.IsDir() {
		packs, err := parser.ParseDir(token.NewFileSet(), main, nil, 0)
		if err != nil {
			return fmt.Errorf("failed to parse dir: %s: %w", main, err)
		}
		for _, pack := range packs {
			for _, file := range pack.Files {
				if hasMain(file) {
					return nil
				}
			}
		}
		return ErrNoMain{binary}
	}
	file, err := parser.ParseFile(token.NewFileSet(), main, nil, 0)
	if err != nil {
		return fmt.Errorf("failed to parse file: %s: %w", main, err)
	}
	if hasMain(file) {
		return nil
	}
	return ErrNoMain{binary}
}

const mode = packages.NeedName |
	packages.NeedTypes |
	packages.NeedSyntax |
	packages.NeedTypesInfo |
	packages.NeedDeps |
	packages.NeedFiles |
	packages.NeedModule

// All finds all the `func main`'s in the given dir following the given patterns.
// The result is either a map of binaryName -> ./relative/path or an error.
// This only works on a go module.
func All(dir string, patterns ...string) (map[string]string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("could not find '%s' absolute path: %w", dir, err)
	}

	cfg := &packages.Config{
		Mode: mode,
		Dir:  absDir,
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

		binaryName := BinaryNameFor(pkg)
		if binaryName == "" {
			log.WithField("package", pkg.PkgPath).Warn("didn't find a binary name for package")
			continue
		}

		relPkgDir := pkg.Dir
		if rel, err := filepath.Rel(absDir, relPkgDir); err == nil {
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
		return nil, ErrNoMains
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

// BinaryNameFor returns the 'go build' chosen binary name for the given
// package.
func BinaryNameFor(pkg *packages.Package) string {
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
