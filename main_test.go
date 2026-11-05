package e2e_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rogpeppe/go-internal/testscript"
)

func TestMain(m *testing.M) {
	// Empty map: resolve "apinfra" from PATH (local bin/ prepended in Setup).
	testscript.Main(m, map[string]func(){})
}

func TestE2e(t *testing.T) {
	testscript.Run(t, testscript.Params{
		Dir: "testdata",
		Setup: func(env *testscript.Env) error {
			env.Setenv("HOME", env.WorkDir)

			localBinDir, err := filepath.Abs("bin")
			if err != nil {
				return fmt.Errorf("resolve bin dir: %w", err)
			}

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
		},
		Cmds: map[string]func(*testscript.TestScript, bool, []string){
			dockerAssertRunning:       dockerAssertRunningCmd,
			dockerAssertNoPublicPorts: dockerAssertNoPublicPortsCmd,
			dockerAssertPortBinding:   dockerAssertPortBindingCmd,
			acmeRateLimitStartCmd:     acmeRateLimitStartCmdFn,
			acmeRateLimitStopCmd:      acmeRateLimitStopCmdFn,
			pebbleStartCmd:            pebbleStartCmdFn,
			pebbleStopCmd:             pebbleStopCmdFn,
		},
	})
}

func propagateEnv(env *testscript.Env, keys ...string) {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			env.Setenv(key, value)
		}
	}
}
