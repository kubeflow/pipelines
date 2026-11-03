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

package end2end

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"sigs.k8s.io/yaml"
)

func TestDriverRetryRuntimeFixture(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	for _, task := range []string{"restart-driver", "exhaust-driver"} {
		t.Run(task, func(t *testing.T) {
			encoded, err := yaml.YAMLToJSON([]byte(strings.ReplaceAll(driverRetryPipeline, "TARGET_TASK", task)))
			require.NoError(t, err)
			job := &pipelinespec.PipelineJob{PipelineSpec: &structpb.Struct{}}
			require.NoError(t, protojson.Unmarshal(encoded, job.PipelineSpec))
			workflow, err := argocompiler.Compile(job, nil, nil)
			require.NoError(t, err)
			require.NoError(t, util.NewWorkflow(workflow).Validate(true, false))
		})
	}
}

func TestDriverRetryCleanupStatusHTTPShape(t *testing.T) {
	RegisterTestingT(t)
	var task run_model.V2PipelineTask
	require.NoError(t, json.Unmarshal([]byte(`{"inputs":{"parameters":[{"parameter_key":"status","value":{"state":"FAILED","pipeline_task_name":"body"}}]}}`), &task))
	expectDriverRetryCleanupStatus(&task, "FAILED")
}
