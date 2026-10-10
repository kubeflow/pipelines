// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package storage

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
)

func TestRecurringRunReplayLookup(t *testing.T) {
	db, d := NewFakeDBOrFatal()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	testRecurringRunReplayLookup(t, db, d)
}

func testRecurringRunReplayLookup(t *testing.T, db *sql.DB, d dialect.DBDialect) {
	orm, err := OpenTransferDB(db, d.Name())
	require.NoError(t, err)
	const jobID = "replay-target"
	store := NewRunStore(db, util.NewFakeTimeForEpoch(), d)
	insert := func(id, name string, rawJob any, created int64, reference *model.ResourceReference) {
		t.Helper()
		run := &model.Run{UUID: "replay-" + id, DisplayName: name,
			RunDetails: model.RunDetails{CreatedAtInSec: created, State: model.RuntimeStateSucceeded}}
		require.NoError(t, orm.Create(run).Error)
		// Preserve empty strings as well as NULL despite GORM's default:null tag.
		require.NoError(t, orm.Model(run).Update("JobUUID", rawJob).Error)
		if reference != nil {
			reference.ResourceUUID = run.UUID
			require.NoError(t, orm.Create(reference).Error)
		}
	}
	lookup := func(t *testing.T, name, want string) {
		t.Helper()
		got, err := store.GetRunByRecurringRunIDAndDisplayName(jobID, name)
		require.NoError(t, err)
		require.Equal(t, want, got)
	}
	lookup(t, "missing", "")
	for _, tc := range []struct {
		name                        string
		rawJob                      any
		referenceJob                string
		resourceType, referenceType model.ResourceType
		match                       bool
	}{
		{"direct", jobID, "", "", "", true},
		{"empty-job", "", jobID, model.RunResourceType, model.JobResourceType, true},
		{"null-job", nil, jobID, model.RunResourceType, model.JobResourceType, true},
		{"direct-and-reference", jobID, jobID, model.RunResourceType, model.JobResourceType, true},
		{"direct-overrides-reference", jobID, "other", model.RunResourceType, model.JobResourceType, true},
		{"foreign-direct", "other", jobID, model.RunResourceType, model.JobResourceType, false},
		{"foreign-reference", "", "other", model.RunResourceType, model.JobResourceType, false},
		{"wrong-resource-type", nil, jobID, model.JobResourceType, model.JobResourceType, false},
		{"namespace-reference", nil, jobID, model.RunResourceType, model.NamespaceResourceType, false},
		{"unassociated", "", "", "", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var reference *model.ResourceReference
			if tc.referenceJob != "" {
				reference = &model.ResourceReference{ResourceType: tc.resourceType, ReferenceType: tc.referenceType, ReferenceUUID: tc.referenceJob}
			}
			insert(tc.name, tc.name, tc.rawJob, 1, reference)
			want := ""
			if tc.match {
				want = "replay-" + tc.name
			}
			lookup(t, tc.name, want)
			lookup(t, tc.name+"-missing", "")
			// Removing LIMIT exposes accidental duplicates between UNION ALL branches.
			query, args, err := d.FinalizeSelect(recurringRunReplayQuery(d.QuoteIdentifier, jobID, tc.name).RemoveLimit())
			require.NoError(t, err)
			var ids []string
			require.NoError(t, orm.Raw(query, args...).Scan(&ids).Error)
			if tc.match {
				require.Equal(t, []string{want}, ids)
			} else {
				require.Empty(t, ids)
			}
		})
	}
	t.Run("global-order", func(t *testing.T) {
		for _, tc := range []struct {
			id      string
			rawJob  any
			created int64
			want    string
		}{
			{"order-a", jobID, 10, "order-a"},
			{"order-b", nil, 20, "order-b"},
			{"order-c", jobID, 20, "order-c"},
			{"order-d", "", 20, "order-d"},
			{"order-z", jobID, 5, "order-d"},
		} {
			insert(tc.id, "ordered", tc.rawJob, tc.created, &model.ResourceReference{
				ResourceType: model.RunResourceType, ReferenceType: model.JobResourceType, ReferenceUUID: jobID})
			lookup(t, "ordered", "replay-"+tc.want)
		}
	})
}

func TestRecurringRunReplayQueryDialects(t *testing.T) {
	const jobID, name = "job' OR 1=1 --", "tick' OR 1=1 --"
	for _, driver := range []string{"mysql", "pgx", "sqlite"} {
		t.Run(driver, func(t *testing.T) {
			d := dialect.NewDBDialect(driver)
			query, args, err := d.FinalizeSelect(recurringRunReplayQuery(d.QuoteIdentifier, jobID, name))
			require.NoError(t, err)
			require.ElementsMatch(t, []any{jobID, name, model.JobResourceType, jobID, model.RunResourceType, name}, args)
			require.NotContains(t, query, jobID)
			require.NotContains(t, query, name)
			if driver == "pgx" {
				require.NotContains(t, query, "?")
				placeholders := regexp.MustCompile(`\$\d+`).FindAllString(query, -1)
				require.Len(t, placeholders, len(args))
				for i, placeholder := range placeholders {
					require.Equal(t, fmt.Sprintf("$%d", i+1), placeholder)
				}
			} else {
				require.Equal(t, len(args), strings.Count(query, "?"))
			}
		})
	}
}

