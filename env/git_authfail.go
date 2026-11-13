package env

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
)

const (
	gitAuthFailStartCmd = "git-authfail-start"
	gitAuthFailStopCmd  = "git-authfail-stop"

	// Fixed listen addr so e2e .apinfra can point repo= at this stub.
	gitAuthFailListenAddr = "127.0.0.1:18081"
	gitAuthFailBaseURL    = "http://127.0.0.1:18081"
)

//nolint:gochecknoglobals // shared across testscript cmds in one e2e process
var (
	gitAuthFailMu     sync.Mutex
	gitAuthFailServer *http.Server
)

func gitAuthFailStartCmdFn(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", gitAuthFailStartCmd)
	}

	if len(args) != 0 {
		ts.Fatalf("usage: %s", gitAuthFailStartCmd)
	}

	err := startGitAuthFailServer()
	if err != nil {
		ts.Fatalf("%s: %v", gitAuthFailStartCmd, err)
	}

	ts.Logf("%s listening at %s (HTTP 401)", gitAuthFailStartCmd, gitAuthFailBaseURL)
}

func gitAuthFailStopCmdFn(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", gitAuthFailStopCmd)
	}

	if len(args) != 0 {
		ts.Fatalf("usage: %s", gitAuthFailStopCmd)
	}

	err := stopGitAuthFailServer()
	if err != nil {
		ts.Fatalf("%s: %v", gitAuthFailStopCmd, err)
	}
}

func startGitAuthFailServer() error {
	gitAuthFailMu.Lock()
	defer gitAuthFailMu.Unlock()

	// Idempotent: a prior scenario may have left the stub up after a mid-script fail.
	if gitAuthFailServer != nil {
		return nil
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("WWW-Authenticate", `Basic realm="git"`)
		w.WriteHeader(http.StatusUnauthorized)
	})

	listener, err := net.Listen("tcp", gitAuthFailListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", gitAuthFailListenAddr, err)
	}

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	gitAuthFailServer = server

	go func() {
		serveErr := server.Serve(listener)
		if serveErr != nil && serveErr != http.ErrServerClosed {
			fmt.Printf("git-authfail stub serve: %v\n", serveErr)
		}
	}()

	return nil
}

func stopGitAuthFailServer() error {
	gitAuthFailMu.Lock()
	defer gitAuthFailMu.Unlock()

	if gitAuthFailServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := gitAuthFailServer.Shutdown(ctx)
	gitAuthFailServer = nil
	if err != nil {
		return fmt.Errorf("shutdown git auth-fail stub: %w", err)
	}

	return nil
}
