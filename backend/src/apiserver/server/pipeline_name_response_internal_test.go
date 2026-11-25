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
	"context"
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runByIDRef builds a run that references its pipeline the legacy way, by ID.
func runByIDRef(pipelineID, pipelineVersionID string) *apiv2.Run {
	return &apiv2.Run{
		DisplayName: "run1",
		PipelineSource: &apiv2.Run_PipelineVersionReference{
			PipelineVersionReference: &apiv2.PipelineVersionReference{
				PipelineId:        pipelineID,
				PipelineVersionId: pipelineVersionID,
			},
		},
	}
}

// An ID-created run is the legacy case the names have to cover: nothing about
// its creation mentioned a name, so the names can only come from the read path.
func TestGetRun_ReportsNames_ForIdCreatedRun(t *testing.T) {
	manager := pipelineNameRefManager(t, nil)
	pipeline, versions := createNamedPipeline(t, manager, "", "training", "v1")
	server := createRunServer(manager)

	created, err := server.CreateRun(context.Background(), &apiv2.CreateRunRequest{
		Run: runByIDRef(pipeline.UUID, versions[0].UUID),
	})
	require.NoError(t, err)
	assert.Equal(t, "training", created.GetPipelineVersionReference().GetPipelineName())
	assert.Equal(t, "v1", created.GetPipelineVersionReference().GetPipelineVersionName())

	got, err := server.GetRun(context.Background(), &apiv2.GetRunRequest{RunId: created.GetRunId()})
	require.NoError(t, err)
	assert.Equal(t, "training", got.GetPipelineVersionReference().GetPipelineName())
	assert.Equal(t, "v1", got.GetPipelineVersionReference().GetPipelineVersionName())
}

// A name-created run reports the stored names, not whatever the request spelled.
func TestCreateRun_ReportsNames_ForNameCreatedRun(t *testing.T) {
	manager := pipelineNameRefManager(t, nil)
	_, versions := createNamedPipeline(t, manager, "", "training", "v1")

	created, err := createRunServer(manager).CreateRun(context.Background(), &apiv2.CreateRunRequest{
		Run: runByName("", pipelineNameRef("training", versions[0].Name)),
	})
	require.NoError(t, err)
	assert.Equal(t, "training", created.GetPipelineVersionReference().GetPipelineName())
	assert.Equal(t, "v1", created.GetPipelineVersionReference().GetPipelineVersionName())
}

// Several runs on one page share a pipeline, and each still reports its own names.
func TestListRuns_ReportsNames(t *testing.T) {
	manager := pipelineNameRefManager(t, nil)
	training, trainingVersions := createNamedPipeline(t, manager, "", "training", "v1")
	scoring, scoringVersions := createNamedPipeline(t, manager, "", "scoring", "v9")
	server := createRunServer(manager)

	for _, ref := range []*apiv2.Run{
		runByIDRef(training.UUID, trainingVersions[0].UUID),
		runByIDRef(training.UUID, trainingVersions[0].UUID),
		runByIDRef(scoring.UUID, scoringVersions[0].UUID),
	} {
		_, err := server.CreateRun(context.Background(), &apiv2.CreateRunRequest{Run: ref})
		require.NoError(t, err)
	}

	listed, err := server.ListRuns(context.Background(), &apiv2.ListRunsRequest{})
	require.NoError(t, err)
	require.Len(t, listed.GetRuns(), 3)
	namesByPipelineID := map[string]string{}
	for _, run := range listed.GetRuns() {
		ref := run.GetPipelineVersionReference()
		namesByPipelineID[ref.GetPipelineId()] = ref.GetPipelineName()
		assert.NotEmpty(t, ref.GetPipelineVersionName(), "run %s lost its version name", run.GetRunId())
	}
	assert.Equal(t, map[string]string{training.UUID: "training", scoring.UUID: "scoring"}, namesByPipelineID)
}

// Deleting a pipeline does not delete its runs, so the names go empty rather
// than failing the read.
func TestGetRun_EmptyNames_WhenPipelineDeleted(t *testing.T) {
	manager := pipelineNameRefManager(t, nil)
	pipeline, versions := createNamedPipeline(t, manager, "", "training", "v1")
	server := createRunServer(manager)

	created, err := server.CreateRun(context.Background(), &apiv2.CreateRunRequest{
		Run: runByIDRef(pipeline.UUID, versions[0].UUID),
	})
	require.NoError(t, err)
	require.NoError(t, manager.DeletePipeline(pipeline.UUID, true))

	got, err := server.GetRun(context.Background(), &apiv2.GetRunRequest{RunId: created.GetRunId()})
	require.NoError(t, err)
	assert.Empty(t, got.GetPipelineVersionReference().GetPipelineName())
	assert.Empty(t, got.GetPipelineVersionReference().GetPipelineVersionName())
	// The reference itself survives, so clients keep the IDs they had before.
	assert.Equal(t, pipeline.UUID, got.GetPipelineVersionReference().GetPipelineId())
}

