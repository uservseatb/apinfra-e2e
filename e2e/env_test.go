package e2e_tests

import (
	"testing"

	"github.com/uservseatb/apinfra-e2e/env"
)

func TestE2eEnv(t *testing.T) {
	env.RunTheme(t, "env")
}
