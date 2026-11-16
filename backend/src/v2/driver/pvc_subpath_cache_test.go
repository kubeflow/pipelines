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
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	clientmanager "github.com/kubeflow/pipelines/backend/src/v2/client_manager"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/common"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	k8score "k8s.io/api/core/v1"
)

// cacheStoreAPI behaves like the API server cache index: FindCachedTask
// returns a prior successful task only when its fingerprint was stored.
type cacheStoreAPI struct {
	kfpapi.API
	store   map[string]*apiv2.PipelineTask
	lookups []string
}

func (c *cacheStoreAPI) FindCachedTask(
	_ context.Context,
	req *apiv2.FindCachedTaskRequest,
) (*apiv2.FindCachedTaskResponse, error) {
	c.lookups = append(c.lookups, req.GetCacheFingerprint())
	return &apiv2.FindCachedTaskResponse{Task: c.store[req.GetCacheFingerprint()]}, nil
}

type pvcRun struct {
	fingerprint string // "" when no cache lookup happened
	cached      bool
	mounts      []k8score.VolumeMount
}

func pvcMount(pvc, mountPath, subPath string) *kubernetesplatform.PvcMount {
	return &kubernetesplatform.PvcMount{
		PvcNameParameter: common.InputParamConstant(pvc),
		MountPath:        mountPath,
		SubPath:          subPath,
	}
}

// runPVCTask drives the real Container() for an otherwise identical task with
// the given PVC mounts, against a shared cache store.
func runPVCTask(
	t *testing.T,
	store map[string]*apiv2.PipelineTask,
	enableCache, cacheDisabled bool,
	mounts ...*kubernetesplatform.PvcMount,
) pvcRun {
	t.Helper()
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/cache_test.yaml")
	require.NoError(t, tc.Push("create-dataset"))
	defer func() {
		_, ok := tc.Pop()
		require.True(t, ok)
	}()

	opts := tc.setupContainerOptions(tc.RootTask, tc.GetLast().GetTaskSpec(),
		&kubernetesplatform.KubernetesExecutorConfig{PvcMount: mounts})
	opts.Task = proto.Clone(tc.GetLast().GetTaskSpec()).(*pipelinespec.PipelineTaskSpec)
	opts.Task.CachingOptions = &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: enableCache}
	opts.CacheDisabled = cacheDisabled

	api := &cacheStoreAPI{API: tc.MockAPI, store: store}
	tc.ClientManager = clientmanager.NewFakeClientManager(tc.ClientManager.K8sClient(), api)

	execution, err := Container(context.Background(), opts, tc.ClientManager)
	require.NoError(t, err)
	require.NotNil(t, execution)

	run := pvcRun{cached: execution.Cached != nil && *execution.Cached}
	require.LessOrEqual(t, len(api.lookups), 1)
	if len(api.lookups) == 1 {
		run.fingerprint = api.lookups[0]
	}
	if execution.PodSpecPatch != "" {
		var pod k8score.PodSpec
		require.NoError(t, json.Unmarshal([]byte(execution.PodSpecPatch), &pod))
		require.NotEmpty(t, pod.Containers)
		run.mounts = pod.Containers[0].VolumeMounts
	}
	return run
}

func cachedTaskFor(name string) *apiv2.PipelineTask {
	return &apiv2.PipelineTask{
		TaskId:  "prior-" + name,
		RunId:   "prior-run",
		Name:    "create-dataset",
		State:   apiv2.PipelineTask_SUCCEEDED,
		Outputs: &apiv2.PipelineTask_InputOutputs{Artifacts: []*apiv2.PipelineTask_InputOutputs_IOArtifact{}},
	}
}

// TestContainer_PVCSubPathDoesNotReuseCacheFromOtherSubPath is the end-to-end
// reproducer: a task that mounted data:experiment-a must not satisfy the cache
// lookup of an otherwise identical task that mounts data:experiment-b.
func TestContainer_PVCSubPathDoesNotReuseCacheFromOtherSubPath(t *testing.T) {
	store := map[string]*apiv2.PipelineTask{}

	first := runPVCTask(t, store, true, false, pvcMount("data", "/data", "experiment-a"))
	require.NotEmpty(t, first.fingerprint)
	require.False(t, first.cached)
	store[first.fingerprint] = cachedTaskFor("experiment-a")

	second := runPVCTask(t, store, true, false, pvcMount("data", "/data", "experiment-b"))
	t.Logf("fingerprint experiment-a=%s experiment-b=%s cached=%v", first.fingerprint, second.fingerprint, second.cached)

	require.False(t, second.cached,
		"task mounting data:experiment-b was served the cached result of data:experiment-a")
	assert.NotEqual(t, first.fingerprint, second.fingerprint)
	require.Len(t, second.mounts, 1)
	assert.Equal(t, "experiment-b", second.mounts[0].SubPath, "the pod must still receive the subPath")
}

