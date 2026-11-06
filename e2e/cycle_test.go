package e2e_tests

import (
	"testing"

	"apinfra_e2e/env"
)

func TestE2eCycle(t *testing.T) {
	env.RunTheme(t, "cycle")
}
