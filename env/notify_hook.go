package env

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rogpeppe/go-internal/testscript"
)

const (
	notifyHookStartCmd = "notify-hook-start"
	notifyHookStopCmd  = "notify-hook-stop"
	notifyHookFailCmd  = "notify-hook-fail"
	notifyHookOKCmd    = "notify-hook-ok"

	notifyHookListenAddr = "127.0.0.1:18100"
	notifyHookBaseURL    = "http://127.0.0.1:18100"
)

//nolint:gochecknoglobals // shared across testscript cmds in one e2e process
var (
	notifyHookMu         sync.Mutex
	notifyHookServer     *http.Server
	notifyHookFile       string
	notifyHookStatusCode = http.StatusOK
)

func notifyHookStartCmdFn(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", notifyHookStartCmd)
	}

	if len(args) != 0 {
		ts.Fatalf("usage: %s", notifyHookStartCmd)
	}

	path := filepath.Join(ts.Getenv("WORK"), "notify-hook.bodies")
	err := startNotifyHookServer(path)
	if err != nil {
		ts.Fatalf("%s: %v", notifyHookStartCmd, err)
	}

	ts.Logf("%s listening at %s/hook (bodies → %s)", notifyHookStartCmd, notifyHookBaseURL, path)
}

func notifyHookStopCmdFn(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", notifyHookStopCmd)
	}

	if len(args) != 0 {
		ts.Fatalf("usage: %s", notifyHookStopCmd)
	}

	err := stopNotifyHookServer()
	if err != nil {
		ts.Fatalf("%s: %v", notifyHookStopCmd, err)
	}
}

func notifyHookFailCmdFn(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", notifyHookFailCmd)
	}

	if len(args) != 0 {
		ts.Fatalf("usage: %s", notifyHookFailCmd)
	}

	setNotifyHookStatus(http.StatusInternalServerError)
	ts.Logf("%s → HTTP %d", notifyHookFailCmd, http.StatusInternalServerError)
}

func notifyHookOKCmdFn(ts *testscript.TestScript, neg bool, args []string) {
	if neg {
		ts.Fatalf("unsupported: ! %s", notifyHookOKCmd)
	}

	if len(args) != 0 {
		ts.Fatalf("usage: %s", notifyHookOKCmd)
	}

	setNotifyHookStatus(http.StatusOK)
	ts.Logf("%s → HTTP %d", notifyHookOKCmd, http.StatusOK)
}

func setNotifyHookStatus(code int) {
	notifyHookMu.Lock()
	defer notifyHookMu.Unlock()

	notifyHookStatusCode = code
}

func startNotifyHookServer(bodiesPath string) error {
	notifyHookMu.Lock()
	defer notifyHookMu.Unlock()

	if notifyHookServer != nil {
		return nil
	}

	err := os.WriteFile(bodiesPath, nil, 0o600)
	if err != nil {
		return fmt.Errorf("create bodies file: %w", err)
	}

	notifyHookFile = bodiesPath
	notifyHookStatusCode = http.StatusOK

	mux := http.NewServeMux()
	mux.HandleFunc("/hook", func(w http.ResponseWriter, r *http.Request) {
		body, readErr := io.ReadAll(r.Body)
		if readErr != nil {
			w.WriteHeader(http.StatusBadRequest)

			return
		}

		notifyHookMu.Lock()
		status := notifyHookStatusCode
		bodiesPath := notifyHookFile
		notifyHookMu.Unlock()

		if status < http.StatusOK || status >= http.StatusMultipleChoices {
			w.WriteHeader(status)

			return
		}

		notifyHookMu.Lock()
		f, openErr := os.OpenFile(bodiesPath, os.O_APPEND|os.O_WRONLY, 0o600)
		if openErr != nil {
			notifyHookMu.Unlock()
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		_, _ = f.Write(body)
		_, _ = f.Write([]byte("\n---\n"))
		_ = f.Close()
		notifyHookMu.Unlock()

		w.WriteHeader(status)
	})

	listener, err := net.Listen("tcp", notifyHookListenAddr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", notifyHookListenAddr, err)
	}

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	notifyHookServer = server

	go func() {
		serveErr := server.Serve(listener)
		if serveErr != nil && serveErr != http.ErrServerClosed {
			fmt.Printf("notify-hook stub serve: %v\n", serveErr)
		}
	}()

	return nil
}

func stopNotifyHookServer() error {
	notifyHookMu.Lock()
	defer notifyHookMu.Unlock()

	if notifyHookServer == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := notifyHookServer.Shutdown(ctx)
	notifyHookServer = nil
	notifyHookFile = ""
	notifyHookStatusCode = http.StatusOK
	if err != nil {
		return fmt.Errorf("shutdown notify-hook stub: %w", err)
	}

	return nil
}
