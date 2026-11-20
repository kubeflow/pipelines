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

package mlmd

import (
	"encoding/json"
	"os"
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
)

func TestReconstructNativeFingerprintFromReleasedSource(t *testing.T) {
	data, err := os.ReadFile("testdata/engine/release_218_source_graph.json")
	require.NoError(t, err)
	var source struct {
		Runs []struct {
			ID                   string `json:"id"`
			PipelineName         string `json:"pipeline_name"`
			PipelineSpecManifest string `json:"pipeline_spec_manifest"`
		} `json:"runs"`
		Properties []struct {
			ExecutionID int    `json:"execution_id"`
			Name        string `json:"name"`
			StringValue string `json:"string_value"`
			IntValue    *int64 `json:"int_value"`
		} `json:"execution_properties"`
	}
	require.NoError(t, json.Unmarshal(data, &source))
	execution := &pb.Execution{CustomProperties: map[string]*pb.Value{}}
	for _, property := range source.Properties {
		if property.ExecutionID != 19 {
			continue
		}
		value := &pb.Value{Value: &pb.Value_StringValue{StringValue: property.StringValue}}
		if property.IntValue != nil {
			value.Value = &pb.Value_IntValue{IntValue: *property.IntValue}
		}
		execution.CustomProperties[property.Name] = value
	}
	var run cacheRun
	for _, sourceRun := range source.Runs {
		if sourceRun.ID == "2ec7ad3b-d821-4b1d-8355-190393a06ddc" {
			run = cacheRun{name: sourceRun.PipelineName, spec: sourceRun.PipelineSpecManifest}
		}
	}
	require.NotEmpty(t, run.spec)
	key, err := reconstructNativeFingerprint(execution, &run)
	require.NoError(t, err)
	require.Len(t, key, 64)
	require.NotEqual(t, propertyString(execution.CustomProperties, "cache_fingerprint"), key)

	state := pb.Execution_COMPLETE
	execution.Id = proto.Int64(19)
	execution.LastKnownState = &state
	const runID = "2ec7ad3b-d821-4b1d-8355-190393a06ddc"
	const cutoff int64 = 1793800000
	newPlan := func() Plan {
		return Plan{Tasks: []model.Task{{
			UUID: ExecutionUUID(19), RunUUID: runID,
			State: model.TaskStatus(apiv2.PipelineTask_SUCCEEDED),
		}}}
	}
	for _, scenario := range []struct {
		name    string
		created int64
		wantKey bool
	}{
		{"within window", cutoff - 1, true},
		{"at lower bound", cutoff - cacheBackfillWindowSeconds, true},
		{"older than window", cutoff - cacheBackfillWindowSeconds - 1, false},
		{"after cutoff", cutoff + 1, false},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			plan := newPlan()
			count := backfillNativeCacheKeys(&plan, &Snapshot{Executions: []*pb.Execution{execution}},
				map[int64]LegacyTask{19: {CreatedAtInSec: scenario.created}}, map[string]*cacheRun{runID: &run}, cutoff)
			if scenario.wantKey {
				require.Equal(t, 1, count)
				require.Equal(t, key, plan.Tasks[0].Fingerprint)
			} else {
				require.Zero(t, count)
				require.Empty(t, plan.Tasks[0].Fingerprint)
			}
			require.Equal(t, propertyString(execution.CustomProperties, "cache_fingerprint"),
				plan.Tasks[0].StatusMetadata["customProperties"].(map[string]any)["legacy_cache_fingerprint"])
			require.Empty(t, plan.Skipped)
		})
	}

	t.Run("mismatched source key is reported and skipped", func(t *testing.T) {
		bad := proto.Clone(execution).(*pb.Execution)
		bad.CustomProperties["cache_fingerprint"] = &pb.Value{Value: &pb.Value_StringValue{StringValue: "wrong"}}
		plan := newPlan()
		count := backfillNativeCacheKeys(&plan, &Snapshot{Executions: []*pb.Execution{bad}},
			map[int64]LegacyTask{19: {CreatedAtInSec: cutoff - 1}}, map[string]*cacheRun{runID: &run}, cutoff)
		require.Zero(t, count)
		require.Empty(t, plan.Tasks[0].Fingerprint)
		require.Equal(t, []Skip{{Kind: "cache_key", ID: 19, Reason: "reconstructed inputs do not reproduce source fingerprint"}}, plan.Skipped)
	})

	t.Run("missing saved pipeline spec is reported and skipped", func(t *testing.T) {
		plan := newPlan()
		count := backfillNativeCacheKeys(&plan, &Snapshot{Executions: []*pb.Execution{execution}},
			map[int64]LegacyTask{19: {CreatedAtInSec: cutoff - 1}},
			map[string]*cacheRun{runID: {name: "math-pipeline"}}, cutoff)
		require.Zero(t, count)
		require.Empty(t, plan.Tasks[0].Fingerprint)
		require.Equal(t, []Skip{{Kind: "cache_key", ID: 19, Reason: "saved pipeline spec is absent"}}, plan.Skipped)
	})

	t.Run("cached reuse keeps provenance but never becomes lookup target", func(t *testing.T) {
		cached := proto.Clone(execution).(*pb.Execution)
		state := pb.Execution_CACHED
		cached.LastKnownState = &state
		plan := newPlan()
		plan.Tasks[0].State = model.TaskStatus(apiv2.PipelineTask_CACHED)
		count := backfillNativeCacheKeys(&plan, &Snapshot{Executions: []*pb.Execution{cached}},
			map[int64]LegacyTask{19: {CreatedAtInSec: cutoff - 1}}, map[string]*cacheRun{runID: &run}, cutoff)
		require.Zero(t, count)
		require.Empty(t, plan.Tasks[0].Fingerprint)
		require.NotEmpty(t, plan.Tasks[0].StatusMetadata["customProperties"].(map[string]any)["legacy_cache_fingerprint"])
	})

	t.Run("cache row from another namespace cannot qualify a key", func(t *testing.T) {
		plan := newPlan()
		plan.Tasks[0].Namespace = "owner"
		count := backfillNativeCacheKeys(&plan, &Snapshot{Executions: []*pb.Execution{execution}},
			map[int64]LegacyTask{19: {RunUUID: runID, Namespace: "other", CreatedAtInSec: cutoff - 1}},
			map[string]*cacheRun{runID: &run}, cutoff)
		require.Zero(t, count)
		require.Empty(t, plan.Tasks[0].Fingerprint)
		require.Equal(t, []Skip{{Kind: "cache_key", ID: 19, Reason: "legacy cache row does not belong to the run namespace"}}, plan.Skipped)
	})

	t.Run("missing output prevents reuse", func(t *testing.T) {
		missingOutput := proto.Clone(execution).(*pb.Execution)
		delete(missingOutput.CustomProperties, "outputs")
		plan := newPlan()
		count := backfillNativeCacheKeys(&plan, &Snapshot{Executions: []*pb.Execution{missingOutput}},
			map[int64]LegacyTask{19: {CreatedAtInSec: cutoff - 1}}, map[string]*cacheRun{runID: &run}, cutoff)
		require.Zero(t, count)
		require.Empty(t, plan.Tasks[0].Fingerprint)
		require.Len(t, plan.Skipped, 1)
		require.Contains(t, plan.Skipped[0].Reason, "captured outputs are unavailable")
	})

	t.Run("legacy custom key cannot cross pipeline scope", func(t *testing.T) {
		task, _, err := cacheTaskSpec(run.parsed, "double")
		require.NoError(t, err)
		task.GetCachingOptions().CacheKey = "shared-key"
		_, err = reconstructNativeFingerprint(execution, &run)
		require.ErrorContains(t, err, "pipeline-scoped legacy lookup")
	})
}
