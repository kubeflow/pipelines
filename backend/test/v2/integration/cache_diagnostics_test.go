package integration

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// Run an intentionally failing subtest in a separate process so we can verify
// testify's Fatal/teardown ordering without failing the parent regression test.
func TestCacheDiagnosticsHook(t *testing.T) {
	if scenario := os.Getenv("KFP_CACHE_DIAGNOSTICS_PROBE"); scenario != "" {
		*runIntegrationTests = scenario != "disabled"
		suite.Run(t, &cacheDiagnosticsProbe{scenario: scenario})
		return
	}
	dir := t.TempDir()
	script := `#!/bin/sh
case "$*" in
  *"get pods"*) printf '{"items":[]}' ;;
  *) printf 'captured by kubectl: %s\n' "$*" ;;
esac
`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kubectl"), []byte(script), 0700))
	for _, scenario := range []string{"failed", "passed", "resource-namespace", "disabled"} {
		t.Run(scenario, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestCacheDiagnosticsHook$", "-test.v")
			cmd.Env = append(os.Environ(), "KFP_CACHE_DIAGNOSTICS_PROBE="+scenario, "PATH="+dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			output, err := cmd.CombinedOutput()
			text := string(output)
			if scenario == "passed" {
				require.NoError(t, err, text)
			} else {
				require.Error(t, err, text)
				require.Contains(t, text, "original assertion failure")
			}
			if scenario == "passed" || scenario == "disabled" {
				require.NotContains(t, text, "captured by kubectl")
				return
			}
			require.Contains(t, text, "Cache failure diagnostics:")
			require.Contains(t, text, "captured by kubectl")
			require.Contains(t, text, "cleanup sentinel")
			require.Less(t, strings.Index(text, "captured by kubectl"), strings.Index(text, "cleanup sentinel"))
			namespace := "control-plane"
			if scenario == "resource-namespace" {
				namespace = "user-namespace"
				require.NotContains(t, text, "--namespace=control-plane")
			}
			require.Contains(t, text, "--namespace="+namespace)
		})
	}
}

type cacheDiagnosticsProbe struct {
	suite.Suite
	scenario string
}

func (s *cacheDiagnosticsProbe) TestAssertion() {
	if s.scenario != "passed" {
		require.FailNow(s.T(), "original assertion failure")
	}
}

func (s *cacheDiagnosticsProbe) TearDownTest() {
	cache := &CacheTestSuite{namespace: "control-plane"}
	if s.scenario == "resource-namespace" {
		cache.resourceNamespace = "user-namespace"
	}
	cache.SetT(s.T())
	cache.TearDownTest()
}

func (s *cacheDiagnosticsProbe) TearDownSuite() {
	fmt.Println("cleanup sentinel")
}
