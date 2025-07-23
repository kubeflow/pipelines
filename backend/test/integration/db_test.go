// Copyright 2023 The Kubeflow Authors
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

package integration

import (
	"database/sql"
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	_ "github.com/jackc/pgx/v5/stdlib"
	cm "github.com/kubeflow/pipelines/backend/src/apiserver/client_manager"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DBTestSuite struct {
	suite.Suite
}

// Skip if it's not integration test running.
func (s *DBTestSuite) SetupTest() {
	if !*runIntegrationTests {
		s.T().SkipNow()
		return
	}
}

// Test MySQL initializes correctly
func (s *DBTestSuite) TestInitDBClient_MySQL_Fresh() {
	if *runPostgreSQLTests {
		s.T().SkipNow()
		return
	}
	t := s.T()
	viper.Set("DBDriverName", "mysql")
	viper.Set("DBConfig.MySQLConfig.DBName", "mlpipeline")
	// The default port-forwarding IP address that test uses is different compared to production
	viper.Set("DBConfig.MySQLConfig.Host", "localhost")
	duration, _ := time.ParseDuration("1m")
	db := cm.InitDBClient(duration)
	assert.NotNil(t, db)

	// Extract underlying *sql.DB and wrap in gorm.DB
	sqlDB := db.DB
	dialector := mysql.New(mysql.Config{Conn: sqlDB})
	gdb, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm.DB for MySQL: %v", err)
	}

	verifyAddDisplayNameColumn(t, gdb, "mysql", model.Pipeline{}.TableName())
	verifyAddDisplayNameColumn(t, gdb, "mysql", model.PipelineVersion{}.TableName())

	verifyLengths(t, gdb)
	verifyIndexesMySQL(t, gdb)
	verifyConstraintsMySQL(t, gdb)
}

// Test PostgreSQL initializes correctly
func (s *DBTestSuite) TestInitDBClient_PostgreSQL_Fresh() {
	if !*runPostgreSQLTests {
		s.T().SkipNow()
		return
	}
	t := s.T()
	viper.Set("DBDriverName", "pgx")
	viper.Set("DBConfig.PostgreSQLConfig.DBName", "mlpipeline")
	// The default port-forwarding IP address that test uses is different compared to production
	viper.Set("DBConfig.PostgreSQLConfig.Host", "127.0.0.3")
	viper.Set("DBConfig.PostgreSQLConfig.User", "user")
	viper.Set("DBConfig.PostgreSQLConfig.Password", "password")
	duration, _ := time.ParseDuration("1m")
	db := cm.InitDBClient(duration)
	assert.NotNil(t, db)

	// Extract underlying *sql.DB and wrap in gorm.DB
	sqlDB := db.DB
	dialector := postgres.New(postgres.Config{Conn: sqlDB})
	gdb, err := gorm.Open(dialector, &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open gorm.DB for PostgreSQL: %v", err)
	}

	verifyAddDisplayNameColumn(t, gdb, "pgx", model.Pipeline{}.TableName())
	verifyAddDisplayNameColumn(t, gdb, "pgx", model.PipelineVersion{}.TableName())
}

// verifyAddDisplayNameColumn tests that the DisplayName column exists in allowed tables
// and is not added to unauthorized tables.
func verifyAddDisplayNameColumn(t *testing.T, db *gorm.DB, driverName string, tableName string) {
	// Use the table name passed in for testing.

	// The column should already exist after InitDBClient runs migration.
	hasCol := db.Migrator().HasColumn(tableName, "DisplayName")
	require.True(t, hasCol, "DisplayName column should exist in %s table after initialization", tableName)

	// Calling again should be idempotent: it should not return an error
	err := cm.AddDisplayNameColumn(db, tableName, driverName)
	require.NoError(t, err, "repeated call to AddDisplayNameColumn on allowed table should not error")

	// It fails on a non-whitelisted table.
	err = cm.AddDisplayNameColumn(db, "unauthorized_table", driverName)
	require.Error(t, err)
}

// verifyLengths checks that all columns we shrank from GORM v2 to v1 have the expected VARCHAR length in MySQL.
func verifyLengths(t *testing.T, db *gorm.DB) {
	for _, s := range cm.LengthSpecs() {
		tbl, col, _ := cm.FieldMeta(db, s.Model, s.Field)
		got := getColumnLengthMySQL(t, db, tbl, col) // compare with information_schema
		require.Equal(t, s.Max, got)
	}
}