func TestRecurringRun_ReportsNames(t *testing.T) {
	manager := pipelineNameRefManager(t, nil)
	pipeline, versions := createNamedPipeline(t, manager, "", "training", "v1")
	server := createJobServer(manager)

	created, err := server.CreateRecurringRun(context.Background(), &apiv2.CreateRecurringRunRequest{
		RecurringRun: &apiv2.RecurringRun{
			DisplayName:    "recurring1",
			Mode:           apiv2.RecurringRun_DISABLE,
			MaxConcurrency: 1,
			PipelineSource: &apiv2.RecurringRun_PipelineVersionReference{
				PipelineVersionReference: &apiv2.PipelineVersionReference{
					PipelineId:        pipeline.UUID,
					PipelineVersionId: versions[0].UUID,
				},
			},
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "training", created.GetPipelineVersionReference().GetPipelineName())
	assert.Equal(t, "v1", created.GetPipelineVersionReference().GetPipelineVersionName())

	got, err := server.GetRecurringRun(context.Background(), &apiv2.GetRecurringRunRequest{
		RecurringRunId: created.GetRecurringRunId(),
	})
	require.NoError(t, err)
	assert.Equal(t, "training", got.GetPipelineVersionReference().GetPipelineName())
	assert.Equal(t, "v1", got.GetPipelineVersionReference().GetPipelineVersionName())
}

// The names are informational: when the lookup fails, reads still succeed and
// report the IDs with empty names, rather than failing the whole page.
func TestReads_EmptyNames_WhenNameLookupFails(t *testing.T) {
	initEnvVars()
	clients, err := resource.NewFakeClientManager(util.NewFakeTimeForEpoch(), util.NewUUIDGenerator())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, clients.Close()) })
	manager := resource.NewResourceManager(clients, &resource.ResourceManagerOptions{CollectMetrics: false})
	pipeline, versions := createNamedPipeline(t, manager, "", "training", "v1")
	runServer := createRunServer(manager)
	jobServer := createJobServer(manager)

	createdRun, err := runServer.CreateRun(context.Background(), &apiv2.CreateRunRequest{
		Run: runByIDRef(pipeline.UUID, versions[0].UUID),
	})
	require.NoError(t, err)
	createdRecurringRun, err := jobServer.CreateRecurringRun(context.Background(), &apiv2.CreateRecurringRunRequest{
		RecurringRun: &apiv2.RecurringRun{
			DisplayName:    "recurring1",
			Mode:           apiv2.RecurringRun_DISABLE,
			MaxConcurrency: 1,
			PipelineSource: &apiv2.RecurringRun_PipelineVersionReference{
				PipelineVersionReference: &apiv2.PipelineVersionReference{
					PipelineId:        pipeline.UUID,
					PipelineVersionId: versions[0].UUID,
				},
			},
		},
	})
	require.NoError(t, err)

	// Make both name queries fail without touching the run and job tables.
	_, err = clients.DB().Exec(`ALTER TABLE pipelines RENAME TO pipelines_unavailable`)
	require.NoError(t, err)
	_, err = clients.DB().Exec(`ALTER TABLE pipeline_versions RENAME TO pipeline_versions_unavailable`)
	require.NoError(t, err)

	assertIdsWithoutNames := func(t *testing.T, ref *apiv2.PipelineVersionReference) {
		t.Helper()
		assert.Equal(t, pipeline.UUID, ref.GetPipelineId())
		assert.Equal(t, versions[0].UUID, ref.GetPipelineVersionId())
		assert.Empty(t, ref.GetPipelineName())
		assert.Empty(t, ref.GetPipelineVersionName())
	}

	gotRun, err := runServer.GetRun(context.Background(), &apiv2.GetRunRequest{RunId: createdRun.GetRunId()})
	require.NoError(t, err)
	assertIdsWithoutNames(t, gotRun.GetPipelineVersionReference())

	listedRuns, err := runServer.ListRuns(context.Background(), &apiv2.ListRunsRequest{})
	require.NoError(t, err)
	require.Len(t, listedRuns.GetRuns(), 1)
	assertIdsWithoutNames(t, listedRuns.GetRuns()[0].GetPipelineVersionReference())

	gotRecurringRun, err := jobServer.GetRecurringRun(context.Background(), &apiv2.GetRecurringRunRequest{
		RecurringRunId: createdRecurringRun.GetRecurringRunId(),
	})
	require.NoError(t, err)
	assertIdsWithoutNames(t, gotRecurringRun.GetPipelineVersionReference())

	listedRecurringRuns, err := jobServer.ListRecurringRuns(context.Background(), &apiv2.ListRecurringRunsRequest{})
	require.NoError(t, err)
	require.Len(t, listedRecurringRuns.GetRecurringRuns(), 1)
	assertIdsWithoutNames(t, listedRecurringRuns.GetRecurringRuns()[0].GetPipelineVersionReference())
}
