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

package storage

import (
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

func TestListPipelineVersions_NullDescriptionCursorDoesNotRepeatRows(t *testing.T) {
	for _, nullCount := range []int{1, 3, 5} {
		for _, direction := range []string{"asc", "desc"} {
			t.Run(fmt.Sprintf("nulls_%d_%s", nullCount, direction), func(t *testing.T) {
				testNullableVersionDescription(t, nullCount, direction)
			})
		}
	}
}

func testNullableVersionDescription(t *testing.T, nullCount int, direction string) {
	t.Helper()
	db, dbDialect := NewFakeDBOrFatal()
	defer db.Close()
	store := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewFakeUUIDGeneratorOrFatal(DefaultFakePipelineId, nil), dbDialect)
	pipeline, err := store.CreatePipeline(&model.Pipeline{Name: "pagination", Status: model.PipelineReady})
	require.NoError(t, err)
	var nullIDs []string
	for i := 0; i < 5; i++ {
		id := fmt.Sprintf("123e4567-e89b-12d3-a456-42665544010%d", i)
		store.uuid = util.NewFakeUUIDGeneratorOrFatal(id, nil)
		version, err := store.CreatePipelineVersion(&model.PipelineVersion{
			Name: fmt.Sprintf("version%d", i), Description: model.LargeText(fmt.Sprintf("description%d", i)),
			PipelineId: pipeline.UUID, Status: model.PipelineVersionReady,
		})
		require.NoError(t, err)
		if i < nullCount {
			nullIDs = append(nullIDs, version.UUID)
		}
	}
	for _, id := range nullIDs {
		_, err = db.Exec(`UPDATE pipeline_versions SET Description = NULL WHERE UUID = ?`, id)
		require.NoError(t, err)
	}
	opts, err := list.NewOptions(&model.PipelineVersion{}, 2, "description "+direction, nil)
	require.NoError(t, err)
	seen := map[string]bool{}
	for page := 0; page < 10; page++ {
		rows, total, token, err := store.ListPipelineVersions(pipeline.UUID, opts, nil)
		require.NoError(t, err)
		for _, row := range rows {
			require.False(t, seen[row.UUID], "duplicate version on page %d", page)
			seen[row.UUID] = true
		}
		if token == "" {
			require.Len(t, seen, total)
			return
		}
		opts, err = list.NewOptionsFromToken(token, 2)
		require.NoError(t, err)
	}
	t.Fatal("pagination did not finish")
}