func TestContainer_PVCMountCacheIdentity(t *testing.T) {
	fp := func(mounts ...*kubernetesplatform.PvcMount) string {
		r := runPVCTask(t, map[string]*apiv2.PipelineTask{}, true, false, mounts...)
		require.NotEmpty(t, r.fingerprint)
		return r.fingerprint
	}

	t.Run("same PVC and same subPath share a fingerprint", func(t *testing.T) {
		assert.Equal(t, fp(pvcMount("data", "/data", "experiment-a")), fp(pvcMount("data", "/data", "experiment-a")))
	})
	t.Run("same PVC with different subPaths differ", func(t *testing.T) {
		assert.NotEqual(t, fp(pvcMount("data", "/data", "experiment-a")), fp(pvcMount("data", "/data", "experiment-b")))
	})
	t.Run("different PVCs differ", func(t *testing.T) {
		assert.NotEqual(t, fp(pvcMount("data", "/data", "")), fp(pvcMount("other", "/data", "")))
		assert.NotEqual(t, fp(pvcMount("data", "/data", "experiment-a")), fp(pvcMount("other", "/data", "experiment-a")))
	})
	t.Run("subPath differs from PVC root", func(t *testing.T) {
		assert.NotEqual(t, fp(pvcMount("data", "/data", "")), fp(pvcMount("data", "/data", "experiment-a")))
	})
	t.Run("empty subPath equals omitted subPath", func(t *testing.T) {
		omitted := &kubernetesplatform.PvcMount{PvcNameParameter: common.InputParamConstant("data"), MountPath: "/data"}
		assert.Equal(t, fp(omitted), fp(pvcMount("data", "/data", "")))
	})
	t.Run("no subPath keeps the legacy name-only fingerprint", func(t *testing.T) {
		// Golden values produced by the name-only cache key before subPath was
		// part of the identity; tasks without subPath must keep their cache.
		assert.Equal(t, "7b03f3dfb29bc068bdff232894caab34477db494a0a85d6ac5c5f1801b862985",
			fp(pvcMount("data", "/data", "")))
		assert.Equal(t, "f5c398191b20a594a368f34c0ae195bcd8729f2f689c2e5b39179af3afed4c91",
			fp(pvcMount("data", "/data", ""), pvcMount("models", "/models", "")))
	})
	t.Run("multiple mounts: one changed subPath differs", func(t *testing.T) {
		assert.NotEqual(t,
			fp(pvcMount("data", "/data", "experiment-a"), pvcMount("models", "/models", "v1")),
			fp(pvcMount("data", "/data", "experiment-b"), pvcMount("models", "/models", "v1")))
	})
	t.Run("multiple mounts: order does not matter", func(t *testing.T) {
		assert.Equal(t,
			fp(pvcMount("data", "/data", "experiment-a"), pvcMount("models", "/models", "v1")),
			fp(pvcMount("models", "/models", "v1"), pvcMount("data", "/data", "experiment-a")))
	})
	t.Run("same PVC mounted twice with different subPaths differs from one subPath", func(t *testing.T) {
		assert.NotEqual(t,
			fp(pvcMount("data", "/a", "experiment-a"), pvcMount("data", "/b", "experiment-b")),
			fp(pvcMount("data", "/a", "experiment-a"), pvcMount("data", "/b", "experiment-a")))
	})
	t.Run("nested subPath is not confused with a sibling", func(t *testing.T) {
		assert.NotEqual(t, fp(pvcMount("data", "/data", "a/b")), fp(pvcMount("data", "/data", "a")))
	})
}

// A PVC name containing "/" is not a valid volume name and would alias
// "<name>/<subPath>" in the cache identity, so it must not reach a cache lookup.
func TestContainer_PVCNameWithSlashIsRejectedBeforeCacheLookup(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/cache_test.yaml")
	require.NoError(t, tc.Push("create-dataset"))
	defer func() {
		_, ok := tc.Pop()
		require.True(t, ok)
	}()
	opts := tc.setupContainerOptions(tc.RootTask, tc.GetLast().GetTaskSpec(),
		&kubernetesplatform.KubernetesExecutorConfig{PvcMount: []*kubernetesplatform.PvcMount{
			pvcMount("data/experiment-a", "/data", ""),
		}})
	opts.Task = proto.Clone(tc.GetLast().GetTaskSpec()).(*pipelinespec.PipelineTaskSpec)
	opts.Task.CachingOptions = &pipelinespec.PipelineTaskSpec_CachingOptions{EnableCache: true}
	api := &cacheStoreAPI{API: tc.MockAPI, store: map[string]*apiv2.PipelineTask{}}
	tc.ClientManager = clientmanager.NewFakeClientManager(tc.ClientManager.K8sClient(), api)

	_, err := Container(context.Background(), opts, tc.ClientManager)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid PVC name")
	assert.Empty(t, api.lookups)
}

func TestContainer_PVCSubPathCacheDisabled(t *testing.T) {
	for name, testCase := range map[string]struct{ enableCache, cacheDisabled bool }{
		"task cache disabled":   {enableCache: false},
		"driver cache disabled": {enableCache: true, cacheDisabled: true},
	} {
		t.Run(name, func(t *testing.T) {
			r := runPVCTask(t, map[string]*apiv2.PipelineTask{}, testCase.enableCache, testCase.cacheDisabled,
				pvcMount("data", "/data", "experiment-a"))
			assert.Empty(t, r.fingerprint, "no cache lookup expected")
			assert.False(t, r.cached)
			require.Len(t, r.mounts, 1)
			assert.Equal(t, "experiment-a", r.mounts[0].SubPath)
		})
	}
}
