package e2e_tests

import (
	"flag"
	"testing"

	"github.com/uservseatb/apinfra-e2e/env"
)

// TestScenario runs one testdata script when a path is passed via go test -args.
//
//	task e2e -- -run TestScenario cycle/happy_path
//	task e2e -- cycle/happy_path
//
// With no -args (full suite), this test is a no-op.
func TestScenario(t *testing.T) {
	args := flag.Args()
	if len(args) == 0 {
		return
	}
	if len(args) != 1 {
		t.Fatalf("expected one scenario path (theme/name), got %q", args)
	}
	env.RunScenario(t, args[0])
}