func TestRecurringRunReplayUsesScheduleIndexes(t *testing.T) {
	db, d := NewFakeDBOrFatal()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	query, args, err := d.FinalizeSelect(recurringRunReplayQuery(d.QuoteIdentifier, "target", "missing"))
	require.NoError(t, err)
	rows, err := db.Query("EXPLAIN QUERY PLAN "+query, args...)
	require.NoError(t, err)
	defer rows.Close()
	var details []string
	for rows.Next() {
		var id, parent, unused int
		var detail string
		require.NoError(t, rows.Scan(&id, &parent, &unused, &detail))
		details = append(details, detail)
	}
	require.NoError(t, rows.Err())
	plan := strings.Join(details, "\n")
	t.Log(plan)
	require.Contains(t, plan, "SEARCH run_details USING INDEX idx_run_details_job_uuid (JobUUID=?)")
	require.Contains(t, plan, "SEARCH resource_references USING INDEX referencefilter (ResourceType=? AND ReferenceUUID=? AND ReferenceType=?)")
	require.Regexp(t, `SEARCH run_details USING INDEX \S+ \(UUID=\?\)`, plan)
	require.NotContains(t, plan, "SCAN run_details")
	require.NotContains(t, plan, "SCAN resource_references")
}

func recurringRunReplayWithAssociation(q dialect.QuoteFunction, jobID, name string) sq.SelectBuilder {
	return sq.Select(q("UUID")).From(q("run_details")).
		Where(recurringRunAssociation(q, jobID)).Where(sq.Eq{q("DisplayName"): name}).
		OrderBy(q("CreatedAtInSec")+" DESC", q("UUID")+" DESC").Limit(1)
}

// BenchmarkRecurringRunReplayHistory keeps schedule size fixed while unrelated
// completed history grows. Setup is outside the timed query execution loop.
func BenchmarkRecurringRunReplayHistory(b *testing.B) {
	for _, historySize := range []int{0, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("history-%d", historySize), func(b *testing.B) {
			db, err := sql.Open("sqlite3", ":memory:")
			require.NoError(b, err)
			b.Cleanup(func() { require.NoError(b, db.Close()) })
			db.SetMaxOpenConns(1)
			for _, statement := range []string{
				`CREATE TABLE run_details (UUID text PRIMARY KEY, JobUUID text, DisplayName text, CreatedAtInSec bigint, State text)`,
				`CREATE INDEX idx_run_details_job_uuid ON run_details (JobUUID)`,
				`CREATE TABLE resource_references (ResourceUUID text, ResourceType text, ReferenceUUID text, ReferenceType text, PRIMARY KEY (ResourceUUID, ResourceType, ReferenceType))`,
				`CREATE INDEX referencefilter ON resource_references (ResourceType, ReferenceUUID, ReferenceType)`,
				`INSERT INTO run_details VALUES ('direct', 'target', 'hit', 1, 'SUCCEEDED'), ('legacy', NULL, 'hit', 2, 'SUCCEEDED')`,
				`INSERT INTO resource_references VALUES ('legacy', 'Run', 'target', 'Job')`,
			} {
				_, err := db.Exec(statement)
				require.NoError(b, err)
			}
			if historySize > 0 {
				_, err = db.Exec(`WITH RECURSIVE history(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM history WHERE n < ?)
					INSERT INTO run_details SELECT 'history-' || n, CASE WHEN n % 2 = 0 THEN '' ELSE NULL END,
					CASE WHEN n % 3 = 0 THEN 'hit' WHEN n % 3 = 1 THEN 'miss' ELSE 'other' END, n, 'SUCCEEDED' FROM history`, historySize)
				require.NoError(b, err)
			}
			d := dialect.NewDBDialect("sqlite")
			for _, queryCase := range []struct {
				name  string
				build func(dialect.QuoteFunction, string, string) sq.SelectBuilder
			}{
				{"or-exists", recurringRunReplayWithAssociation},
				{"schedule-scoped", recurringRunReplayQuery},
			} {
				for _, key := range []string{"hit", "miss"} {
					b.Run(queryCase.name+"/"+key, func(b *testing.B) {
						query, args, err := d.FinalizeSelect(queryCase.build(d.QuoteIdentifier, "target", key))
						require.NoError(b, err)
						var id string
						err = db.QueryRow(query, args...).Scan(&id)
						if key == "hit" {
							require.NoError(b, err)
							require.Equal(b, "legacy", id)
						} else {
							require.ErrorIs(b, err, sql.ErrNoRows)
						}
						b.ReportAllocs()
						b.ResetTimer()
						for i := 0; i < b.N; i++ {
							if err := db.QueryRow(query, args...).Scan(&id); err != nil && err != sql.ErrNoRows {
								b.Fatal(err)
							}
						}
					})
				}
			}
		})
	}
}
