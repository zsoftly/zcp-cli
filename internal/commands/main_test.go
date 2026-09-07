package commands

import (
	"os"
	"testing"
)

// TestMain redirects every network-facing default in this package's tests to
// unroutable/isolated targets before any test runs. buildClientAndPrinter
// resolves the API URL as flagURL > ZCP_API_URL env > profile APIURL >
// DefaultAPIURL (see internal/config.ActiveAPIURL), so a test that passes
// --api-url still works: the flag always wins over this env default. A test
// that forgets to stub the API, and forgets to pass --api-url, now fails fast
// against an unroutable local port instead of silently hitting whatever API
// the developer's environment or profile happens to point at.
//
// ZCP_BEARER_TOKEN is set so command-layer validation runs far enough to
// reach that unroutable request instead of stopping early on "no active
// profile configured and ZCP_BEARER_TOKEN not set". HOME and XDG_CONFIG_HOME
// point at a throwaway directory so config.Load() never reads a real
// developer profile that could otherwise supply its own api_url or token.
//
// ZCP_OUTPUT, ZCP_PROFILE, ZCP_PROJECT, ZCP_REGION, and ZCP_CLOUD_PROVIDER
// are cleared so a developer's shell environment can't change test
// behaviour: e.g. ZCP_OUTPUT=json in the shell would otherwise make every
// test expecting table output fail.
func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "zcp-cli-commands-test-*")
	if err != nil {
		panic(err)
	}

	os.Setenv("ZCP_API_URL", "http://127.0.0.1:9")
	os.Setenv("ZCP_BEARER_TOKEN", "test-token")
	os.Setenv("HOME", dir)
	os.Setenv("XDG_CONFIG_HOME", dir)
	os.Setenv("APPDATA", dir)
	os.Unsetenv("ZCP_OUTPUT")
	os.Unsetenv("ZCP_PROFILE")
	os.Unsetenv("ZCP_PROJECT")
	os.Unsetenv("ZCP_REGION")
	os.Unsetenv("ZCP_CLOUD_PROVIDER")

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}
