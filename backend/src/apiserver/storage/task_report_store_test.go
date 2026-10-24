// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"database/sql"
	"testing"
	"time"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTaskReportArtifactFailureRollsBackMetadataAndLinks(t *testing.T) {
	db, tasks, runtime, task := runtimeFixture(t)
	_, err := db.Exec("UPDATE run_details SET ExecutorType = 'docker' WHERE UUID = ?", task.RunUUID)
	require.NoError(t, err)
	require.NoError(t, runtime.Create(runtimeAttemptFixture(task, "attempt")))
	identity := &model.TaskReportIdentity{TaskUUID: task.UUID, AttemptUUID: "attempt"}
	artifacts := []*model.Artifact{
		{Namespace: task.Namespace, Name: "valid", Type: model.ArtifactType(apiv2.Artifact_Model)},
		{Namespace: task.Namespace, Name: "invalid", Type: model.ArtifactType(apiv2.Artifact_Model), Metadata: model.JSONData{"invalid": make(chan bool)}},
	}
	links := []*model.ArtifactTask{
		{TaskID: task.UUID, RunUUID: task.RunUUID, Type: model.IOType(apiv2.IOType_OUTPUT), ArtifactKey: "valid"},
		{TaskID: task.UUID, RunUUID: task.RunUUID, Type: model.IOType(apiv2.IOType_OUTPUT), ArtifactKey: "invalid"},
	}
	created, createdLinks, err := tasks.ReportArtifacts(identity, artifacts, links, false)
	require.Error(t, err)
	assert.Nil(t, created)
	assert.Nil(t, createdLinks)
	for _, table := range []string{"artifacts", "artifact_tasks"} {
		var count int
		require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM "+table).Scan(&count))
		assert.Zero(t, count)
	}
}

func TestTaskReportArgoReusePreservesExistingRandomLink(t *testing.T) {
	db, tasks, _, task := runtimeFixture(t)
	uri := "file:///shared/model"
	source := &model.Artifact{Namespace: task.Namespace, Name: "model", URI: &uri, Type: model.ArtifactType(apiv2.Artifact_Model)}
	link := &model.ArtifactTask{TaskID: task.UUID, RunUUID: task.RunUUID, Type: model.IOType(apiv2.IOType_OUTPUT), ArtifactKey: "model"}
	store := NewArtifactStore(db, tasks.time, tasks.uuid, tasks.dbDialect)
	existingArtifact, existingLink, err := store.CreateArtifactWithTask(source, link)
	require.NoError(t, err)
	created, links, err := tasks.ReportArtifacts(nil, []*model.Artifact{source}, []*model.ArtifactTask{link}, true)
	require.NoError(t, err)
	require.Len(t, created, 1)
	require.Len(t, links, 1)
	assert.Equal(t, existingArtifact.UUID, created[0].UUID)
	assert.Equal(t, existingLink.UUID, links[0].UUID)
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM artifact_tasks").Scan(&count))
	assert.Equal(t, 1, count)
}

func TestTaskReportFenceProductionTransaction(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d := recurringIntegrationDatabases(t, driver)
			clock, ids := util.NewRealTime(), util.NewUUIDGenerator()
			executor := "docker"
			_, err := NewRunStore(dbs[0], clock, d).CreateRun(&model.Run{
				UUID: "report-run", RunDetails: model.RunDetails{State: model.RuntimeStateRunning, ExecutorType: &executor},
			})
			require.NoError(t, err)
			tasks := NewTaskStore(dbs[0], clock, ids, d)
			task, err := tasks.CreateTask(&model.Task{RunUUID: "report-run", Name: "reporter", State: model.TaskStatus(apiv2.PipelineTask_RUNNING), Pods: model.JSONSlice{}, TypeAttrs: model.JSONData{}})
			require.NoError(t, err)
			require.NoError(t, tasks.RuntimeStore().Create(runtimeAttemptFixture(task, "attempt-a")))
			entered, release := make(chan struct{}), make(chan struct{})
			t.Cleanup(func() {
				select {
				case <-release:
				default:
					close(release)
				}
			})
			reportDone := make(chan error, 1)
			go func() {
				reportDone <- tasks.withTaskReport(&model.TaskReportIdentity{TaskUUID: task.UUID, AttemptUUID: "attempt-a"}, []string{task.UUID},
					func(tx *sql.Tx, _ *taskReportScope) error {
						close(entered)
						<-release
						return tasks.updateTaskInTransaction(tx, &model.Task{
							UUID: task.UUID, OutputParameters: model.JSONSlice{map[string]any{"parameterKey": "result", "type": "OUTPUT", "value": "attempt-a"}},
						}, false)
					})
			}()
			<-entered
			next := runtimeAttemptFixture(task, "attempt-b")
			retryDone := make(chan error, 1)
			go func() {
				retryDone <- NewTaskStore(dbs[1], clock, ids, d).RuntimeStore().BeginRetry(task.UUID, "attempt-a", next)
			}()
			select {
			case err := <-retryDone:
				close(release)
				require.NoError(t, <-reportDone)
				t.Fatalf("retry committed before the guarded report write finished: %v", err)
			case <-time.After(100 * time.Millisecond):
			}
			close(release)
			require.NoError(t, <-reportDone)
			require.NoError(t, <-retryDone)
			stored, err := tasks.GetTask(task.UUID)
			require.NoError(t, err)
			assert.Empty(t, stored.OutputParameters, "retry must clear a prior report committed before ownership changed")
			_, err = tasks.ReportTasks(&model.TaskReportIdentity{TaskUUID: task.UUID, AttemptUUID: "attempt-a"},
				[]*model.Task{{UUID: task.UUID, StatusMetadata: model.JSONData{"old": true}}})
			require.ErrorIs(t, err, ErrRuntimeTaskConflict)
			current, err := tasks.RuntimeStore().Get(task.UUID)
			require.NoError(t, err)
			assert.Equal(t, "attempt-b", *current.AttemptUUID)
			assert.Equal(t, 1, current.RetryCount)
			number := 0.75
			source := []*model.Artifact{
				{Name: "model", Type: model.ArtifactType(apiv2.Artifact_Model), Metadata: model.JSONData{"attempt": "b"}},
				{Name: "metric", Type: model.ArtifactType(apiv2.Artifact_Metric), NumberValue: &number},
			}
			links := []*model.ArtifactTask{
				{TaskID: task.UUID, RunUUID: task.RunUUID, Type: model.IOType(apiv2.IOType_OUTPUT), ArtifactKey: "model"},
				{TaskID: task.UUID, RunUUID: task.RunUUID, Type: model.IOType(apiv2.IOType_OUTPUT), ArtifactKey: "metric"},
			}
			identity := &model.TaskReportIdentity{TaskUUID: task.UUID, AttemptUUID: "attempt-b"}
			first, firstLinks, err := tasks.ReportArtifacts(identity, source, links, false)
			require.NoError(t, err)
			source[0].CreatedAtInSec = 123
			replayed, replayedLinks, err := NewTaskStore(dbs[2], clock, ids, d).ReportArtifacts(identity, source, links, false)
			require.NoError(t, err)
			for i := range first {
				assert.Equal(t, first[i].UUID, replayed[i].UUID)
				assert.Equal(t, firstLinks[i].UUID, replayedLinks[i].UUID)
			}
			_, _, err = tasks.ReportArtifacts(&model.TaskReportIdentity{TaskUUID: task.UUID, AttemptUUID: "attempt-a"}, source, links, false)
			require.ErrorIs(t, err, ErrRuntimeTaskConflict)
		})
	}
}
