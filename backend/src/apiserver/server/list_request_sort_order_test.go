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

package server

import (
	"database/sql"
	"fmt"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func TestValidatedListOptions_LargeTextSortKeepsFirstAndLaterPageOrder(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE pipelines (UUID TEXT PRIMARY KEY, Description TEXT)`)
	require.NoError(t, err)
	for i, description := range []string{"alpha", "Bravo", "charlie"} {
		_, err = db.Exec(`INSERT INTO pipelines VALUES (?, ?)`, fmt.Sprintf("row%d", i+1), description)
		require.NoError(t, err)
	}
	quote := func(value string) string { return `"` + value + `"` }
	token := ""
	seen := map[string]bool{}
	for page := 0; page < 5; page++ {
		opts, err := validatedListOptions(&model.Pipeline{}, token, 1, "description", "")
		require.NoError(t, err)
		query, args, err := opts.AddPaginationToSelect(sq.Select("UUID", "Description").From("pipelines"), quote, "").ToSql()
		require.NoError(t, err)
		rows, err := db.Query(query, args...)
		require.NoError(t, err)
		var values []*model.Pipeline
		for rows.Next() {
			var id, description string
			require.NoError(t, rows.Scan(&id, &description))
			values = append(values, &model.Pipeline{UUID: id, Description: model.LargeText(description)})
		}
		require.NoError(t, rows.Err())
		require.NoError(t, rows.Close())
		if len(values) == 0 {
			require.Len(t, seen, 3)
			return
		}
		require.False(t, seen[values[0].UUID], "duplicate on page %d", page)
		seen[values[0].UUID] = true
		if len(values) == 1 {
			require.Len(t, seen, 3)
			return
		}
		token, err = opts.NextPageToken(values[1])
		require.NoError(t, err)
	}
	t.Fatal("pagination did not finish")
}
