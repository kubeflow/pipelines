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

func TestTaskPagination_NullableSortFields(t *testing.T) {
	for _, child := range []bool{false, true} {
		for _, field := range []struct {
			api, column string
			numeric     bool
		}{
			{"name", "Name", false}, {"display_name", "DisplayName", false}, {"scope_path", "ScopePath", false},
			{"start_time", "StartedInSec", true}, {"end_time", "FinishedInSec", true},
		} {
			for _, direction := range []string{"asc", "desc"} {
				for _, size := range []int{1, 2, 3} {
					t.Run(fmt.Sprintf("child%v/%s/%s/page%d", child, field.api, direction, size), func(t *testing.T) {
						db, store, _ := initializeTaskStore()
						defer db.Close()
						var parent *string
						if child {
							store.uuid = util.NewFakeUUIDGeneratorOrFatal(testUUID1, nil)
							row, err := store.CreateTask(&model.Task{Namespace: "ns1", RunUUID: "run-1", Name: "parent"})
							require.NoError(t, err)
							parent = &row.UUID
						}
						values := []interface{}{nil, "", "a", "a", nil}
						if field.numeric {
							values = []interface{}{nil, 0, 1, 1, nil}
						}
						var nullIDs []string
						for i := 0; i < 5; i++ {
							id := fmt.Sprintf("123e4567-e89b-12d3-a456-42665544210%d", i)
							store.uuid = util.NewFakeUUIDGeneratorOrFatal(id, nil)
							_, err := store.CreateTask(&model.Task{Namespace: "ns1", RunUUID: "run-1", Name: fmt.Sprintf("task%d", i), ParentTaskUUID: parent})
							require.NoError(t, err)
							_, err = db.Exec("UPDATE tasks SET "+store.dbDialect.QuoteIdentifier(field.column)+" = ? WHERE UUID = ?", values[i], id)
							require.NoError(t, err)
							if values[i] == nil {
								nullIDs = append(nullIDs, id)
							}
						}
						fetch := func(opts *list.Options) ([]*model.Task, int, string, error) {
							if child {
								return store.ListChildTasksByParentAndRun(*parent, "run-1", opts)
							}
							return store.ListTasks(&model.FilterContext{}, opts)
						}
						opts, err := list.NewOptions(&model.Task{}, 100, field.api+" "+direction, nil)
						require.NoError(t, err)
						all, total, _, err := fetch(opts)
						require.NoError(t, err)
						require.Equal(t, 5, total)
						var expected []string
						for _, row := range all {
							expected = append(expected, row.UUID)
						}
						require.ElementsMatch(t, nullIDs, expected[len(expected)-2:])
						opts, err = list.NewOptions(&model.Task{}, size, field.api+" "+direction, nil)
						require.NoError(t, err)
						var actual []string
						seen := map[string]bool{}
						for page := 0; page < 10; page++ {
							rows, count, token, err := fetch(opts)
							require.NoError(t, err)
							require.Equal(t, total, count)
							for _, row := range rows {
								require.False(t, seen[row.UUID], "duplicate on page%d", page)
								seen[row.UUID] = true
								actual = append(actual, row.UUID)
							}
							if token == "" {
								require.Equal(t, expected, actual)
								return
							}
							opts, err = list.NewOptionsFromToken(token, size)
							require.NoError(t, err)
						}
						t.Fatal("pagination did not finish")
					})
				}
			}
		}
	}
}
