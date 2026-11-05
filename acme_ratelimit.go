package e2e_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
)

const (
	acmeRateLimitStartCmd = "acme-ratelimit-start"
	acmeRateLimitStopCmd  = "acme-ratelimit-stop"

	// Fixed listen addr so e2e .apinfra can point certs.letsencrypt.* at this stub.
	acmeRateLimitListenAddr = "127.0.0.1:18080"
	acmeRateLimitDirectory  = "https://localhost:18080/directory"

	rateLimitedProblemType = "urn:ietf:params:acme:error:rateLimited"
	replayNonceHeader      = "Replay-Nonce"
	e2eReplayNonce         = "e2e-ratelimit-nonce"
)

//nolint:gochecknoglobals // shared across testscript cmds in one e2e process
var (
	rateLimitMu     sync.Mutex
	rateLimitServer *http.Server
	rateLimitURL    string
)

func acmeRateLimitStartCmdFn(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", acmeRateLimitStartCmd)
	}

	if len(args) != 0 {
		ts.Fatalf("usage: %s", acmeRateLimitStartCmd)
	}

	url, err := startRateLimitACMEServer(acmeRateLimitListenAddr)
	if err != nil {
		ts.Fatalf("%s: %v", acmeRateLimitStartCmd, err)
	}

	// Same minica trust as pebble-start so lego accepts the stub's TLS cert.
	caPath, err := rateLimitCAPath()
	if err != nil {
		ts.Fatalf("%s: %v", acmeRateLimitStartCmd, err)
	}

	ts.Setenv("LEGO_CA_CERTIFICATES", caPath)
	ts.Setenv("LEGO_CA_SERVER_NAME", "localhost")
	ts.Logf("%s listening at %s", acmeRateLimitStartCmd, url)
}

func rateLimitCAPath() (string, error) {
	fixturesDir, err := e2eFixturesDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(fixturesDir, "pebble.minica.pem"), nil
}

func acmeRateLimitStopCmdFn(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", acmeRateLimitStopCmd)
	}

	if len(args) != 0 {
		ts.Fatalf("usage: %s", acmeRateLimitStopCmd)
	}

	err := stopRateLimitACMEServer()
	if err != nil {
		ts.Fatalf("%s: %v", acmeRateLimitStopCmd, err)
	}
}

func startRateLimitACMEServer(addr string) (string, error) {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	// Reuse a stub left running by a failed sibling scenario (fixed listen addr).
	if rateLimitServer != nil {
		return rateLimitURL, nil
	}

	fixturesDir, err := e2eFixturesDir()
	if err != nil {
		return "", err
	}

	certFile := filepath.Join(fixturesDir, "certs", "localhost", "cert.pem")
	keyFile := filepath.Join(fixturesDir, "certs", "localhost", "key.pem")

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return "", fmt.Errorf("listen %s: %w", addr, err)
	}

	// Directory links use https://localhost so the pebble fixture cert SAN matches.
	baseURL := "https://localhost:18080"
	mux := http.NewServeMux()
	mux.HandleFunc("/directory", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"newNonce":   baseURL + "/new-nonce",
			"newAccount": baseURL + "/new-account",
			"newOrder":   baseURL + "/new-order",
			"revokeCert": baseURL + "/revoke-cert",
			"keyChange":  baseURL + "/key-change",
			"meta": map[string]any{
				"termsOfService": "http://apinfra.e2e/tos",
			},
		})
	})
	mux.HandleFunc("/new-nonce", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set(replayNonceHeader, e2eReplayNonce)
		w.Header().Set("Cache-Control", "no-store")
		if r.Method == http.MethodHead || r.Method == http.MethodGet {
			w.WriteHeader(http.StatusNoContent)

			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})
	mux.HandleFunc("/new-account", writeRateLimited)
	mux.HandleFunc("/new-order", writeRateLimited)
	mux.HandleFunc("/revoke-cert", writeRateLimited)
	mux.HandleFunc("/key-change", writeRateLimited)

	srv := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	rateLimitServer = srv
	rateLimitURL = acmeRateLimitDirectory

	go func() {
		serveErr := srv.ServeTLS(listener, certFile, keyFile)
		if serveErr != nil && serveErr != http.ErrServerClosed {
			fmt.Printf("acme-ratelimit stub serve: %v\n", serveErr)
		}
	}()

	return rateLimitURL, nil
}

func writeRateLimited(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set(replayNonceHeader, e2eReplayNonce)
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(http.StatusTooManyRequests)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"type":   rateLimitedProblemType,
		"detail": "e2e forced rate limit",
		"status": http.StatusTooManyRequests,
	})
}

func stopRateLimitACMEServer() error {
	rateLimitMu.Lock()
	defer rateLimitMu.Unlock()

	if rateLimitServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := rateLimitServer.Shutdown(ctx)
	rateLimitServer = nil
	rateLimitURL = ""
	if err != nil {
		return fmt.Errorf("shutdown rate-limit ACME stub: %w", err)
	}

	return nil
}
