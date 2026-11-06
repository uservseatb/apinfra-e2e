package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// repoRoot is the module root (parent of this package directory).
func repoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("runtime.Caller failed for env paths")
	}

	root := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("module root %s: %w", root, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("module root is not a directory: %s", root)
	}

	return root, nil
}
