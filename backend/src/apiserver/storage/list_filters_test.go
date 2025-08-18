// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
)

// normalize SQL for robust comparison across small formatting diffs
func squeezeSpaces(s string) string { return strings.Join(strings.Fields(s), " ") }
func normalizeSQL(s string) string {
	s = squeezeSpaces(s)
	// strip identifier quotes across dialects
	s = strings.ReplaceAll(s, "`", "")
	s = strings.ReplaceAll(s, "\"", "")
	// make case-insensitive on keywords/identifiers
	return strings.ToLower(s)
}

type dialectSpec struct {
	name string
	d    dialect.DBDialect
}

func TestFilterByExperiment(t *testing.T) {
	dialects := []dialectSpec{
		{name: "mysql", d: dialect.NewDBDialect("mysql")},
		{name: "pgx", d: dialect.NewDBDialect("pgx")},
	}

	cases := []struct {
		name  string
		count bool
		expID string
		want  map[string]string // want SQL per dialect
	}{
		{
			name:  "select",
			count: false,
			expID: "123",
			want: map[string]string{
				"mysql": "SELECT * FROM `testTable` WHERE `ExperimentUUID` = ?",
				"pgx":   "SELECT * FROM \"testTable\" WHERE \"ExperimentUUID\" = $1",
			},
		},
		{
			name:  "count",
			count: true,
			expID: "123",
			want: map[string]string{
				"mysql": "SELECT count(*) FROM `testTable` WHERE `ExperimentUUID` = ?",
				"pgx":   "SELECT count(*) FROM \"testTable\" WHERE \"ExperimentUUID\" = $1",
			},
		},
	}

	for _, dl := range dialects {
		t.Run(dl.name, func(t *testing.T) {
			qb := dl.d.QueryBuilder()
			q := dl.d.QuoteIdentifier

			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					builder, err := FilterByExperiment(qb, q, "testTable", []string{"*"}, tc.count, tc.expID)
					require.NoError(t, err)

					gotSQL, args, err := builder.ToSql()
					require.NoError(t, err)
					require.Equal(t, 1, len(args))

					require.Equal(t,
						normalizeSQL(tc.want[dl.name]),
						normalizeSQL(gotSQL),
					)
				})
			}
		})
	}
}

func TestFilterByNamespace(t *testing.T) {
	dialects := []dialectSpec{
		{name: "mysql", d: dialect.NewDBDialect("mysql")},
		{name: "pgx", d: dialect.NewDBDialect("pgx")},
	}

	cases := []struct {
		name  string
		count bool
		ns    string
		want  map[string]string
	}{
		{
			name:  "select",
			count: false,
			ns:    "ns",
			want: map[string]string{
				"mysql": "SELECT * FROM `testTable` WHERE `Namespace` = ?",
				"pgx":   "SELECT * FROM \"testTable\" WHERE \"Namespace\" = $1",
			},
		},
		{
			name:  "count",
			count: true,
			ns:    "ns",
			want: map[string]string{
				"mysql": "SELECT count(*) FROM `testTable` WHERE `Namespace` = ?",
				"pgx":   "SELECT count(*) FROM \"testTable\" WHERE \"Namespace\" = $1",
			},
		},
	}

	for _, dl := range dialects {
		t.Run(dl.name, func(t *testing.T) {
			qb := dl.d.QueryBuilder()
			q := dl.d.QuoteIdentifier
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					builder, err := FilterByNamespace(qb, q, "testTable", []string{"*"}, tc.count, tc.ns)
					require.NoError(t, err)

					gotSQL, args, err := builder.ToSql()
					require.NoError(t, err)
					require.Equal(t, 1, len(args))

					require.Equal(t,
						normalizeSQL(tc.want[dl.name]),
						normalizeSQL(gotSQL),
					)
				})
			}
		})
	}
}

func TestFilterByResourceReference(t *testing.T) {
	dialects := []dialectSpec{
		{name: "mysql", d: dialect.NewDBDialect("mysql")},
		{name: "pgx", d: dialect.NewDBDialect("pgx")},
	}

	type in struct {
		count bool
		fctx  *model.FilterContext
	}

	cases := []struct {
		name string
		in   in
		// For complex subquery, check key fragments rather than full string equality.
		wantBase  string              // base prefix to match (dialect-agnostic, normalized)
		wantFrag  map[string][]string // dialect -> fragments that must appear
		wantArgsN int
	}{
		{
			name:      "no-filter-select",
			in:        in{count: false, fctx: &model.FilterContext{}},
			wantBase:  "select * from testtable",
			wantFrag:  map[string][]string{"mysql": {}, "pgx": {}},
			wantArgsN: 0,
		},
		{
			name:      "no-filter-count",
			in:        in{count: true, fctx: &model.FilterContext{}},
			wantBase:  "select count(*) from testtable",
			wantFrag:  map[string][]string{"mysql": {}, "pgx": {}},
			wantArgsN: 0,
		},
		{
			name:     "with-ref-select",
			in:       in{count: false, fctx: &model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: "test3"}}},
			wantBase: "select * from testtable where uuid in (select resourceuuid from resource_references as rf where",
			wantFrag: map[string][]string{
				"mysql": {"rf.resourcetype = ?", "rf.referenceuuid = ?", "rf.referencetype = ?"},
				"pgx":   {"rf.resourcetype = $1", "rf.referenceuuid = $2", "rf.referencetype = $3"},
			},
			wantArgsN: 3,
		},
		{
			name:     "with-ref-count",
			in:       in{count: true, fctx: &model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: "test4"}}},
			wantBase: "select count(*) from testtable where uuid in (select resourceuuid from resource_references as rf where",
			wantFrag: map[string][]string{
				"mysql": {"rf.resourcetype = ?", "rf.referenceuuid = ?", "rf.referencetype = ?"},
				"pgx":   {"rf.resourcetype = $1", "rf.referenceuuid = $2", "rf.referencetype = $3"},
			},
			wantArgsN: 3,
		},
	}

	for _, dl := range dialects {
		t.Run(dl.name, func(t *testing.T) {
			qb := dl.d.QueryBuilder()
			q := dl.d.QuoteIdentifier
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					builder, err := FilterByResourceReference(qb, q, "testTable", []string{"*"}, model.RunResourceType, tc.in.count, tc.in.fctx)
					require.NoError(t, err)

					gotSQL, args, err := builder.ToSql()
					require.NoError(t, err)
					require.Equal(t, tc.wantArgsN, len(args))

					ng := normalizeSQL(gotSQL)
					require.Truef(t, strings.HasPrefix(ng, tc.wantBase), "sql prefix mismatch:\nwant prefix: %s\ngot: %s", tc.wantBase, ng)

					for _, frag := range tc.wantFrag[dl.name] {
						require.Containsf(t, ng, frag, "missing fragment %q in sql: %s", frag, ng)
					}
				})
			}
		})
	}
}
