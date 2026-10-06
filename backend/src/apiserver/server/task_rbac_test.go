// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	rbacv1 "k8s.io/api/rbac/v1"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
)

func TestTaskRuntimeAndEditorRoles(t *testing.T) {
	for _, tc := range []struct{ path, name string }{
		{"base/pipeline/pipeline-runner-role.yaml", "pipeline-runner"},
		{"base/installs/multi-user/view-edit-cluster-roles.yaml", "aggregate-to-kubeflow-pipelines-edit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f, err := os.Open(filepath.Join("../../../../manifests/kustomize", tc.path))
			require.NoError(t, err)
			defer f.Close()
			decoder := yamlutil.NewYAMLOrJSONDecoder(f, 4096)
			found := false
			for {
				var role rbacv1.ClusterRole
				err := decoder.Decode(&role)
				if err == io.EOF {
					break
				}
				require.NoError(t, err)
				if role.Name != tc.name {
					continue
				}
				for _, rule := range role.Rules {
					if len(rule.APIGroups) == 1 && rule.APIGroups[0] == "pipelines.kubeflow.org" {
						for _, verb := range rule.Verbs {
							if verb == "createTask" {
								require.Contains(t, rule.Resources, "runs")
								found = true
								if tc.name == "pipeline-runner" {
									require.ElementsMatch(t, []string{"list", "createTask"}, rule.Verbs)
								}
							}
						}
					}
				}
			}
			require.True(t, found, "task-writing permission must be deployed with endpoint authorization")
		})
	}
}
