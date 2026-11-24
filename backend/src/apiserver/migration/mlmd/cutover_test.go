// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mlmd_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCutoverRequiresMaintenanceAndOrdersWriterFence(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "calls")
	programs := map[string]string{
		"curl": `#!/bin/bash
for ((i=1; i<=$#; i++)); do
  if [[ "${!i}" == "--output" ]]; then
    j=$((i+1)); echo 'down for maintenance' > "${!j}"
  fi
done
echo "$KFP_TEST_CURL_STATUS"
`,
		"kubectl": `#!/bin/bash
if [[ " $* " == *" get pods "* ]]; then exit 0; fi
echo "kubectl $*" >> "$KFP_TEST_LOG"
`,
		"mlmd-migrate": `#!/bin/bash
echo "mlmd-migrate $1" >> "$KFP_TEST_LOG"
if [[ "$1" == "--check-release-eligibility" && "$KFP_TEST_RELEASE_HELD" == "true" ]]; then
  echo 'production MLMD migration is disabled' >&2
  exit 1
fi
if [[ "$1" == "validate" && "$KFP_TEST_FAIL_VALIDATE" == "true" ]]; then exit 1; fi
echo '{"status":"ok"}'
`,
	}
	for name, source := range programs {
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(source), 0700))
	}
	run := func(code string, failValidation, releaseHeld string) (string, error) {
		command := exec.Command("bash", "cutover.sh")
		command.Env = append(os.Environ(),
			"PATH="+dir+":"+os.Getenv("PATH"),
			"KFP_TEST_CURL_STATUS="+code,
			"KFP_TEST_FAIL_VALIDATE="+failValidation,
			"KFP_TEST_RELEASE_HELD="+releaseHeld,
			"KFP_TEST_LOG="+logFile,
			"KFP_MLMD_MYSQL_DSN=fixture-dsn",
			"KFP_MLMD_MAINTENANCE_URL=http://fixture.invalid",
		)
		output, err := command.CombinedOutput()
		return string(output), err
	}
	output, err := run("503", "false", "true")
	require.Error(t, err)
	require.Contains(t, output, "production MLMD migration is disabled")
	log, err := os.ReadFile(logFile)
	require.NoError(t, err)
	require.Equal(t, "mlmd-migrate --check-release-eligibility\n", string(log))
	require.NoError(t, os.Truncate(logFile, 0))
	output, err = run("200", "false", "false")
	require.Error(t, err)
	require.Contains(t, output, "maintenance gate is not returning HTTP 503")
	log, err = os.ReadFile(logFile)
	require.NoError(t, err)
	require.Equal(t, "mlmd-migrate --check-release-eligibility\n", string(log))
	require.NoError(t, os.Truncate(logFile, 0))
	output, err = run("503", "true", "false")
	require.Error(t, err, output)
	log, err = os.ReadFile(logFile)
	require.NoError(t, err)
	require.Contains(t, string(log), "mlmd-migrate validate")
	require.NotContains(t, string(log), "scale deployment/ml-pipeline --replicas=0")
	require.NoError(t, os.Truncate(logFile, 0))
	output, err = run("503", "false", "false")
	require.NoError(t, err, output)
	log, err = os.ReadFile(logFile)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(log)), "\n")
	require.Equal(t, []string{
		"mlmd-migrate --check-release-eligibility",
		"kubectl -n kubeflow scale deployment/ml-pipeline-scheduledworkflow --replicas=0",
		"mlmd-migrate --drain-check",
		"mlmd-migrate validate",
		"kubectl -n kubeflow scale deployment/ml-pipeline --replicas=0",
		"mlmd-migrate --gate-only",
	}, lines)
}
