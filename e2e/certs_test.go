package e2e_tests

import (
	"testing"

	"apinfra_e2e/env"
)

func TestE2eCerts(t *testing.T) {
	env.RunTheme(t, "certs")
}
