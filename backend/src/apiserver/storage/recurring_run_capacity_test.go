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
	"github.com/stretchr/testify/require"
)

func TestRecurringRunActiveCount(t *testing.T) {
	db, d := NewFakeDBOrFatal()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	testRecurringRunActiveCount(t, db, d)
}

func testRecurringRunActiveCount(t *testing.T, db *sql.DB, d dialect.DBDialect) {
	orm, err := OpenTransferDB(db, d.Name())
	require.NoError(t, err)
	const jobID = "capacity-target"
	insert := func(t *testing.T, id string, rawJob, state any, conditions string, reference *model.ResourceReference) {
		t.Helper()
		run := &model.Run{UUID: "capacity-" + id, DisplayName: id}
		require.NoError(t, orm.Create(run).Error)
		// GORM's default:null would otherwise make empty strings indistinguishable from NULL.
		require.NoError(t, orm.Model(run).Updates(map[string]any{
			"JobUUID": rawJob, "State": state, "Conditions": conditions,
		}).Error)
		if reference != nil {
			reference.ResourceUUID = run.UUID
			require.NoError(t, orm.Create(reference).Error)
		}
	}
	count := func(t *testing.T, want int64) {
		t.Helper()
		query, args, err := d.FinalizeSelect(recurringRunActiveCount(d.QuoteIdentifier, jobID))
		require.NoError(t, err)
		var got int64
		require.NoError(t, db.QueryRow(query, args...).Scan(&got))
		require.Equal(t, want, got)
	}
	count(t, 0)
	var want int64
	for _, tc := range []struct {
		name                        string
		rawJob, state               any
		conditions, referenceJob    string
		resourceType, referenceType model.ResourceType
		active                      bool
	}{
		{name: "direct", rawJob: jobID, state: "RUNNING", active: true},
		{name: "empty-job-reference", rawJob: "", state: "RUNNING", referenceJob: jobID, active: true},
		{name: "null-job-reference", state: nil, conditions: "Running", referenceJob: jobID, active: true},
		{name: "direct-and-matching-reference", rawJob: jobID, state: "RUNNING", referenceJob: jobID, active: true},
		{name: "direct-overrides-foreign-reference", rawJob: jobID, state: "RUNNING", referenceJob: "other", active: true},
		{name: "foreign-direct-overrides-reference", rawJob: "other", state: "RUNNING", referenceJob: jobID},
		{name: "foreign-reference", rawJob: "", state: "RUNNING", referenceJob: "other"},
		{name: "wrong-resource-type", state: "RUNNING", referenceJob: jobID, resourceType: model.JobResourceType},
		{name: "wrong-reference-type", state: "RUNNING", referenceJob: jobID, referenceType: model.ExperimentResourceType},
		{name: "unassociated", rawJob: "", state: "RUNNING"},
		{name: "active-state-overrides-terminal-condition", rawJob: jobID, state: "RUNNING", conditions: "Succeeded", active: true},
		{name: "reference-active-state-overrides-terminal-condition", state: "RUNNING", conditions: "Succeeded", referenceJob: jobID, active: true},
		{name: "unknown-state-is-active", rawJob: "", state: "", referenceJob: jobID, active: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var reference *model.ResourceReference
			if tc.referenceJob != "" {
				if tc.resourceType == "" {
					tc.resourceType = model.RunResourceType
				}
				if tc.referenceType == "" {
					tc.referenceType = model.JobResourceType
				}
				reference = &model.ResourceReference{ResourceType: tc.resourceType, ReferenceType: tc.referenceType, ReferenceUUID: tc.referenceJob}
			}
			insert(t, tc.name, tc.rawJob, tc.state, tc.conditions, reference)
			if tc.active {
				want++
			}
			count(t, want)
		})
	}
	for _, ownership := range []struct {
		name   string
		rawJob any
	}{
		{"direct", jobID}, {"empty-job", ""}, {"null-job", nil},
	} {
		for stateIndex, state := range terminalRunStateStrings {
			for _, source := range []struct {
				name       string
				state      any
				conditions string
			}{
				{"state", state, "Running"},
				{"empty-state-fallback", "", state},
				{"null-state-fallback", nil, state},
			} {
				name := fmt.Sprintf("%s-%d-%s-%s", ownership.name, stateIndex, state, source.name)
				t.Run(name, func(t *testing.T) {
					insert(t, name, ownership.rawJob, source.state, source.conditions, &model.ResourceReference{
						ResourceType: model.RunResourceType, ReferenceType: model.JobResourceType, ReferenceUUID: jobID,
					})
					count(t, want)
				})
			}
		}
	}
}

func TestRecurringRunActiveCountQueryDialects(t *testing.T) {
	const jobID = "job' OR 1=1 --"
	for _, name := range []string{"mysql", "pgx", "sqlite"} {
		t.Run(name, func(t *testing.T) {
			d := dialect.NewDBDialect(name)
			q := d.QuoteIdentifier
			query, args, err := d.FinalizeSelect(recurringRunActiveCount(q, jobID))
			require.NoError(t, err)
			for table, columns := range map[string][]string{
				"run_details":         {"UUID", "JobUUID", "State", "Conditions"},
				"resource_references": {"ResourceUUID", "ResourceType", "ReferenceUUID", "ReferenceType"},
			} {
				for _, column := range columns {
					require.Contains(t, query, q(table)+"."+q(column))
				}
			}
			expectedArgs := []any{jobID}
			for _, state := range terminalRunStateStrings {
				expectedArgs = append(expectedArgs, state)
			}
			expectedArgs = append(expectedArgs, model.JobResourceType, jobID, model.RunResourceType)
			for _, state := range terminalRunStateStrings {
				expectedArgs = append(expectedArgs, state)
			}
			require.Equal(t, expectedArgs, args)
			require.NotContains(t, query, jobID)
			if name == "pgx" {
				require.NotContains(t, query, "?")
				placeholders := regexp.MustCompile(`\$\d+`).FindAllString(query, -1)
				require.Len(t, placeholders, len(args))
				for i, placeholder := range placeholders {
					require.Equal(t, fmt.Sprintf("$%d", i+1), placeholder)
				}
			} else {
				require.Equal(t, len(args), strings.Count(query, "?"))
				require.NotContains(t, query, "$1")
			}
		})
	}
}

