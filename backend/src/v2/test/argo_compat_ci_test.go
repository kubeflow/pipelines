// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestArgoCompatibilityCIDoesNotReferenceDeletedMLMD(t *testing.T) {
	repoRoot := findRepoRoot(t)
	actionPath := filepath.Join(repoRoot, ".github", "actions", "test-and-report", "action.yml")
	presubmitPath := filepath.Join(repoRoot, "backend", "src", "v2", "test", "presubmit-v2-go-test.sh")

	actionBytes, err := os.ReadFile(actionPath)
	if err != nil {
		t.Fatalf("read action.yml: %v", err)
	}
	action := string(actionBytes)
	if strings.Contains(action, "metadata-grpc-service") {
		t.Fatalf("test-and-report action still references deleted metadata-grpc-service")
	}
	if strings.Contains(action, "MLflow and MLMD both require localhost:8080") {
		t.Fatalf("test-and-report action still has obsolete MLflow/MLMD port conflict guard")
	}

	presubmitBytes, err := os.ReadFile(presubmitPath)
	if err != nil {
		t.Fatalf("read presubmit script: %v", err)
	}
	presubmit := string(presubmitBytes)
	if strings.Contains(presubmit, "metadata-grpc-service") {
		t.Fatalf("presubmit-v2-go-test.sh still port-forwards deleted metadata-grpc-service")
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not find repo root from %s", dir)
		}
		dir = parent
	}
}
