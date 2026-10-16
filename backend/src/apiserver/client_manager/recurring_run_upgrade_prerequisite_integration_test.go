// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0

package clientmanager

import (
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestRecurringRunUpgradePrerequisiteProductionDatabases(t *testing.T) {
	for _, driver := range []string{"mysql", "pgx"} {
		t.Run(driver, func(t *testing.T) {
			db := prerequisiteProductionDatabase(t, driver)
			require.NoError(t, requireCompletedRecurringRunAdoption(db))
			require.NoError(t, db.AutoMigrate(&prerequisiteJob{}))
			require.NoError(t, requireCompletedRecurringRunAdoption(db))
			require.NoError(t, db.Create(&prerequisiteJob{UUID: "legacy", Enabled: false}).Error)
			require.ErrorContains(t, requireCompletedRecurringRunAdoption(db), "no API-owned scheduling state schema")
			require.False(t, db.Migrator().HasTable(&model.RecurringRunAdoption{}))
			require.NoError(t, db.Delete(&prerequisiteJob{}, map[string]any{"UUID": "legacy"}).Error)
			prerequisiteSchema(t, db)
			prerequisiteFixture(t, db, "adopted", false)
			require.NoError(t, requireCompletedRecurringRunAdoption(db))
			require.NoError(t, db.Model(&model.RecurringRunAdoption{}).Where(map[string]any{"ID": recurringRunAdoptionSourceID + ":adopted"}).Update("Ready", false).Error)
			require.NoError(t, requireCompletedRecurringRunAdoption(db))
			require.NoError(t, db.Model(&model.RecurringRunAdoption{}).Where(map[string]any{"ID": recurringRunAdoptionSourceID + ":adopted"}).Update("Ready", true).Error)
			require.NoError(t, db.Delete(&model.RecurringRunState{}, map[string]any{"JobUUID": "adopted"}).Error)
			require.ErrorContains(t, requireCompletedRecurringRunAdoption(db), "missing scheduling state")
		})
	}
}

// Each engine test owns a fresh database/schema and never migrates the supplied
// administrative database. These DSNs are provided by the backend CI services.
func prerequisiteProductionDatabase(t *testing.T, driver string) *gorm.DB {
	t.Helper()
	env := "KFP_RECURRING_MYSQL_TEST_DSN"
	if driver == "pgx" {
		env = "KFP_RECURRING_POSTGRES_TEST_DSN"
	}
	dsn := os.Getenv(env)
	if dsn == "" {
		t.Skip("set " + env + " for production database prerequisite coverage")
	}
	name := fmt.Sprintf("kfp_upgrade_%d", time.Now().UnixNano())
	q := dialect.NewDBDialect(driver).QuoteIdentifier
	var admin, connection *sql.DB
	var create, drop string
	var dialector gorm.Dialector
	if driver == "mysql" {
		config, err := mysqldriver.ParseDSN(dsn)
		require.NoError(t, err)
		config.DBName = ""
		admin, err = sql.Open(driver, config.FormatDSN())
		require.NoError(t, err)
		create = "CREATE DATABASE " + q(name) + " CHARACTER SET utf8mb4"
		drop = "DROP DATABASE " + q(name)
		config.DBName = name
		connection, err = sql.Open(driver, config.FormatDSN())
		require.NoError(t, err)
		dialector = mysql.New(mysql.Config{Conn: connection})
	} else {
		config, err := pgx.ParseConfig(dsn)
		require.NoError(t, err)
		admin = stdlib.OpenDB(*config)
		create = "CREATE SCHEMA " + q(name)
		drop = "DROP SCHEMA " + q(name) + " CASCADE"
		config.RuntimeParams["search_path"] = name
		connection = stdlib.OpenDB(*config)
		dialector = postgres.New(postgres.Config{Conn: connection})
	}
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	_, err := admin.Exec(create)
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec(drop); require.NoError(t, err) })
	t.Cleanup(func() { require.NoError(t, connection.Close()) })
	db, err := gorm.Open(dialector, &gorm.Config{})
	require.NoError(t, err)
	return db
}