// getColumnLengthMySQL returns CHARACTER_MAXIMUM_LENGTH for the given MySQL table/column.
func getColumnLengthMySQL(t *testing.T, db *gorm.DB, table, col string) int {
	t.Helper()
	var l sql.NullInt64
	err := db.Raw(`
		SELECT CHARACTER_MAXIMUM_LENGTH
		FROM information_schema.columns
		WHERE table_schema = DATABASE()
		AND table_name = ?
		AND column_name = ?`, table, col).Scan(&l).Error
	require.NoErrorf(t, err, "query column length failed for %s.%s", table, col)
	if !l.Valid {
		t.Fatalf("no length returned for %s.%s", table, col)
	}
	return int(l.Int64)
}

// ---- Single source of truth for composite index naming ----

type IndexSpec struct {
	Table string
	Name  string
}

var indexSpecs = []IndexSpec{
	// Composite unique indexes
	{"pipelines", "namespace_name"},
	{"pipeline_versions", "idx_pipelineid_name"},
	{"experiments", "idx_name_namespace"},

	// non-unique composite indexes
	{"resource_references", "referencefilter"},
	{"run_details", "experimentuuid_createatinsec"},
	{"run_details", "experimentuuid_conditions_finishedatinsec"},
	{"run_details", "namespace_createatinsec"},
	{"run_details", "namespace_conditions_finishedatinsecc"},
	{"pipeline_versions", "idx_pipeline_versions_CreatedAtInSec"},
	{"pipeline_versions", "idx_pipeline_versions_PipelineId"},
	{"tasks", "tasks_RunUUID_run_details_UUID_foreign"},
}

// verifyIndexesMySQL ensures expected index names exist in MySQL using information_schema only.
func verifyIndexesMySQL(t *testing.T, db *gorm.DB) {
	for _, spec := range indexSpecs {
		nameSet := fetchIndexNamesMySQL(t, db, spec.Table)
		assert.Containsf(t, nameSet, spec.Name, "index %s should exist on %s", spec.Name, spec.Table)
	}
}

func fetchIndexNamesMySQL(t *testing.T, db *gorm.DB, table string) map[string]struct{} {
	t.Helper()
	type row struct {
		IndexName string
	}
	var rows []row
	err := db.Raw(`
		SELECT INDEX_NAME
		FROM information_schema.statistics
		WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?
		GROUP BY INDEX_NAME`, table).Scan(&rows).Error
	require.NoError(t, err)
	out := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		out[r.IndexName] = struct{}{}
	}
	return out
}

// ---- Single source of truth for foreign-key constraints ----
type ConstraintSpec struct {
	Table string
	Name  string
}

var constraintSpecs = []ConstraintSpec{
	{"tasks", "tasks_RunUUID_run_details_UUID_foreign"},
	{"run_metrics", "run_metrics_RunUUID_run_details_UUID_foreign"},
	{"pipeline_versions", "pipeline_versions_PipelineId_pipelines_UUID_foreign"},
}

// verifyConstraintsMySQL checks that critical foreign key constraint names exist via information_schema only.
func verifyConstraintsMySQL(t *testing.T, db *gorm.DB) {
	for _, spec := range constraintSpecs {
		nameSet := fetchFKNamesMySQL(t, db, spec.Table)
		assert.Containsf(t, nameSet, spec.Name, "foreign key %s should exist on %s", spec.Name, spec.Table)
	}
}

func fetchFKNamesMySQL(t *testing.T, db *gorm.DB, table string) map[string]struct{} {
	t.Helper()
	type row struct {
		Name string
	}
	var rows []row
	err := db.Raw(`
	SELECT CONSTRAINT_NAME AS Name
	FROM information_schema.TABLE_CONSTRAINTS
	WHERE TABLE_SCHEMA = DATABASE()
	AND TABLE_NAME = ?
	AND CONSTRAINT_TYPE = 'FOREIGN KEY'`, table).Scan(&rows).Error
	require.NoError(t, err)
	out := make(map[string]struct{}, len(rows))
	for _, r := range rows {
		out[r.Name] = struct{}{}
	}
	return out
}

func TestDB(t *testing.T) {
	suite.Run(t, new(DBTestSuite))
}
