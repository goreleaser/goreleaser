package build

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"

	"github.com/goreleaser/goreleaser/config"
	"github.com/goreleaser/goreleaser/context"
	"github.com/pkg/errors"
)

func checkMain(ctx *context.Context, build config.Build) error {
	var main = build.Main
	if main == "" {
		main = "."
	}
	stat, ferr := os.Stat(main)
	if os.IsNotExist(ferr) {
		return errors.Wrapf(ferr, "could not open %s", main)
	}
	if stat.IsDir() {
		packs, err := parser.ParseDir(token.NewFileSet(), main, nil, 0)
		if err != nil {
			return errors.Wrapf(err, "failed to parse dir: %s", main)
		}
		for _, pack := range packs {
			for _, file := range pack.Files {
				if hasMain(file) {
					return nil
				}
			}
		}
		return fmt.Errorf("build for %s does not contain a main function", build.Binary)
	}
	file, err := parser.ParseFile(token.NewFileSet(), build.Main, nil, 0)
	if err != nil {
		return errors.Wrapf(err, "failed to parse file: %s", build.Main)
	}
	if hasMain(file) {
		return nil
	}
	return fmt.Errorf("build for %s does not contain a main function", build.Binary)
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
