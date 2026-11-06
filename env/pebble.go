package env

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
)

const (
	pebbleStartCmd = "pebble-start"
	pebbleStopCmd  = "pebble-stop"

	pebbleListenURL     = "https://127.0.0.1:14000/dir"
	pebbleBinaryName    = "pebble"
	pebbleModuleInstall = "github.com/letsencrypt/pebble/v2/cmd/pebble@v2.10.1"
	pebbleReadyAttempts = 60
	pebbleReadyDelay    = 500 * time.Millisecond
)

//nolint:gochecknoglobals // shared across testscript cmds in one e2e process
var (
	pebbleMu      sync.Mutex
	pebbleProcess *exec.Cmd
)

func pebbleStartCmdFn(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", pebbleStartCmd)
	}

	if len(args) != 0 {
		ts.Fatalf("usage: %s", pebbleStartCmd)
	}

	caPath, err := startPebble()
	if err != nil {
		ts.Fatalf("%s: %v", pebbleStartCmd, err)
	}

	// Trust pebble/minica for HTTPS ACME (Pebble + rate-limit stub share fixtures).
	ts.Setenv("LEGO_CA_CERTIFICATES", caPath)
	ts.Setenv("LEGO_CA_SERVER_NAME", "localhost")
	ts.Logf("%s ready at %s", pebbleStartCmd, pebbleListenURL)
}

func pebbleStopCmdFn(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", pebbleStopCmd)
	}

	if len(args) != 0 {
		ts.Fatalf("usage: %s", pebbleStopCmd)
	}

	err := stopPebble()
	if err != nil {
		ts.Fatalf("%s: %v", pebbleStopCmd, err)
	}
}

func startPebble() (string, error) {
	pebbleMu.Lock()
	defer pebbleMu.Unlock()

	if pebbleProcess != nil && pebbleProcess.Process != nil {
		return "", fmt.Errorf("pebble already running (pid %d)", pebbleProcess.Process.Pid)
	}

	fixturesDir, err := e2eFixturesDir()
	if err != nil {
		return "", err
	}

	caPath := filepath.Join(fixturesDir, "pebble.minica.pem")

	pebbleBin, err := lookOrInstallPebble()
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(fixturesDir, "pebble-eab.json")
	cmd := exec.Command(pebbleBin, "-config", configPath) // #nosec G204 -- fixed fixture path + local binary
	cmd.Dir = fixturesDir
	cmd.Env = append(os.Environ(),
		"PEBBLE_VA_ALWAYS_VALID=1",
		"PEBBLE_VA_NOSLEEP=1",
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Start()
	if err != nil {
		return "", fmt.Errorf("start pebble: %w", err)
	}

	pebbleProcess = cmd

	err = waitPebbleReady()
	if err != nil {
		_ = stopPebbleLocked()

		return "", err
	}

	return caPath, nil
}

func stopPebble() error {
	pebbleMu.Lock()
	defer pebbleMu.Unlock()

	return stopPebbleLocked()
}

func stopPebbleLocked() error {
	if pebbleProcess == nil || pebbleProcess.Process == nil {
		pebbleProcess = nil

		return nil
	}

	_ = pebbleProcess.Process.Kill()
	_ = pebbleProcess.Wait()
	pebbleProcess = nil

	return nil
}

func lookOrInstallPebble() (string, error) {
	path, err := exec.LookPath(pebbleBinaryName)
	if err == nil {
		return path, nil
	}

	install := exec.Command("go", "install", pebbleModuleInstall) // #nosec G204 -- pinned module path
	install.Stdout = os.Stdout
	install.Stderr = os.Stderr
	err = install.Run()
	if err != nil {
		return "", fmt.Errorf("go install %s: %w", pebbleModuleInstall, err)
	}

	path, err = exec.LookPath(pebbleBinaryName)
	if err == nil {
		return path, nil
	}

	gopathBin, err := pebbleGopathBin()
	if err != nil {
		return "", err
	}

	if st, statErr := os.Stat(gopathBin); statErr == nil && !st.IsDir() {
		return gopathBin, nil
	}

	return "", fmt.Errorf("pebble binary not found on PATH after install (also checked %s)", gopathBin)
}

func pebbleGopathBin() (string, error) {
	out, err := exec.Command("go", "env", "GOPATH").Output()
	if err != nil {
		return "", fmt.Errorf("go env GOPATH: %w", err)
	}

	gopath := filepath.Clean(strings.TrimSpace(string(out)))
	if gopath == "" || gopath == "." {
		return "", errors.New("empty GOPATH")
	}

	return filepath.Join(gopath, "bin", pebbleBinaryName), nil
}

func waitPebbleReady() error {
	client := &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 -- local pebble e2e only
		},
	}

	var lastErr error
	for range pebbleReadyAttempts {
		resp, err := client.Get(pebbleListenURL) // #nosec G107 -- fixed local pebble URL
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}

			lastErr = fmt.Errorf("status %s", resp.Status)
		} else {
			lastErr = err
		}

		time.Sleep(pebbleReadyDelay)
	}

	return fmt.Errorf("pebble not ready at %s: %w", pebbleListenURL, lastErr)
}

func e2eFixturesDir() (string, error) {
	root, err := repoRoot()
	if err != nil {
		return "", err
	}

	dir := filepath.Join(root, "fixtures")
	info, err := os.Stat(dir)
	if err != nil {
		return "", fmt.Errorf("fixtures dir %s: %w", dir, err)
	}

	if !info.IsDir() {
		return "", fmt.Errorf("fixtures path is not a directory: %s", dir)
	}

	return dir, nil
}
