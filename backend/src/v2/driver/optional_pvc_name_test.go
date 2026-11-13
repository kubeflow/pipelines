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

package driver

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
)

// Compiled from a pipeline whose mount_pvc names come from Optional[str] = None inputs,
// a required str input and a nested pipeline. Task keys below refer to that file.
const optionalPVCNamePipeline = "test_data/optional_pvc_name.yaml"

const (
	optionalPVCCachedTask   = "noop"   // caching on: the cache fingerprint resolves PVC names first
	optionalPVCUncachedTask = "noop-2" // caching off: only the pod spec patch resolves PVC names
	requiredPVCTask         = "noop-3"
	optionalSecretTask      = "noop-4"
	nestedPVCDag            = "nested"
	nestedPVCTask           = "noop"
)

func optionalPVCRuntimeConfig(params map[string]*structpb.Value) *pipelinespec.PipelineJob_RuntimeConfig {
	values := map[string]*structpb.Value{"required_pvc_name": structpb.NewStringValue("required-pvc")}
	for k, v := range params {
		values[k] = v
	}
	return &pipelinespec.PipelineJob_RuntimeConfig{ParameterValues: values}
}

// runOptionalPVCTask runs the container driver for one task and returns its pod spec patch and task.
func runOptionalPVCTask(
	t *testing.T,
	tc *TestContext,
	parent *apiv2.PipelineTask,
	taskName string,
	editConfig func(*kubernetesplatform.KubernetesExecutorConfig),
) (*k8score.PodSpec, *apiv2.PipelineTask, error) {
	t.Helper()
	require.NoError(t, tc.Push(taskName))
	defer tc.Pop()
	k8sCfg, err := util.LoadKubernetesExecutorConfig(tc.GetLast().GetComponentSpec(), tc.PlatformSpec)
	require.NoError(t, err)
	if editConfig != nil {
		editConfig(k8sCfg)
	}
	opts := tc.setupContainerOptions(parent, tc.GetLast().GetTaskSpec(), k8sCfg)
	opts.CacheDisabled = !tc.GetLast().GetTaskSpec().GetCachingOptions().GetEnableCache()

	execution, err := Container(context.Background(), opts, tc.ClientManager)
	if err != nil {
		return nil, nil, err
	}
	task, err := tc.ClientManager.KFPAPIClient().GetTask(context.Background(), &apiv2.GetTaskRequest{TaskId: execution.TaskID})
	require.NoError(t, err)
	podSpec := &k8score.PodSpec{}
	require.NoError(t, json.Unmarshal([]byte(execution.PodSpecPatch), podSpec))
	return podSpec, task, nil
}

func pvcClaimNames(podSpec *k8score.PodSpec) []string {
	var names []string
	for _, volume := range podSpec.Volumes {
		if volume.PersistentVolumeClaim != nil {
			names = append(names, volume.PersistentVolumeClaim.ClaimName)
		}
	}
	return names
}

