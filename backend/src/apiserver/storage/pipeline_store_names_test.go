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
	"database/sql"
	"errors"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Ready rows resolve to their names. Missing IDs and rows that are no longer
// ready are left out, and a repeated ID is harmless.
func TestPipelineStore_NamesForIds(t *testing.T) {
	db, testDialect := NewFakeDBOrFatal()
	defer db.Close()
	store := NewPipelineStore(db, util.NewFakeTimeForEpoch(), util.NewUUIDGenerator(), testDialect)

	training, err := store.CreatePipeline(createPipeline("training", "", ""))
	require.NoError(t, err)
	deleting, err := store.CreatePipeline(createPipeline("deleting", "", ""))
	require.NoError(t, err)
	require.NoError(t, store.UpdatePipelineStatus(deleting.UUID, model.PipelineDeleting))

	v1, err := store.CreatePipelineVersion(createPipelineVersion(training.UUID, "v1", "", "", "", ""))
	require.NoError(t, err)
	v2, err := store.CreatePipelineVersion(createPipelineVersion(training.UUID, "v2", "", "", "", ""))
	require.NoError(t, err)
	require.NoError(t, store.UpdatePipelineVersionStatus(v2.UUID, model.PipelineVersionDeleting))

	pipelineNames, err := store.GetPipelineNamesForIds([]string{training.UUID, training.UUID, deleting.UUID, "missing"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{training.UUID: "training"}, pipelineNames)

	versionNames, err := store.GetPipelineVersionNamesForIds([]string{v1.UUID, v2.UUID, "missing"})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{v1.UUID: "v1"}, versionNames)

	empty, err := store.GetPipelineNamesForIds(nil)
	require.NoError(t, err)
	assert.Empty(t, empty)
}

// recordingQuerier captures the SQL it is given and fails, so the generated
// query can be checked for a dialect that has no database in unit tests.
type recordingQuerier struct {
	query string
	args  []any
}

func (q *recordingQuerier) Query(query string, args ...any) (*sql.Rows, error) {
	q.query, q.args = query, args
	return nil, errors.New("recorded")
}

// The column names are capitalized, so each dialect must quote them: unquoted,
// PostgreSQL folds them to lower case and the query fails.
func TestQueryNamesForIds_QuotesIdentifiersPerDialect(t *testing.T) {
	for engine, want := range map[string]string{
		"mysql": "SELECT `UUID`, `Name` FROM `pipelines` WHERE (`UUID` IN (?,?) AND `Status` = ?)",
		"pgx":   `SELECT "UUID", "Name" FROM "pipelines" WHERE ("UUID" IN ($1,$2) AND "Status" = $3)`,
	} {
		t.Run(engine, func(t *testing.T) {
			querier := &recordingQuerier{}
			_, err := queryNamesForIds(querier, dialect.NewDBDialect(engine), "pipelines", []string{"a", "b"}, "READY")
			require.ErrorContains(t, err, "recorded")
			assert.Equal(t, want, querier.query)
			assert.Equal(t, []any{"a", "b", "READY"}, querier.args)
		})
	}
}
