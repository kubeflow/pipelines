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

package list

import (
	"database/sql"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// These predicates and the nil-value guard are copied from 2.17.2's
// list.AddSortingToSelect. SQLite has the same NULL-last DESC ordering as MySQL.
func TestHistoricalNullableDescendingCursorRequiresRestart(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(`CREATE TABLE items (id TEXT, value TEXT); INSERT INTO items VALUES ('a','z'),('b',''),('e',NULL),('d',NULL)`)
	require.NoError(t, err)
	ids := func(query string) []string {
		rows, err := db.Query(query)
		require.NoError(t, err)
		defer rows.Close()
		var got []string
		for rows.Next() {
			var id string
			require.NoError(t, rows.Scan(&id))
			got = append(got, id)
		}
		require.NoError(t, rows.Err())
		return got
	}
	// New non-NULL DESC token sent to an old reader drops the NULL suffix.
	require.Equal(t, []string{"b"}, ids(`SELECT id FROM items WHERE value < '' OR (value = '' AND id <= 'b') ORDER BY value DESC,id DESC`))
	require.Equal(t, []string{"b", "e", "d"}, ids(`SELECT id FROM items WHERE value < '' OR (value = '' AND id <= 'b') OR value IS NULL ORDER BY value DESC,id DESC`))
	// New true-NULL cursor sent to an old reader skips its WHERE guard entirely.
	require.Equal(t, []string{"a", "b"}, ids(`SELECT id FROM items ORDER BY value DESC,id DESC LIMIT 2`))
	// Old scanner coerces lookahead NULL 'e' to "" after page one [a,b]. A new
	// reader cannot distinguish that from an ordinary empty-string cursor, and
	// would repeat b with its corrected predicate. Even non-nil values are unsafe.
	require.Equal(t, []string{"b", "e", "d"}, ids(`SELECT id FROM items WHERE value < '' OR (value = '' AND id <= 'e') OR value IS NULL ORDER BY value DESC,id DESC`))
	opts := &Options{token: &token{SortByFieldName: "Name", SortBySQLColumn: "Name", SortByFieldValue: "", KeyFieldName: "UUID", KeyFieldValue: "e", IsDesc: true}}
	require.Equal(t, codes.FailedPrecondition, status.Code(opts.ValidateOrdering(&model.Task{})))
}
