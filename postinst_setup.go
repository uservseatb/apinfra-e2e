package e2e_test

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"

	"github.com/rogpeppe/go-internal/testscript"
)

const (
	hostConfigName = ".apinfra"
	hostEnvName    = ".env"
	hostBinaryName = "apinfra"

	exampleConfigName = "apinfra.example"
	exampleEnvName    = "env.example"

	fileModePrivate    = 0o600
	fileModeExecutable = 0o755
	debExamplesRelPath = "fixtures/deb"
)

func prependGopathBin(env *testscript.Env) {
	bin, err := pebbleGopathBin()
	if err != nil {
		return
	}

	dir := filepath.Dir(bin)
	path := env.Getenv("PATH")
	if path == "" {
		path = os.Getenv("PATH")
	}

	env.Setenv("PATH", dir+string(os.PathListSeparator)+path)
}

// setupPostinstHome mirrors what the .deb package + postinst place under $HOME:
//
//	~/apinfra   (packaged binary placeholder)
//	~/.apinfra  (from apinfra.example, only if missing)
//	~/.env      (from env.example, only if missing)
//
// testscript unpacks the txtar archive into WorkDir before Setup runs, so a
// scenario can override host files by declaring them in the archive — same
// "keep existing" rule as postinst:
//
//	-- .apinfra --
//	namespace=staging
//	repo=https://example.invalid/config.git
//
//	-- .env --
//	DEMO_API_KEY=test-secret
func setupPostinstHome(homeDir string) error {
	examplesDir, err := debExamplesDir()
	if err != nil {
		return err
	}

	err = installFileIfMissing(
		filepath.Join(homeDir, hostConfigName),
		filepath.Join(examplesDir, exampleConfigName),
		fileModePrivate,
	)
	if err != nil {
		return fmt.Errorf("install %s: %w", hostConfigName, err)
	}

	err = installFileIfMissing(
		filepath.Join(homeDir, hostEnvName),
		filepath.Join(examplesDir, exampleEnvName),
		fileModePrivate,
	)
	if err != nil {
		return fmt.Errorf("install %s: %w", hostEnvName, err)
	}

	err = ensureHostBinary(filepath.Join(homeDir, hostBinaryName))
	if err != nil {
		return fmt.Errorf("install %s: %w", hostBinaryName, err)
	}

	return nil
}

func debExamplesDir() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("runtime.Caller failed for e2e test file")
	}

	dir := filepath.Join(filepath.Dir(file), debExamplesRelPath)

	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("deb examples dir %s: %w", dir, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("deb examples path is not a directory: %s", dir)
	}

	return dir, nil
}

func installFileIfMissing(dst, src string, mode os.FileMode) error {
	_, err := os.Stat(dst)
	if err == nil {
		return os.Chmod(dst, mode) // #nosec G302 -- match postinst secure_file (0600)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", dst, err)
	}

	in, err := os.Open(src) // #nosec G304 -- path from repo build/deb examples
	if err != nil {
		return fmt.Errorf("open example %s: %w", src, err)
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode) // #nosec G304 G302
	if err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}
	defer func() { _ = out.Close() }()

	_, err = io.Copy(out, in)
	if err != nil {
		return fmt.Errorf("copy %s -> %s: %w", src, dst, err)
	}

	return out.Close()
}

func ensureHostBinary(path string) error {
	_, err := os.Stat(path)
	if err == nil {
		return os.Chmod(path, fileModeExecutable) // #nosec G302 -- match packaged binary mode
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("stat %s: %w", path, err)
	}

	// Package ships the real binary under $HOME; e2e exec uses PATH -> bin/apinfra.
	// Place an empty executable so the home layout matches a fresh install.
	err = os.WriteFile(path, nil, fileModeExecutable) // #nosec G306 -- executable placeholder
	if err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	return nil
}
