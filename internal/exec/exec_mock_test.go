package exec

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	if v := os.Getenv(MockEnvVar); v != "" {
		os.Exit(ExecuteMockData(v))
		return
	}

	os.Exit(m.Run())
}
