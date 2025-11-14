package docker

import (
	"os"
	"path/filepath"

	"charm.land/lipgloss/v2/tree"
)

func fileNotFoundDetails(wd string) string {
	const msg = "Seems like you tried to copy a file that is not available in the build context."
	tree := tree.New().Root(wd)
	if err := buildTree(tree, wd); err != nil {
		return msg
	}
	return msg + "\n" + tree.String()
}

func buildTree(parent *tree.Tree, path string) error {
	items, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	for _, item := range items {
		if item.IsDir() {
			branch := tree.Root(item.Name())
			parent.Child(branch)
			if err := buildTree(branch, filepath.Join(path, item.Name())); err != nil {
				return err
			}
			continue
		}
		parent.Child(item.Name())
	}
	return nil
}
