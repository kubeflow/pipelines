// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func reportAtGeneration(id string, enabled bool, generation int64) *util.ScheduledWorkflow {
	return util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
		TypeMeta:   metav1.TypeMeta{APIVersion: "kubeflow.org/v1beta1", Kind: "ScheduledWorkflow"},
		ObjectMeta: metav1.ObjectMeta{UID: types.UID(id), Name: id, Namespace: "test", Generation: generation},
		Spec:       swfapi.ScheduledWorkflowSpec{Enabled: enabled},
	})
}

// These tests use the same disposable databases and independent connections as
// the recurring-run claim tests; the backend presubmit supplies both DSNs.
func TestRecurringRunReportProductionDatabases(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d := recurringIntegrationDatabases(t, driver)
			q, qb := d.QuoteIdentifier, d.QueryBuilder()
			stores := make([]*JobStore, len(dbs))
			for i, db := range dbs {
				stores[i] = NewJobStore(db, util.NewFakeTimeForEpoch(), nil, d)
			}
			createJob := func(t *testing.T, id string) {
				t.Helper()
				_, err := stores[0].CreateJob(&model.Job{UUID: id, K8SName: id, DisplayName: id, Namespace: "test", Enabled: true})
				require.NoError(t, err)
			}
			generation := func(t *testing.T, id string) int64 {
				t.Helper()
				query, args, err := qb.Select(q("Generation")).From(q("recurring_run_report_states")).Where(sq.Eq{q("JobUUID"): id}).ToSql()
				require.NoError(t, err)
				var value int64
				require.NoError(t, dbs[0].QueryRow(query, args...).Scan(&value))
				return value
			}
			for _, initiallyAbsent := range []bool{true, false} {
				for _, apiWins := range []bool{true, false} {
					t.Run(fmt.Sprintf("concurrent/absent-%v/api-wins-%v", initiallyAbsent, apiWins), func(t *testing.T) {
						id := fmt.Sprintf("concurrent-%v-%v", initiallyAbsent, apiWins)
						createJob(t, id)
						if initiallyAbsent {
							query, args, err := qb.Delete(q("recurring_run_report_states")).Where(sq.Eq{q("JobUUID"): id}).ToSql()
							require.NoError(t, err)
							_, err = dbs[0].Exec(query, args...)
							require.NoError(t, err)
						} else {
							require.NoError(t, stores[0].ChangeJobModeAtGeneration(id, true, 1))
						}
						errs := recurringConcurrent(len(stores), func(i int) error {
							version := int64(i + 2)
							enabled := i != len(stores)-1
							if (i%2 == (len(stores)-1)%2) == apiWins {
								return stores[i].ChangeJobModeAtGeneration(id, enabled, version)
							}
							return stores[i].UpdateJob(reportAtGeneration(id, enabled, version))
						})
						for _, err := range errs {
							require.NoError(t, err)
						}
						latest := int64(len(stores) + 1)
						require.Equal(t, latest, generation(t, id))
						before, err := stores[0].GetJob(id)
						require.NoError(t, err)
						require.False(t, before.Enabled)
						// Legacy reports cannot reopen a schedule after its first
						// versioned report or API mode change, even in the same second.
						for _, stale := range []int64{0, 1, latest - 1} {
							require.NoError(t, stores[1].UpdateJob(reportAtGeneration(id, true, stale)))
							require.NoError(t, stores[2].ChangeJobModeAtGeneration(id, true, stale))
						}
						after, err := stores[0].GetJob(id)
						require.NoError(t, err)
						require.Equal(t, before, after)
						require.Equal(t, latest, generation(t, id))
					})
				}
			}
			t.Run("concurrent-first-reports-for-distinct-jobs", func(t *testing.T) {
				for i := range stores {
					createJob(t, fmt.Sprintf("distinct-%d", i))
				}
				errs := recurringConcurrent(len(stores), func(i int) error {
					id := fmt.Sprintf("distinct-%d", i)
					if i%2 == 0 {
						return stores[i].ChangeJobModeAtGeneration(id, false, 1)
					}
					return stores[i].UpdateJob(reportAtGeneration(id, false, 1))
				})
				for i, err := range errs {
					require.NoError(t, err)
					id := fmt.Sprintf("distinct-%d", i)
					require.Equal(t, int64(1), generation(t, id))
					job, err := stores[0].GetJob(id)
					require.NoError(t, err)
					require.False(t, job.Enabled)
				}
			})
			t.Run("blocked-report-observes-committed-generation", func(t *testing.T) {
				const id = "blocked-report"
				createJob(t, id)
				require.NoError(t, stores[0].ChangeJobModeAtGeneration(id, true, 1))
				var sessionID int64
				sessionQuery := "SELECT CONNECTION_ID()"
				if driver == "pgx" {
					sessionQuery = "SELECT pg_backend_pid()"
				}
				require.NoError(t, dbs[1].QueryRow(sessionQuery).Scan(&sessionID))
				tx, err := dbs[0].Begin()
				require.NoError(t, err)
				defer tx.Rollback()
				for _, update := range []sq.UpdateBuilder{
					qb.Update(q("jobs")).Set(q("Enabled"), false).Where(sq.Eq{q("UUID"): id}),
					qb.Update(q("recurring_run_report_states")).Set(q("Generation"), 3).Where(sq.Eq{q("JobUUID"): id}),
				} {
					query, args, err := update.ToSql()
					require.NoError(t, err)
					_, err = tx.Exec(query, args...)
					require.NoError(t, err)
				}
				result := make(chan error, 1)
				go func() { result <- stores[1].UpdateJob(reportAtGeneration(id, true, 2)) }()
				recurringWaitForLock(t, dbs[2], driver, sessionID)
				require.NoError(t, tx.Commit())
				select {
				case err := <-result:
					require.NoError(t, err)
				case <-time.After(10 * time.Second):
					t.Fatal("report did not finish after the mode change committed")
				}
				job, err := stores[0].GetJob(id)
				require.NoError(t, err)
				require.False(t, job.Enabled)
				require.Equal(t, int64(3), generation(t, id))
			})
			t.Run("unchanged-mode-advances-fence-and-equal-generation-reports-status", func(t *testing.T) {
				const id = "unchanged-mode"
				createJob(t, id)
				require.NoError(t, stores[0].ChangeJobModeAtGeneration(id, true, 8))
				require.Equal(t, int64(8), generation(t, id))
				require.NoError(t, stores[1].UpdateJob(reportAtGeneration(id, false, 7)))
				report := reportAtGeneration(id, true, 8)
				report.Status.Conditions = []swfapi.ScheduledWorkflowCondition{{Type: swfapi.ScheduledWorkflowEnabled, Status: corev1.ConditionTrue}}
				require.NoError(t, stores[2].UpdateJob(report))
				require.NoError(t, stores[3].UpdateJob(report))
				job, err := stores[0].GetJob(id)
				require.NoError(t, err)
				require.True(t, job.Enabled)
				require.Equal(t, "ENABLED", job.Conditions)
				require.Equal(t, int64(8), generation(t, id))
			})
			t.Run("failed-job-write-rolls-back-generation", func(t *testing.T) {
				const id = "rollback-report"
				createJob(t, id)
				require.NoError(t, stores[0].ChangeJobModeAtGeneration(id, true, 1))
				before, err := stores[0].GetJob(id)
				require.NoError(t, err)
				if driver == "mysql" {
					_, err = dbs[0].Exec("CREATE TRIGGER fail_report_job BEFORE UPDATE ON jobs FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT = 'simulated report job failure'")
				} else {
					_, err = dbs[0].Exec(`CREATE FUNCTION fail_report_job() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'simulated report job failure'; END $$`)
					require.NoError(t, err)
					_, err = dbs[0].Exec(`CREATE TRIGGER fail_report_job BEFORE UPDATE ON jobs FOR EACH ROW EXECUTE FUNCTION fail_report_job()`)
				}
				require.NoError(t, err)
				t.Cleanup(func() {
					drop := "DROP TRIGGER fail_report_job"
					if driver == "pgx" {
						drop += " ON jobs"
					}
					_, err := dbs[0].Exec(drop)
					require.NoError(t, err)
				})
				require.ErrorContains(t, stores[1].UpdateJob(reportAtGeneration(id, false, 2)), "simulated report job failure")
				require.ErrorContains(t, stores[2].ChangeJobModeAtGeneration(id, false, 3), "simulated report job failure")
				after, err := stores[0].GetJob(id)
				require.NoError(t, err)
				require.Equal(t, before, after)
				require.Equal(t, int64(1), generation(t, id))
			})
			t.Run("missing-workflow-tombstone", func(t *testing.T) {
				const id = "missing-workflow"
				createJob(t, id)
				require.NoError(t, stores[0].ChangeJobModeAtGeneration(id, true, 5))
				require.NoError(t, stores[0].ChangeJobModeAtGeneration(id, false, -1))
				for _, version := range []int64{0, 5, 100} {
					require.NoError(t, stores[1].UpdateJob(reportAtGeneration(id, true, version)))
					require.NoError(t, stores[2].ChangeJobModeAtGeneration(id, true, version))
				}
				job, err := stores[0].GetJob(id)
				require.NoError(t, err)
				require.False(t, job.Enabled)
				require.Equal(t, int64(-1), generation(t, id))
			})
		})
	}
}

