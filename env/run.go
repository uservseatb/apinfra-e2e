package env

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

// RunTheme runs testscript scenarios under testdata/<theme> (module root).
func RunTheme(t *testing.T, theme string) {
	t.Helper()

	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}

	testscript.Run(t, testscript.Params{
		Dir:   filepath.Join(root, "testdata", theme),
		Setup: setup(root),
		Cmds:  commands,
	})
}

// RunScenario runs a single testdata/<theme>/<name>.txt script.
// rel is theme/name or theme/name.txt (optional testdata/ prefix).
func RunScenario(t *testing.T, rel string) {
	t.Helper()

	root, err := repoRoot()
	if err != nil {
		t.Fatal(err)
	}

	file, err := resolveScenarioFile(root, rel)
	if err != nil {
		t.Fatal(err)
	}

	testscript.Run(t, testscript.Params{
		Files: []string{file},
		Setup: setup(root),
		Cmds:  commands,
	})
}

// resolveScenarioFile maps theme/name[.txt] to an absolute path under testdata/.
func resolveScenarioFile(root, rel string) (string, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	rel = strings.TrimPrefix(rel, "./")
	rel = strings.TrimPrefix(rel, "testdata/")
	if rel == "" {
		return "", fmt.Errorf("empty scenario path")
	}
	if strings.Contains(rel, "..") {
		return "", fmt.Errorf("invalid scenario path: %q", rel)
	}
	if !strings.HasSuffix(rel, ".txt") {
		rel += ".txt"
	}
	if strings.Count(rel, "/") != 1 {
		return "", fmt.Errorf("scenario path must be theme/name, got %q", strings.TrimSuffix(rel, ".txt"))
	}

	abs := filepath.Clean(filepath.Join(root, "testdata", filepath.FromSlash(rel)))
	base := filepath.Join(root, "testdata") + string(os.PathSeparator)
	if !strings.HasPrefix(abs, base) {
		return "", fmt.Errorf("scenario path escapes testdata: %q", rel)
	}
	if _, err := os.Stat(abs); err != nil {
		return "", fmt.Errorf("scenario %s: %w", strings.TrimSuffix(rel, ".txt"), err)
	}
	return abs, nil
}

func setup(root string) func(env *testscript.Env) error {
	return func(env *testscript.Env) error {
		env.Setenv("HOME", env.WorkDir)

		localBinDir := filepath.Join(root, "bin")
		binPath := filepath.Join(localBinDir, "apinfra")
		if _, err := os.Stat(binPath); err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf(
					"apinfra binary not found at %s — place the compiled binary there before running e2e tests",
					binPath,
				)
			}
			return fmt.Errorf("stat apinfra binary: %w", err)
		}

		propagateEnv(env,
			"DOCKER_HOST",
			"DOCKER_CONTEXT",
			"DOCKER_CERT_PATH",
			"DOCKER_TLS_VERIFY",
			"DOCKER_API_VERSION",
			"PATH",
		)

		prependGopathBin(env)
		// Local bin/apinfra must win over GOPATH/bin and the host PATH.
		env.Setenv("PATH", localBinDir+string(os.PathListSeparator)+env.Getenv("PATH"))

		return setupPostinstHome(env.WorkDir)
	}
}

var commands = map[string]func(*testscript.TestScript, bool, []string){
	dockerAssertRunning:       dockerAssertRunningCmd,
	dockerAssertNoPublicPorts: dockerAssertNoPublicPortsCmd,
	dockerAssertPortBinding:   dockerAssertPortBindingCmd,
	acmeRateLimitStartCmd:     acmeRateLimitStartCmdFn,
	acmeRateLimitStopCmd:      acmeRateLimitStopCmdFn,
	pebbleStartCmd:            pebbleStartCmdFn,
	pebbleStopCmd:             pebbleStopCmdFn,
	gitAuthFailStartCmd:       gitAuthFailStartCmdFn,
	gitAuthFailStopCmd:        gitAuthFailStopCmdFn,
}

func propagateEnv(env *testscript.Env, keys ...string) {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			env.Setenv(key, value)
		}
	}
}
