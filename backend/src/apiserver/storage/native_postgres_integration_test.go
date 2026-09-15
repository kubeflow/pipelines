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

package storage

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestNativeStoresPostgreSQL uses KFP_NATIVE_POSTGRES_TEST_DSN to opt in.
// It creates and removes only its own uniquely named schema; the supplied role
// must have CREATE SCHEMA permission. Use a disposable PostgreSQL database.
func TestNativeStoresPostgreSQL(t *testing.T) {
	dsn := os.Getenv("KFP_NATIVE_POSTGRES_TEST_DSN")
	if dsn == "" {
		t.Skip("set KFP_NATIVE_POSTGRES_TEST_DSN to run PostgreSQL integration coverage")
	}
	config, err := pgx.ParseConfig(dsn)
	require.NoError(t, err)
	admin := stdlib.OpenDB(*config)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	d := dialect.NewDBDialect("pgx")
	schema := fmt.Sprintf("kfp_native_test_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE SCHEMA " + d.QuoteIdentifier(schema))
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := admin.Exec("DROP SCHEMA " + d.QuoteIdentifier(schema) + " CASCADE")
		require.NoError(t, err)
	})
	config.RuntimeParams["search_path"] = schema
	db := stdlib.OpenDB(*config)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	orm, err := gorm.Open(postgres.New(postgres.Config{Conn: db}), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, orm.AutoMigrate(model.AllModels()...))

	clock := util.NewFakeTimeForEpoch()
	uuid := util.NewUUIDGenerator()
	runs := NewRunStore(db, clock, d)
	_, err = runs.CreateRun(&model.Run{
		UUID: "pg-run", DisplayName: "PostgreSQL native storage", Namespace: "pg-ns",
		StorageState: model.StorageStateAvailable,
		RunDetails:   model.RunDetails{State: model.RuntimeStateRunning},
	})
	require.NoError(t, err)
	tasks := NewTaskStore(db, clock, uuid, d)
	artifacts := NewArtifactStore(db, clock, uuid, d)
	links := NewArtifactTaskStore(db, uuid, d)
	taskInput := &model.Task{
		Namespace: "pg-ns", RunUUID: "pg-run", Name: "producer", Fingerprint: "pg-cache",
		ScopePath: "root.producer", Type: model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State: model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED),
		Pods:  model.JSONSlice{}, TypeAttrs: model.JSONData{},
	}
	task, err := tasks.CreateTask(taskInput)
	require.NoError(t, err)
	duplicateTask, err := tasks.CreateTask(taskInput)
	require.NoError(t, err)
	require.Equal(t, task.UUID, duplicateTask.UUID)
	uri := "s3://native-postgres/artifact"
	artifactInput := &model.Artifact{
		Namespace: "pg-ns", Name: "Result", Description: "PostgreSQL description", URI: &uri,
		Metadata: model.JSONData{"accuracy": 0.9},
	}
	linkInput := &model.ArtifactTask{
		TaskID: task.UUID, RunUUID: "pg-run", Type: model.IOType(apiv2beta1.IOType_OUTPUT), ArtifactKey: "result",
		Producer: model.JSONData{"taskName": "producer", "iteration": float64(2)},
	}
	artifact, link, err := artifacts.FindOrCreateArtifactWithTask(artifactInput, linkInput)
	require.NoError(t, err)
	again, sameLink, err := artifacts.FindOrCreateArtifactWithTask(artifactInput, linkInput)
	require.NoError(t, err)
	require.Equal(t, artifact.UUID, again.UUID)
	require.Equal(t, link.UUID, sameLink.UUID)

	loaded, err := tasks.GetTask(task.UUID)
	require.NoError(t, err)
	require.Len(t, loaded.OutputArtifactsHydrated, 1)
	require.Equal(t, artifact.Description, loaded.OutputArtifactsHydrated[0].Value.Description)
	hydratedRun, err := runs.GetRun("pg-run", true)
	require.NoError(t, err)
	require.Len(t, hydratedRun.Tasks, 1)
	require.Len(t, hydratedRun.Tasks[0].OutputArtifactsHydrated, 1)
	lightRun, err := runs.GetRun("pg-run", false)
	require.NoError(t, err)
	require.Equal(t, 1, lightRun.TaskCount)
	require.Empty(t, lightRun.Tasks)
	require.NoError(t, hydrateArtifactsForTasks(db, []*model.Task{loaded}, d))
	require.Len(t, loaded.OutputArtifactsHydrated, 1)
	opts, err := list.NewOptions(&model.Task{}, 1, "", nil)
	require.NoError(t, err)
	page, total, _, err := tasks.ListTasks(&model.FilterContext{}, opts)
	require.NoError(t, err)
	require.Len(t, page, 1)
	require.Equal(t, 1, total)
	artifactOpts, err := list.NewOptions(&model.Artifact{}, 1, "name", nil)
	require.NoError(t, err)
	artifactPage, total, _, err := artifacts.ListArtifacts(&model.FilterContext{}, artifactOpts)
	require.NoError(t, err)
	require.Len(t, artifactPage, 1)
	require.Equal(t, 1, total)
	linkOpts, err := list.NewOptions(&model.ArtifactTask{}, 1, "", nil)
	require.NoError(t, err)
	linkPage, total, _, err := links.ListArtifactTasks(nil, nil, linkOpts)
	require.NoError(t, err)
	require.Len(t, linkPage, 1)
	require.Equal(t, 1, total)

	_, err = tasks.UpdateTask(&model.Task{UUID: task.UUID, DisplayName: "Updated"})
	require.NoError(t, err)
	require.NoError(t, tasks.ResetTasksForRetry([]string{task.UUID}))
	loaded, err = tasks.GetTask(task.UUID)
	require.NoError(t, err)
	require.Equal(t, model.TaskStatus(apiv2beta1.PipelineTask_RUNNING), loaded.State)
	require.NoError(t, links.DeleteOutputArtifactTasksByTaskIDs([]string{task.UUID}))
	require.NoError(t, tasks.DeleteTasksForRun(nil, "pg-run"))
	count, err := tasks.GetTaskCountForRun("pg-run")
	require.NoError(t, err)
	require.Zero(t, count)
}
