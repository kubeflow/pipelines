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

package argocompiler_test

import (
	"path/filepath"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
)

func TestCompileRuntimeTerminationMessagePolicy(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	tests := []struct {
		name              string
		pipelineFile      string
		executorTemplates []string
		importerTemplate  string
	}{
		{
			name:              "container",
			pipelineFile:      "hello_world.yaml",
			executorTemplates: []string{"system-container-impl"},
		},
		{
			name:              "retry",
			pipelineFile:      "critical/pipeline_with_retry.yaml",
			executorTemplates: []string{"retry-system-container-impl"},
		},
		{
			name:              "explicit retry policy",
			pipelineFile:      "pipeline_with_retry_policy.yaml",
			executorTemplates: []string{"retry-system-container-impl-onerror"},
		},
		{
			name:         "pod metadata",
			pipelineFile: "create_pod_metadata_complex.yaml",
			executorTemplates: []string{
				"system-container-impl",
				"metadata-1-2-system-container-impl",
				"metadata-2-0-system-container-impl",
			},
		},
		{
			name:              "importer",
			pipelineFile:      "pipeline_with_importer.yaml",
			executorTemplates: []string{"system-container-impl"},
			importerTemplate:  "system-importer",
		},
		{
			name:              "workspace importer",
			pipelineFile:      "critical/pipeline_with_importer_workspace.yaml",
			executorTemplates: []string{"system-container-impl"},
			importerTemplate:  "system-importer-workspace",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pipelineSpec, platformSpec, err := util.LoadPipelineAndPlatformSpec(filepath.Join(
				"../../../../../test_data/sdk_compiled_pipelines/valid", test.pipelineFile))
			require.NoError(t, err)
			pipelineJSON, err := protojson.Marshal(pipelineSpec)
			require.NoError(t, err)
			pipelineStruct := &structpb.Struct{}
			require.NoError(t, protojson.Unmarshal(pipelineJSON, pipelineStruct))
			workflow, err := argocompiler.Compile(
				&pipelinespec.PipelineJob{PipelineSpec: pipelineStruct},
				platformSpec.GetPlatforms()["kubernetes"],
				&argocompiler.Options{},
			)
			require.NoError(t, err)

			templates := make(map[string]wfapi.Template, len(workflow.Spec.Templates))
			for _, template := range workflow.Spec.Templates {
				templates[template.Name] = template
			}
			for _, name := range []string{"system-container-driver", "system-dag-driver"} {
				template, found := templates[name]
				require.True(t, found, "missing driver template %q", name)
				require.NotNil(t, template.Plugin, "template %q must use the executor plugin", name)
				require.Nil(t, template.Container, "template %q must not create a driver pod", name)
			}
			containerTemplates := append([]string{}, test.executorTemplates...)
			if test.importerTemplate != "" {
				containerTemplates = append(containerTemplates, test.importerTemplate)
			}
			for _, name := range containerTemplates {
				template, found := templates[name]
				require.True(t, found, "missing container template %q", name)
				require.NotNil(t, template.Container, "template %q", name)
				assert.Equal(t, k8score.TerminationMessageFallbackToLogsOnError,
					template.Container.TerminationMessagePolicy, "template %q", name)
			}

			for _, name := range test.executorTemplates {
				var launcher *k8score.Container
				for _, initContainer := range templates[name].InitContainers {
					if initContainer.Name == "kfp-launcher" {
						launcher = &initContainer.Container
						break
					}
				}
				require.NotNil(t, launcher, "template %q must copy the launcher binary", name)
				assert.Equal(t, k8score.TerminationMessageFallbackToLogsOnError,
					launcher.TerminationMessagePolicy, "template %q launcher init container", name)
			}
		})
	}
}