func TestRecurringRunActiveCountUsesScheduleIndexes(t *testing.T) {
	db, d := NewFakeDBOrFatal()
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	explain := func(builder sq.SelectBuilder) string {
		t.Helper()
		query, args, err := d.FinalizeSelect(builder)
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
		return strings.Join(details, "\n")
	}
	// Keep the previous query as a diagnostic comparison, without constraining
	// future optimizers to retain its scan of unrelated history.
	t.Logf("OR/EXISTS plan:\n%s", explain(recurringRunActiveCountWithAssociation(d.QuoteIdentifier, "target")))
	plan := explain(recurringRunActiveCount(d.QuoteIdentifier, "target"))
	t.Logf("schedule-scoped plan:\n%s", plan)
	require.Contains(t, plan, "SEARCH run_details USING INDEX idx_run_details_job_uuid (JobUUID=?)")
	require.Contains(t, plan, "SEARCH resource_references USING INDEX referencefilter (ResourceType=? AND ReferenceUUID=? AND ReferenceType=?)")
	require.Regexp(t, `SEARCH run_details USING INDEX \S+ \(UUID=\?\)`, plan)
	require.NotContains(t, plan, "SCAN run_details")
	require.NotContains(t, plan, "SCAN resource_references")
}

func recurringRunActiveCountWithAssociation(q dialect.QuoteFunction, jobID string) sq.SelectBuilder {
	effectiveState := fmt.Sprintf("COALESCE(NULLIF(%s, ''), %s, '')", q("State"), q("Conditions"))
	return sq.Select("COUNT(*)").From(q("run_details")).
		Where(recurringRunAssociation(q, jobID)).Where(sq.NotEq{effectiveState: terminalRunStateStrings})
}

// BenchmarkRecurringRunActiveCountHistory measures only query execution with a
// fixed schedule and growing unrelated history. The minimal schema retains the
// production indexes used by both query forms; setup is outside the timed loop.
func BenchmarkRecurringRunActiveCountHistory(b *testing.B) {
	for _, historySize := range []int{0, 100_000, 1_000_000} {
		b.Run(fmt.Sprintf("history-%d", historySize), func(b *testing.B) {
			db, err := sql.Open("sqlite3", ":memory:")
			require.NoError(b, err)
			b.Cleanup(func() { require.NoError(b, db.Close()) })
			db.SetMaxOpenConns(1)
			for _, statement := range []string{
				`CREATE TABLE run_details (UUID text PRIMARY KEY, JobUUID text, State text, Conditions text NOT NULL)`,
				`CREATE INDEX idx_run_details_job_uuid ON run_details (JobUUID)`,
				`CREATE TABLE resource_references (ResourceUUID text, ResourceType text, ReferenceUUID text, ReferenceType text, PRIMARY KEY (ResourceUUID, ResourceType, ReferenceType))`,
				`CREATE INDEX referencefilter ON resource_references (ResourceType, ReferenceUUID, ReferenceType)`,
				`INSERT INTO run_details VALUES ('direct', 'target', 'RUNNING', ''), ('legacy-empty', '', NULL, 'Running'), ('legacy-null', NULL, 'RUNNING', ''), ('terminal', 'target', 'SUCCEEDED', '')`,
				`INSERT INTO resource_references VALUES ('direct', 'Run', 'target', 'Job'), ('legacy-empty', 'Run', 'target', 'Job'), ('legacy-null', 'Run', 'target', 'Job')`,
			} {
				_, err := db.Exec(statement)
				require.NoError(b, err)
			}
			if historySize > 0 {
				_, err = db.Exec(`WITH RECURSIVE history(n) AS (SELECT 1 UNION ALL SELECT n + 1 FROM history WHERE n < ?)
					INSERT INTO run_details SELECT 'history-' || n, CASE WHEN n % 2 = 0 THEN '' ELSE NULL END, 'SUCCEEDED', '' FROM history`, historySize)
				require.NoError(b, err)
			}
			d := dialect.NewDBDialect("sqlite")
			for _, queryCase := range []struct {
				name  string
				build func(dialect.QuoteFunction, string) sq.SelectBuilder
			}{
				{"or-exists", recurringRunActiveCountWithAssociation},
				{"schedule-scoped", recurringRunActiveCount},
			} {
				b.Run(queryCase.name, func(b *testing.B) {
					query, args, err := d.FinalizeSelect(queryCase.build(d.QuoteIdentifier, "target"))
					require.NoError(b, err)
					var count int64
					require.NoError(b, db.QueryRow(query, args...).Scan(&count))
					require.EqualValues(b, 3, count)
					b.ReportAllocs()
					b.ResetTimer()
					for i := 0; i < b.N; i++ {
						if err := db.QueryRow(query, args...).Scan(&count); err != nil {
							b.Fatal(err)
						}
					}
				})
			}
		})
	}
}
