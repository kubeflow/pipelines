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
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportedRunProvenanceSurvivesReadsAndUpdates(t *testing.T) {
	db, _, runStore := initializeRunStore()
	defer db.Close()
	run, err := runStore.GetRun("1")
	require.NoError(t, err)
	assert.Nil(t, run.ImportedFrom, "ordinary run creation keeps provenance null")
	assert.Nil(t, run.ImportDigest)
	source, digest := "source-cluster", strings.Repeat("a", 64)
	_, err = db.Exec(`UPDATE run_details SET ImportedFrom = ?, ImportDigest = ? WHERE UUID = ?`, source, digest, "1")
	require.NoError(t, err)

	// Updates made with an older model instance must not erase provenance.
	require.NoError(t, runStore.UpdateRun(run))
	run, err = runStore.GetRun("1")
	require.NoError(t, err)
	require.NotNil(t, run.ImportedFrom)
	require.NotNil(t, run.ImportDigest)
	assert.Equal(t, source, *run.ImportedFrom)
	assert.Equal(t, digest, *run.ImportDigest)

	opts, err := list.NewOptions(&model.Run{}, 10, "id", nil)
	require.NoError(t, err)
	runs, _, _, err := runStore.ListRuns(&model.FilterContext{}, opts)
	require.NoError(t, err)
	require.Len(t, runs, 3)
	assert.Equal(t, run.ImportedFrom, runs[0].ImportedFrom)
	assert.Equal(t, run.ImportDigest, runs[0].ImportDigest)
	assert.Nil(t, runs[1].ImportedFrom)
	assert.Nil(t, runs[1].ImportDigest)
}
