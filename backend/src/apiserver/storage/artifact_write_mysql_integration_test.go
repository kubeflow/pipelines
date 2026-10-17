// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"sync"
	"testing"

	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

func TestCreateArtifactWithTask_ConcurrentSameIdentityMySQL(t *testing.T) {
	dbs, d := recurringIntegrationDatabases(t, "mysql")

	clock := util.NewFakeTimeForEpoch()
	uuid := util.NewUUIDGenerator()

	runs := NewRunStore(dbs[0], clock, d)
	_, err := runs.CreateRun(&model.Run{
		UUID:         "mysql-artifact-run",
		DisplayName:  "MySQL artifact concurrency",
		Namespace:    "mysql-ns",
		StorageState: model.StorageStateAvailable,
		RunDetails:   model.RunDetails{State: model.RuntimeStateRunning},
	})
	require.NoError(t, err)

	tasks := NewTaskStore(dbs[0], clock, uuid, d)
	task, err := tasks.CreateTask(&model.Task{
		Namespace:   "mysql-ns",
		RunUUID:     "mysql-artifact-run",
		Name:        "producer",
		Fingerprint: "mysql-artifact-concurrency",
		ScopePath:   "root.producer",
		Type:        model.TaskType(apiv2beta1.PipelineTask_RUNTIME),
		State:       model.TaskStatus(apiv2beta1.PipelineTask_SUCCEEDED),
		Pods:        model.JSONSlice{},
		TypeAttrs:   model.JSONData{},
	})
	require.NoError(t, err)

	firstStore := NewArtifactStore(
		dbs[0],
		clock,
		util.NewUUIDGenerator(),
		d,
	)
	secondStore := NewArtifactStore(
		dbs[1],
		clock,
		util.NewUUIDGenerator(),
		d,
	)

	artifact := &model.Artifact{
		Namespace: "mysql-ns",
		Type:      model.ArtifactType(apiv2beta1.Artifact_Model),
		Name:      "concurrent-mysql-artifact",
		URI:       strPTR("s3://mysql-artifact/concurrent"),
	}

	newArtifactTask := func() *model.ArtifactTask {
		return &model.ArtifactTask{
			TaskID:      task.UUID,
			RunUUID:     "mysql-artifact-run",
			Type:        model.IOType(apiv2beta1.IOType_OUTPUT),
			ArtifactKey: "output",
			Producer:    model.JSONData{"taskName": "producer"},
		}
	}

	type result struct {
		artifact     *model.Artifact
		artifactTask *model.ArtifactTask
		err          error
	}

	start := make(chan struct{})
	results := make(chan result, 2)

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		<-start

		createdArtifact, createdArtifactTask, err :=
			firstStore.CreateArtifactWithTask(
				artifact,
				newArtifactTask(),
				"mysql-concurrent-operation-1",
			)

		results <- result{
			artifact:     createdArtifact,
			artifactTask: createdArtifactTask,
			err:          err,
		}
	}()

	go func() {
		defer wg.Done()
		<-start

		createdArtifact, createdArtifactTask, err :=
			secondStore.CreateArtifactWithTask(
				artifact,
				newArtifactTask(),
				"mysql-concurrent-operation-1",
			)

		results <- result{
			artifact:     createdArtifact,
			artifactTask: createdArtifactTask,
			err:          err,
		}
	}()

	close(start)
	wg.Wait()
	close(results)

	var resultList []result
	for r := range results {
		resultList = append(resultList, r)
	}

	require.Len(t, resultList, 2)

	for _, r := range resultList {
		require.NoError(t, r.err)
		require.NotNil(t, r.artifact)
		require.NotNil(t, r.artifactTask)
	}

	require.Equal(t, resultList[0].artifact.UUID, resultList[1].artifact.UUID)
	require.Equal(t, resultList[0].artifactTask.UUID, resultList[1].artifactTask.UUID)

	var artifactCount, linkCount, identityCount int

	require.NoError(
		t,
		dbs[0].QueryRow("SELECT count(*) FROM artifacts").Scan(&artifactCount),
	)
	require.NoError(
		t,
		dbs[0].QueryRow("SELECT count(*) FROM artifact_tasks").Scan(&linkCount),
	)
	require.NoError(
		t,
		dbs[0].QueryRow(
			"SELECT count(*) FROM artifact_write_identities",
		).Scan(&identityCount),
	)

	require.Equal(t, 1, artifactCount)
	require.Equal(t, 1, linkCount)
	require.Equal(t, 1, identityCount)
}