func TestOptionalPVCName(t *testing.T) {
	null := structpb.NewNullValue()
	tests := []struct {
		name       string
		params     map[string]*structpb.Value
		task       string
		editConfig func(*kubernetesplatform.KubernetesExecutorConfig)
		wantPVCs   []string
		wantErr    string
	}{
		// The Argo compiler stores an omitted optional input without a default as an explicit null,
		// so both forms reach the driver in practice.
		{name: "optional omitted, caching off", task: optionalPVCUncachedTask},
		{name: "optional omitted, caching on", task: optionalPVCCachedTask},
		{name: "optional null, caching off", params: map[string]*structpb.Value{"optional_pvc_name": null}, task: optionalPVCUncachedTask},
		{name: "optional null, caching on", params: map[string]*structpb.Value{"optional_pvc_name": null}, task: optionalPVCCachedTask},
		{
			name:     "optional supplied, caching off",
			params:   map[string]*structpb.Value{"optional_pvc_name": structpb.NewStringValue("data-pvc")},
			task:     optionalPVCUncachedTask,
			wantPVCs: []string{"data-pvc"},
		},
		{
			name:     "optional supplied, caching on",
			params:   map[string]*structpb.Value{"optional_pvc_name": structpb.NewStringValue("data-pvc")},
			task:     optionalPVCCachedTask,
			wantPVCs: []string{"data-pvc"},
		},
		{name: "required supplied", task: requiredPVCTask, wantPVCs: []string{"required-pvc"}},
		{
			name:    "required null still fails",
			params:  map[string]*structpb.Value{"required_pvc_name": null},
			task:    requiredPVCTask,
			wantErr: `pvc name input "required_pvc_name" resolved to null but is not an optional input`,
		},
		{
			name: "input missing from the parent DAG still fails",
			task: optionalPVCUncachedTask,
			editConfig: func(cfg *kubernetesplatform.KubernetesExecutorConfig) {
				cfg.PvcMount[0].PvcNameParameter = common.InputParamComponent("not_a_pipeline_input")
			},
			wantErr: `pvc name input "not_a_pipeline_input" resolved to null but is not an optional input`,
		},
		{
			name:    "empty name still fails",
			params:  map[string]*structpb.Value{"optional_pvc_name": structpb.NewStringValue("")},
			task:    optionalPVCUncachedTask,
			wantErr: "Expected a non-empty string",
		},
		{
			name:    "wrong type still fails",
			params:  map[string]*structpb.Value{"optional_pvc_name": structpb.NewNumberValue(3)},
			task:    optionalPVCUncachedTask,
			wantErr: "Expected a string",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTestContextWithRootExecuted(t, optionalPVCRuntimeConfig(tt.params), optionalPVCNamePipeline)
			podSpec, task, err := runOptionalPVCTask(t, tc, tc.RootTask, tt.task, tt.editConfig)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantPVCs, pvcClaimNames(podSpec))
			assert.Len(t, podSpec.Containers[0].VolumeMounts, len(tt.wantPVCs))
			if tt.task == optionalPVCCachedTask {
				assert.NotEmpty(t, task.GetCacheFingerprint())
			}
		})
	}
}

func TestOptionalPVCName_SecretComparison(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, optionalPVCRuntimeConfig(nil), optionalPVCNamePipeline)
	podSpec, _, err := runOptionalPVCTask(t, tc, tc.RootTask, optionalSecretTask, nil)
	require.NoError(t, err)
	assert.Empty(t, podSpec.Volumes)
}

// The DAG driver drops a null optional input when it resolves a nested pipeline's inputs,
// so the PVC task inside sees no value at all for child_pvc_name.
func TestOptionalPVCName_NestedPipeline(t *testing.T) {
	for name, tt := range map[string]struct {
		params   map[string]*structpb.Value
		wantPVCs []string
	}{
		"null":     {},
		"supplied": {params: map[string]*structpb.Value{"optional_pvc_name": structpb.NewStringValue("data-pvc")}, wantPVCs: []string{"data-pvc"}},
	} {
		t.Run(name, func(t *testing.T) {
			tc := NewTestContextWithRootExecuted(t, optionalPVCRuntimeConfig(tt.params), optionalPVCNamePipeline)
			_, nestedTask := tc.RunDagDriver(nestedPVCDag, tc.RootTask)
			podSpec, _, err := runOptionalPVCTask(t, tc, nestedTask, nestedPVCTask, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPVCs, pvcClaimNames(podSpec))
		})
	}
}

func TestOptionalPVCName_CacheFingerprint(t *testing.T) {
	fingerprint := func(params map[string]*structpb.Value) string {
		tc := NewTestContextWithRootExecuted(t, optionalPVCRuntimeConfig(params), optionalPVCNamePipeline)
		_, task, err := runOptionalPVCTask(t, tc, tc.RootTask, optionalPVCCachedTask, nil)
		require.NoError(t, err)
		return task.GetCacheFingerprint()
	}
	omitted := fingerprint(nil)
	explicitNull := fingerprint(map[string]*structpb.Value{"optional_pvc_name": structpb.NewNullValue()})
	supplied := fingerprint(map[string]*structpb.Value{"optional_pvc_name": structpb.NewStringValue("data-pvc")})
	otherPVC := fingerprint(map[string]*structpb.Value{"optional_pvc_name": structpb.NewStringValue("other-pvc")})

	assert.Equal(t, omitted, explicitNull)
	assert.NotEqual(t, omitted, supplied, "a skipped mount must not share a cache key with a mounted PVC")
	assert.NotEqual(t, supplied, otherPVC)
}
