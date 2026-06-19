package hzutil

import (
	"path"
	"path/filepath"
)

func SubPackage(module, dir string) string {
	if dir == "" {
		return module
	}
	return path.Join(module, filepath.ToSlash(filepath.Clean(dir)))
}

func SubDir(root, subPkg string) string {
	return filepath.ToSlash(filepath.Join(root, filepath.FromSlash(subPkg)))
}
