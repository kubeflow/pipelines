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
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
)

func TestSelectWithQuotedColumns(t *testing.T) {
	dialects := []dialectSpec{
		{name: "mysql", d: dialect.NewDBDialect("mysql")},
		{name: "pgx", d: dialect.NewDBDialect("pgx")},
	}

	cases := []struct {
		name        string
		columns     []string
		selectCount bool
		want        map[string]string // want SQL per dialect
	}{
		{
			name:        "select count",
			columns:     []string{"col1", "col2"}, // should be ignored
			selectCount: true,
			want: map[string]string{
				"mysql": "SELECT count(*) FROM `testTable`",
				"pgx":   `SELECT count(*) FROM "testTable"`,
			},
		},
		{
			name:        "select multiple columns",
			columns:     []string{"col1", "col2"},
			selectCount: false,
			want: map[string]string{
				"mysql": "SELECT `col1`, `col2` FROM `testTable`",
				"pgx":   `SELECT "col1", "col2" FROM "testTable"`,
			},
		},
		{
			name:        "select all with star",
			columns:     []string{"*"},
			selectCount: false,
			want: map[string]string{
				"mysql": "SELECT `*` FROM `testTable`", // Note: squirrel quotes '*'
				"pgx":   `SELECT "*" FROM "testTable"`,
			},
		},
	}

	for _, dl := range dialects {
		t.Run(dl.name, func(t *testing.T) {
			for _, tc := range cases {
				t.Run(tc.name, func(t *testing.T) {
					builder := selectWithQuotedColumns(dl.d.QueryBuilder(), dl.d.QuoteIdentifier, "testTable", tc.columns, tc.selectCount)
					gotSQL, _, _ := builder.ToSql()
					require.Equal(t, tc.want[dl.name], gotSQL)
				})
			}
		})
	}
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
				"mysql": "SELECT `*` FROM `testTable` WHERE `ExperimentUUID` = ?",
				"pgx":   `SELECT "*" FROM "testTable" WHERE "ExperimentUUID" = $1`,
			},
		},
		{
			name:  "count",
			count: true,
			expID: "123",
			want: map[string]string{
				"mysql": "SELECT count(*) FROM `testTable` WHERE `ExperimentUUID` = ?",
				"pgx":   `SELECT count(*) FROM "testTable" WHERE "ExperimentUUID" = $1`,
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
					require.Equal(t, tc.want[dl.name], gotSQL)
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
				"mysql": "SELECT `*` FROM `testTable` WHERE `Namespace` = ?",
				"pgx":   `SELECT "*" FROM "testTable" WHERE "Namespace" = $1`,
			},
		},
		{
			name:  "count",
			count: true,
			ns:    "ns",
			want: map[string]string{
				"mysql": "SELECT count(*) FROM `testTable` WHERE `Namespace` = ?",
				"pgx":   `SELECT count(*) FROM "testTable" WHERE "Namespace" = $1`,
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
					require.Equal(t, tc.want[dl.name], gotSQL)
				})
			}
		})
	}
}

// TestFilterByResourceReferenceCombinedPredicate verifies that when an additional
// WHERE predicate is appended after FilterByResourceReference, PostgreSQL placeholder
// numbers are assigned sequentially ($1/$2/$3 for the subquery, $4 for the extra
// predicate) with no collision.
func TestFilterByResourceReferenceCombinedPredicate(t *testing.T) {
	fctx := &model.FilterContext{
		ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: "ref-id"},
	}

	cases := []struct {
		name      string
		d         dialect.DBDialect
		wantSQL   string
		wantArgsN int
	}{
		{
			name:      "pgx",
			d:         dialect.NewDBDialect("pgx"),
			wantSQL:   `SELECT "*" FROM "testTable" WHERE "UUID" IN (SELECT "ResourceUUID" FROM "resource_references" AS "rf" WHERE ("rf"."ResourceType" = $1 AND "rf"."ReferenceUUID" = $2 AND "rf"."ReferenceType" = $3)) AND "Name" = $4`,
			wantArgsN: 4,
		},
		{
			name:      "mysql",
			d:         dialect.NewDBDialect("mysql"),
			wantSQL:   "SELECT `*` FROM `testTable` WHERE `UUID` IN (SELECT `ResourceUUID` FROM `resource_references` AS `rf` WHERE (`rf`.`ResourceType` = ? AND `rf`.`ReferenceUUID` = ? AND `rf`.`ReferenceType` = ?)) AND `Name` = ?",
			wantArgsN: 4,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			qb := tc.d.QueryBuilder()
			q := tc.d.QuoteIdentifier

			builder, err := FilterByResourceReference(qb, q, "testTable", []string{"*"}, model.RunResourceType, false, fctx)
			require.NoError(t, err)

			builder = builder.Where(sq.Eq{q("Name"): "foo"})

			gotSQL, args, err := builder.ToSql()
			require.NoError(t, err)
			require.Equal(t, tc.wantArgsN, len(args))
			require.Equal(t, tc.wantSQL, gotSQL)
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
		name      string
		in        in
		want      map[string]string // want full SQL per dialect
		wantArgsN int
	}{
		{
			name: "no-filter-select",
			in:   in{count: false, fctx: &model.FilterContext{}},
			want: map[string]string{
				"mysql": "SELECT `*` FROM `testTable`",
				"pgx":   `SELECT "*" FROM "testTable"`,
			},
			wantArgsN: 0,
		},
		{
			name: "no-filter-count",
			in:   in{count: true, fctx: &model.FilterContext{}},
			want: map[string]string{
				"mysql": "SELECT count(*) FROM `testTable`",
				"pgx":   `SELECT count(*) FROM "testTable"`,
			},
			wantArgsN: 0,
		},
		{
			name: "with-ref-select",
			in:   in{count: false, fctx: &model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: "test3"}}},
			want: map[string]string{
				"mysql": "SELECT `*` FROM `testTable` WHERE `UUID` IN (SELECT `ResourceUUID` FROM `resource_references` AS `rf` WHERE (`rf`.`ResourceType` = ? AND `rf`.`ReferenceUUID` = ? AND `rf`.`ReferenceType` = ?))",
				"pgx":   `SELECT "*" FROM "testTable" WHERE "UUID" IN (SELECT "ResourceUUID" FROM "resource_references" AS "rf" WHERE ("rf"."ResourceType" = $1 AND "rf"."ReferenceUUID" = $2 AND "rf"."ReferenceType" = $3))`,
			},
			wantArgsN: 3,
		},
		{
			name: "with-ref-count",
			in:   in{count: true, fctx: &model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: "test4"}}},
			want: map[string]string{
				"mysql": "SELECT count(*) FROM `testTable` WHERE `UUID` IN (SELECT `ResourceUUID` FROM `resource_references` AS `rf` WHERE (`rf`.`ResourceType` = ? AND `rf`.`ReferenceUUID` = ? AND `rf`.`ReferenceType` = ?))",
				"pgx":   `SELECT count(*) FROM "testTable" WHERE "UUID" IN (SELECT "ResourceUUID" FROM "resource_references" AS "rf" WHERE ("rf"."ResourceType" = $1 AND "rf"."ReferenceUUID" = $2 AND "rf"."ReferenceType" = $3))`,
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
					require.Equal(t, tc.want[dl.name], gotSQL)
				})
			}
		})
	}
}
