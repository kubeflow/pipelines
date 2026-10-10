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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

// Exercise endpoint-specific aliases and prefixes through the real request
// validator and SQL execution. The token layout comes from the captured 2.17.2
// fixture, with model-specific fields substituted; these are not live captures
// for the additional endpoints and do not assert MySQL collation parity.
func TestValidatedListOptions_LegacyEndpointMixedCaseCursors(t *testing.T) {
	cases := []struct {
		name, version, table, apiField, column string
		model                                  list.Listable
	}{
		{"pipeline_v1", "v1beta1", "pipelines", "name", "Name", &model.Pipeline{}},
		{"pipeline_v2", "v2beta1", "pipelines", "display_name", "DisplayName", &model.Pipeline{}},
		{"version_v1", "v1beta1", "pipeline_versions", "name", "Name", &model.PipelineVersion{}},
		{"version_v2", "v2beta1", "pipeline_versions", "display_name", "DisplayName", &model.PipelineVersion{}},
		{"job_v1", "v1beta1", "jobs", "name", "DisplayName", &model.Job{}},
		{"recurring_run_v2", "v2beta1", "jobs", "display_name", "DisplayName", &model.Job{}},
		{"task_v1", "v1beta1", "tasks", "display_name", "Name", &model.Task{}},
	}
	legacy := legacyFilterTokens(t)[0]
	decoded, err := base64.StdEncoding.DecodeString(legacy.Token)
	require.NoError(t, err)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			db, err := sql.Open("sqlite3", ":memory:")
			require.NoError(t, err)
			defer db.Close()
			quote := func(value string) string { return `"` + value + `"` }
			_, err = db.Exec(fmt.Sprintf(`CREATE TABLE %s (UUID TEXT PRIMARY KEY, %s TEXT)`, quote(tc.table), quote(tc.column)))
			require.NoError(t, err)
			for i, name := range []string{"alpha", "ALPHA", "Alpha", "Bravo"} {
				_, err = db.Exec(fmt.Sprintf(`INSERT INTO %s VALUES (?, ?)`, quote(tc.table)), fmt.Sprintf("row%d", i+1), name)
				require.NoError(t, err)
			}
			for _, desc := range []bool{false, true} {
				var token map[string]interface{}
				require.NoError(t, json.Unmarshal(decoded, &token))
				token["SortByFieldName"], token["SortBySQLColumn"] = tc.column, tc.column
				token["SortByFieldValue"], token["KeyFieldValue"] = "ALPHA", "row2"
				token["SortByFieldPrefix"], token["KeyFieldPrefix"] = tc.table+".", tc.table+"."
				token["ModelName"], token["IsDesc"] = tc.table, desc
				token["Filter"].(map[string]interface{})["EQ"] = map[string]interface{}{tc.table + "." + tc.column: []string{"alpha"}}
				encoded, err := json.Marshal(token)
				require.NoError(t, err)
				for _, repeat := range []bool{false, true} {
					sortBy, filterSpec := "", ""
					if repeat {
						sortBy = tc.apiField + " asc"
						if desc {
							sortBy = tc.apiField + " desc"
						}
						operation := "operation"
						if tc.version == "v1beta1" {
							operation = "op"
						}
						filterSpec = fmt.Sprintf(`{"predicates":[{"key":%q,%q:"EQUALS","string_value":"alpha"}]}`, tc.apiField, operation)
					}
					opts, err := validatedListOptions(tc.model, base64.StdEncoding.EncodeToString(encoded), 10, sortBy, filterSpec, tc.version)
					if tc.name == "task_v1" && !desc {
						require.Error(t, err)
						require.Contains(t, err.Error(), "Clear page_token")
						continue
					}
					require.NoError(t, err)
					query := opts.AddFilterToSelect(sq.Select("UUID").From(quote(tc.table)), quote)
					querySQL, args, err := opts.AddPaginationToSelect(query, quote, "").ToSql()
					require.NoError(t, err)
					rows, err := db.Query(querySQL, args...)
					require.NoError(t, err)
					var got []string
					for rows.Next() {
						var id string
						require.NoError(t, rows.Scan(&id))
						got = append(got, id)
					}
					require.NoError(t, rows.Err())
					require.NoError(t, rows.Close())
					want := []string{"row2", "row3"}
					if desc {
						want = []string{"row2", "row1"}
					}
					require.Equal(t, want, got, "descending=%v repeat=%v SQL=%s", desc, repeat, querySQL)
				}
			}
		})
	}
}

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
		opts, err := validatedListOptions(&model.Pipeline{}, token, 1, "description", "", "v2beta1")
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
