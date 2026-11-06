package env

import (
	"fmt"
	"os"
	"path/filepath"
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
}

func propagateEnv(env *testscript.Env, keys ...string) {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			env.Setenv(key, value)
		}
	}
}
