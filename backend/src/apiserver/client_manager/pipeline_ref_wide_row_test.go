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

package clientmanager

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBackfillPipelineRefs_MySQLWideRowCandidateReads(t *testing.T) {
	db, d := setupBackfillTestDB(t, "mysql")
	require.NoError(t, db.Create(&model.Experiment{UUID: "exp", Name: "experiment", Namespace: "ns"}).Error)
	require.NoError(t, db.Create(&model.Pipeline{UUID: "parent", Name: "parent", DisplayName: "parent", Namespace: "ns", Status: model.PipelineReady}).Error)
	require.NoError(t, db.Create(&model.PipelineVersion{UUID: "version", Name: "version", DisplayName: "version", PipelineId: "parent", PipelineSpec: "spec", Status: model.PipelineVersionReady}).Error)
	runs := make([]model.Run, 256)
	for i := range runs {
		runs[i] = model.Run{UUID: fmt.Sprintf("run-%03d", i), ExperimentId: "exp", Namespace: "ns", StorageState: model.StorageStateAvailable}
	}
	require.NoError(t, db.CreateInBatches(&runs, 100).Error)
	// Build the LOBs server-side so neither seed statements nor their bind
	// parameters carry megabytes through the test process or the CI logs.
	wideIDs := []string{"run-010", "run-050", "run-100", "run-150", "run-200", "run-250"}
	require.NoError(t, db.Exec("UPDATE run_details SET WorkflowRuntimeManifest = REPEAT('w', 2097152), PipelineSpecManifest = REPEAT('p', 2097152) WHERE UUID IN ?", wideIDs).Error)
	var bytesStored int64
	require.NoError(t, db.Raw("SELECT SUM(OCTET_LENGTH(WorkflowRuntimeManifest) + OCTET_LENGTH(PipelineSpecManifest)) FROM run_details").Scan(&bytesStored).Error)
	require.EqualValues(t, 24<<20, bytesStored)

	for _, id := range wideIDs[:2] {
		for _, refType := range []model.ResourceType{model.PipelineResourceType, model.PipelineVersionResourceType} {
			require.NoError(t, db.Create(&model.ResourceReference{ResourceUUID: id, ResourceType: model.RunResourceType,
				ReferenceUUID: "ref", ReferenceType: refType, Relationship: model.OwnerRelationship}).Error)
		}
	}
	require.NoError(t, db.Exec("UPDATE run_details SET PipelineVersionId = 'version' WHERE UUID IN ?", wideIDs[2:4]).Error)
	require.NoError(t, db.Exec("ANALYZE TABLE run_details, resource_references, pipeline_versions").Error)

	// Readers must finish while a writer owns every wide row, including the
	// repairable rows. An UPDATE/FOR UPDATE candidate scan would have to wait.
	writer := db.Begin()
	require.NoError(t, writer.Error)
	defer writer.Rollback()
	require.NoError(t, writer.Exec("UPDATE run_details SET DisplayName = 'held' WHERE UUID IN ?", wideIDs).Error)

	require.NoError(t, db.Connection(func(conn *gorm.DB) error {
		conn = conn.Session(&gorm.Session{})
		bytesSent := func(t *testing.T) uint64 {
			t.Helper()
			var name string
			var n uint64
			require.NoError(t, conn.Raw("SHOW SESSION STATUS LIKE 'Bytes_sent'").Row().Scan(&name, &n))
			return n
		}
		for _, tc := range []struct {
			name string
			pass pipelineRefPass
			want []string
		}{
			{"direct", backfillRefColumnSQL(d, "run_details", "PipelineId", model.PipelineResourceType), wideIDs[:2]},
			{"version_reference", backfillRefColumnSQL(d, "run_details", "PipelineVersionId", model.PipelineVersionResourceType), wideIDs[:2]},
			{"version_parent", backfillPipelineIDFromVersionSQL(d, "run_details"), wideIDs[2:4]},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var access []struct{ Table, Type, Key string }
				require.NoError(t, conn.Raw("EXPLAIN "+tc.pass.selectIDs, "", pipelineRefBatchSize).Scan(&access).Error)
				t.Logf("candidate plan (table, access, index): %v", access)
				var plan string
				require.NoError(t, conn.Raw("EXPLAIN FORMAT=JSON "+tc.pass.selectIDs, "", pipelineRefBatchSize).Row().Scan(&plan))
				require.NotContains(t, plan, "WorkflowRuntimeManifest")
				require.NotContains(t, plan, "PipelineSpecManifest")
				for _, cursor := range []string{"", tc.want[len(tc.want)-1]} {
					before := bytesSent(t)
					start := time.Now()
					ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
					var ids []string
					err := conn.WithContext(ctx).Raw(tc.pass.selectIDs, cursor, pipelineRefBatchSize).Scan(&ids).Error
					cancel()
					elapsed := time.Since(start)
					require.NoError(t, err)
					require.Less(t, elapsed, 3*time.Second)
					if cursor == "" {
						require.Equal(t, tc.want, ids)
					} else {
						require.Empty(t, ids)
					}
					transferred := bytesSent(t) - before
					require.Less(t, transferred, uint64(4096), "UUID page must not transfer manifest contents")
					t.Logf("cursor=%q rows=%d session_bytes_sent=%d elapsed=%s", cursor, len(ids), transferred, elapsed)
				}
			})
		}
		return nil
	}))
}