type reportMigrationLogger struct {
	logger.Interface
	mu  sync.Mutex
	ddl []string
}

func (l *reportMigrationLogger) Trace(ctx context.Context, started time.Time, sql func() (string, int64), err error) {
	query, _ := sql()
	fields := strings.Fields(query)
	if len(fields) == 0 {
		return
	}
	verb := strings.ToUpper(fields[0])
	if verb == "CREATE" || verb == "ALTER" || verb == "DROP" {
		l.mu.Lock()
		l.ddl = append(l.ddl, query)
		l.mu.Unlock()
	}
	l.Interface.Trace(ctx, started, sql, err)
}

func TestRecurringRunReportStateUpgradeProductionDatabases(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			dbs, d := recurringIntegrationDatabases(t, driver)
			q, qb := d.QuoteIdentifier, d.QueryBuilder()
			store := NewJobStore(dbs[1], util.NewFakeTimeForEpoch(), nil, d)
			for _, id := range []string{"locked", "writable"} {
				_, err := store.CreateJob(&model.Job{UUID: id, K8SName: id, DisplayName: id, Namespace: "test", Enabled: true,
					PipelineSpec: model.PipelineSpec{RuntimeConfig: model.RuntimeConfig{Parameters: `{"input":"retained"}`, PipelineRoot: "s3://retained"}}})
				require.NoError(t, err)
			}
			var dialector gorm.Dialector
			if driver == "mysql" {
				// Match the schema created by recurringIntegrationDatabases.
				dialector = mysql.New(mysql.Config{Conn: dbs[0]})
			} else {
				dialector = postgres.New(postgres.Config{Conn: dbs[0]})
			}
			orm, err := gorm.Open(dialector, &gorm.Config{})
			require.NoError(t, err)
			// Removing only the unused new table recreates a populated
			// pre-upgrade schema without modifying the established jobs table.
			var count int64
			require.NoError(t, orm.Model(&model.RecurringRunReportState{}).Count(&count).Error)
			require.Zero(t, count)
			require.NoError(t, orm.Migrator().DropTable(&model.RecurringRunReportState{}))
			// Compare the actual AllModels reconciliation with its pre-change
			// baseline. PostgreSQL currently repeats unrelated null-default
			// ALTERs; they must not be mistaken for changes introduced here.
			var previousModels []any
			for _, entry := range model.AllModels() {
				if _, added := entry.(*model.RecurringRunReportState); !added {
					previousModels = append(previousModels, entry)
				}
			}
			baselineTrace := &reportMigrationLogger{Interface: logger.Default.LogMode(logger.Silent)}
			require.NoError(t, orm.Session(&gorm.Session{Logger: baselineTrace}).AutoMigrate(previousModels...))
			baselineDDL := append([]string(nil), baselineTrace.ddl...)
			if len(baselineDDL) > 0 {
				t.Logf("Existing schema reconciliation DDL (not introduced by the report-state table): %v", baselineDDL)
			}
			before, err := store.GetJob("locked")
			require.NoError(t, err)
			tx, err := dbs[2].Begin()
			require.NoError(t, err)
			defer tx.Rollback()
			query, args, err := qb.Update(q("jobs")).Set(q("Description"), "writer survived upgrade").Where(sq.Eq{q("UUID"): "locked"}).ToSql()
			require.NoError(t, err)
			_, err = tx.Exec(query, args...)
			require.NoError(t, err)
			trace := &reportMigrationLogger{Interface: logger.Default.LogMode(logger.Silent)}
			migration := orm.Session(&gorm.Session{Logger: trace})
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- migration.WithContext(ctx).AutoMigrate(model.AllModels()...) }()
			// Existing reads and unrelated writes must remain usable while a
			// jobs transaction is open and schema reconciliation runs.
			visible, err := store.GetJob("locked")
			require.NoError(t, err)
			require.Equal(t, before, visible)
			query, args, err = qb.Update(q("jobs")).Set(q("Description"), "concurrent writer").Where(sq.Eq{q("UUID"): "writable"}).ToSql()
			require.NoError(t, err)
			_, err = dbs[3].ExecContext(ctx, query, args...)
			require.NoError(t, err)
			select {
			case err := <-result:
				require.NoError(t, err, "new table creation must not wait for the open jobs writer")
			case <-ctx.Done():
				t.Fatal("schema reconciliation waited for an existing jobs transaction")
			}
			require.NoError(t, tx.Commit())
			after, err := store.GetJob("locked")
			require.NoError(t, err)
			before.Description = "writer survived upgrade"
			require.Equal(t, before, after)
			writable, err := store.GetJob("writable")
			require.NoError(t, err)
			require.Equal(t, "concurrent writer", writable.Description)
			require.True(t, orm.Migrator().HasTable(&model.RecurringRunReportState{}))
			require.NoError(t, orm.Model(&model.RecurringRunReportState{}).Count(&count).Error)
			require.Zero(t, count, "startup must not backfill report generations")
			trace.mu.Lock()
			ddl := append([]string(nil), trace.ddl...)
			trace.mu.Unlock()
			var addedDDL, existingDDL []string
			for _, statement := range ddl {
				require.NotContains(t, statement, q("jobs"), "the upgrade must not alter or reference the populated jobs table")
				if strings.Contains(statement, "recurring_run_report_states") {
					addedDDL = append(addedDDL, statement)
				} else {
					existingDDL = append(existingDDL, statement)
				}
			}
			require.Equal(t, baselineDDL, existingDDL, "existing-table reconciliation must match the pre-change baseline")
			require.Len(t, addedDDL, 1, "the only added DDL must create the new table")
			require.True(t, strings.HasPrefix(strings.ToUpper(addedDDL[0]), "CREATE TABLE"), addedDDL[0])
			require.NotContains(t, strings.ToUpper(addedDDL[0]), "REFERENCES", "the new table must not add foreign-key dependencies")
			t.Logf("Added report-state DDL: %s", addedDDL[0])
			// A second startup adds no DDL beyond the existing baseline and
			// preserves lazily initialized generation state.
			require.NoError(t, store.ChangeJobModeAtGeneration("locked", false, 7))
			require.NoError(t, migration.AutoMigrate(model.AllModels()...))
			var state model.RecurringRunReportState
			require.NoError(t, orm.First(&state, map[string]any{"JobUUID": "locked"}).Error)
			require.Equal(t, int64(7), state.Generation)
			trace.mu.Lock()
			defer trace.mu.Unlock()
			require.Equal(t, append(ddl, baselineDDL...), trace.ddl)
		})
	}
}
