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

package argocompiler_test

import (
	"strings"
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
)

// The pipeline runs a task with set_retry(20) inside dsl.ParallelFor
// (#13897). Each iteration must retry on its own, so the retry belongs to the
// task in the loop body rather than to the loop.
func TestCompileParallelForTaskRetriesPerIteration(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	pipelineSpec, platformSpec, err := util.LoadPipelineAndPlatformSpec("testdata/parallel_for_with_retry.yaml")
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

	loopBody, found := templates["comp-for-loop-1"]
	require.True(t, found, "missing loop body template")
	require.NotNil(t, loopBody.DAG)
	var retryingTask *wfapi.DAGTask
	for i := range loopBody.DAG.Tasks {
		if loopBody.DAG.Tasks[i].Name == "random-failure-op" {
			retryingTask = &loopBody.DAG.Tasks[i]
		}
	}
	require.NotNil(t, retryingTask, "task missing from the loop body")
	assert.True(t, strings.HasPrefix(retryingTask.Template, "retry-"),
		"task in the loop body should use a retry executor, got %q", retryingTask.Template)
	var maxCount string
	for _, parameter := range retryingTask.Arguments.Parameters {
		if parameter.Name == "retry-max-count" && parameter.Value != nil {
			maxCount = parameter.Value.String()
		}
	}
	assert.Equal(t, "20", maxCount)

	impl, found := templates["retry-system-container-impl"]
	require.True(t, found, "missing retry impl template")
	require.NotNil(t, impl.RetryStrategy)
	require.NotNil(t, impl.RetryStrategy.Limit)
	assert.Equal(t, "{{inputs.parameters.retry-max-count}}", impl.RetryStrategy.Limit.String())
}
