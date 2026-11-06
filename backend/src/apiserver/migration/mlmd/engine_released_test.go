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

package mlmd_test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"testing"

	apiv2 "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmdfixture"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func released218Client(t *testing.T) *mlmdfixture.Source {
	t.Helper()
	source, err := mlmdfixture.Load("testdata/release_218/source_graph.json")
	require.NoError(t, err)
	return source
}

func TestMySQL218ReleasedSourceMigration(t *testing.T) {
	database, _, config, _ := scratchMySQLDatabase(t)
	dump, err := os.ReadFile("testdata/release_218/source.sql")
	require.NoError(t, err)
	prelude, remainder, found := strings.Cut(string(dump), "CREATE DATABASE /*!32312 IF NOT EXISTS*/ `mlpipeline`")
	require.True(t, found)
	_, source, found := strings.Cut(remainder, "USE `mlpipeline`;")
	require.True(t, found)
	source, _, found = strings.Cut(source, "CREATE DATABASE /*!32312 IF NOT EXISTS*/ `metadb`")
	require.True(t, found)
	_, err = database.Exec(prelude + source)
	require.NoError(t, err)
	_, err = database.Exec("CREATE TABLE mlmd_migration_state (ID tinyint NOT NULL PRIMARY KEY, CacheCutoffInSec bigint NOT NULL) ENGINE=InnoDB")
	require.NoError(t, err)
	_, err = database.Exec("INSERT INTO mlmd_migration_state (ID, CacheCutoffInSec) VALUES (1, 1793800000)")
	require.NoError(t, err)
	report, err := mlmd.Migrate(context.Background(), released218Client(t), database)
	require.NoError(t, err)
	require.GreaterOrEqual(t, report.Tasks, 10)
	require.Positive(t, report.CacheKeysVerified)

	var expected struct {
		Tasks []struct {
			SourceExecutionID int `json:"source_execution_id"`
			model.Task
		} `json:"tasks"`
		Artifacts []struct {
			SourceArtifactID int `json:"source_artifact_id"`
			model.Artifact
		} `json:"artifacts"`
		Links []struct {
			SourceEventID int `json:"source_event_id"`
			model.ArtifactTask
		} `json:"links"`
	}
	data, err := os.ReadFile("testdata/release_218/expected_native.json")
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(data, &expected))
	store, err := gorm.Open(mysql.Open(config.FormatDSN()), &gorm.Config{})
	require.NoError(t, err)
	for _, task := range expected.Tasks {
		var actual model.Task
		require.NoError(t, store.First(&actual, "UUID = ?", task.UUID).Error)
		if actual.StatusMetadata != nil {
			_, err := model.JSONDataToProtoMessage(actual.StatusMetadata, func() *apiv2.PipelineTask_StatusMetadata {
				return &apiv2.PipelineTask_StatusMetadata{}
			})
			require.NoError(t, err, "execution %d status metadata must be API-readable", task.SourceExecutionID)
		}
		require.Equal(t, task.RunUUID, actual.RunUUID, "execution %d", task.SourceExecutionID)
		require.Equal(t, task.State, actual.State, "execution %d", task.SourceExecutionID)
		require.Equal(t, task.Type, actual.Type, "execution %d", task.SourceExecutionID)
		require.Equal(t, task.TypeAttrs, actual.TypeAttrs, "execution %d", task.SourceExecutionID)
		require.Len(t, actual.Pods, len(task.Pods), "execution %d", task.SourceExecutionID)
		for index, expectedPod := range task.Pods {
			for key, value := range expectedPod.(map[string]any) {
				require.Equal(t, value, actual.Pods[index].(map[string]any)[key], "execution %d pod %s", task.SourceExecutionID, key)
			}
		}
		require.Equal(t, task.CreatedAtInSec, actual.CreatedAtInSec, "execution %d", task.SourceExecutionID)
		require.Equal(t, task.StartedInSec, actual.StartedInSec, "execution %d", task.SourceExecutionID)
		require.Equal(t, task.FinishedInSec, actual.FinishedInSec, "execution %d", task.SourceExecutionID)
		if task.StatusMetadata != nil {
			legacyKey, _ := task.StatusMetadata["customProperties"].(map[string]any)["legacy_cache_fingerprint"].(string)
			if legacyKey != "" {
				require.Equal(t, legacyKey, actual.StatusMetadata["customProperties"].(map[string]any)["legacy_cache_fingerprint"], "execution %d legacy fingerprint", task.SourceExecutionID)
				require.NotEqual(t, legacyKey, actual.Fingerprint, "execution %d", task.SourceExecutionID)
			}
		}
		if task.SourceExecutionID == 19 { //nolint:gocritic // These fixture cases have distinct assertions and failure messages.
			require.Len(t, actual.Fingerprint, 64, "verified producer must have a native key")
		} else if task.State == model.TaskStatus(apiv2.PipelineTask_CACHED) {
			require.Empty(t, actual.Fingerprint, "execution %d reuse must not be a native cache target", task.SourceExecutionID)
		} else if task.StatusMetadata == nil {
			require.Empty(t, actual.Fingerprint, "execution %d", task.SourceExecutionID)
		}
		if task.SourceExecutionID == 19 {
			require.Equal(t, "bf1ec48e-f30d-4c09-92d6-0765639d2a56", actual.StatusMetadata["customProperties"].(map[string]any)["legacy_task.sourceTaskUUID"])
		}
	}
	var graph releaseGraph
	readFixture(t, "release_218/source_graph.json", &graph)
	unjoinedCount := 0
	for _, source := range graph.Tasks {
		if source.ExecutionID != "" {
			continue
		}
		unjoinedCount++
		var actual model.Task
		require.NoError(t, store.First(&actual, "UUID = ?", source.ID).Error)
		require.Equal(t, source.RunID, actual.RunUUID)
		require.Equal(t, source.CreatedTimestamp, actual.CreatedAtInSec)
		if source.PodName != "" {
			require.Equal(t, source.PodName, actual.Pods[0].(map[string]any)["name"])
		}
		if actual.StatusMetadata != nil {
			_, err := model.JSONDataToProtoMessage(actual.StatusMetadata, func() *apiv2.PipelineTask_StatusMetadata {
				return &apiv2.PipelineTask_StatusMetadata{}
			})
			require.NoError(t, err, "KFP-only task %s status metadata must be API-readable", source.ID)
		}
	}
	require.Positive(t, unjoinedCount, "the released source must contain KFP-only tasks")
	for _, artifact := range expected.Artifacts {
		var actual model.Artifact
		require.NoError(t, store.First(&actual, "UUID = ?", artifact.UUID).Error)
		require.Equal(t, artifact.Name, actual.Name, "artifact %d", artifact.SourceArtifactID)
		require.Equal(t, artifact.NumberValue, actual.NumberValue, "artifact %d", artifact.SourceArtifactID)
	}
	for _, link := range expected.Links {
		var actual model.ArtifactTask
		require.NoError(t, store.First(&actual, "UUID = ?", link.UUID).Error)
		require.Equal(t, link.TaskID, actual.TaskID, "event %d", link.SourceEventID)
		require.Equal(t, link.ArtifactID, actual.ArtifactID, "event %d", link.SourceEventID)
		require.Equal(t, link.Iteration, actual.Iteration, "event %d", link.SourceEventID)
	}
	var originalKey string
	require.NoError(t, database.QueryRow("SELECT Fingerprint FROM tasks WHERE UUID = ?", mlmd.ExecutionUUID(19)).Scan(&originalKey))
	require.NotEmpty(t, originalKey)
	cacheStore := storage.NewTaskStore(database, nil, nil, dialect.NewDBDialect("mysql"))
	cacheHit, err := cacheStore.FindLatestCachedTask("kubeflow", originalKey)
	require.NoError(t, err)
	require.NotNil(t, cacheHit)
	require.Equal(t, mlmd.ExecutionUUID(19), cacheHit.UUID)
	legacyHit, err := cacheStore.FindLatestCachedTask("kubeflow", "78f8bc3791ff083f52854e558b59d1a64efd7f8a1d810be25e7f335cfeda2fa0")
	require.NoError(t, err)
	require.Nil(t, legacyHit)
	repeated, err := mlmd.Migrate(context.Background(), released218Client(t), database)
	require.NoError(t, err)
	require.Equal(t, report.Tasks, repeated.Tasks)
	require.Equal(t, report.CacheKeysVerified, repeated.CacheKeysVerified)
	var repeatedKey string
	require.NoError(t, database.QueryRow("SELECT Fingerprint FROM tasks WHERE UUID = ?", mlmd.ExecutionUUID(19)).Scan(&repeatedKey))
	require.Equal(t, originalKey, repeatedKey)
	var cutoff int64
	require.NoError(t, database.QueryRow("SELECT CacheCutoffInSec FROM mlmd_migration_state WHERE ID=1").Scan(&cutoff))
	require.Equal(t, int64(1793800000), cutoff)
}
