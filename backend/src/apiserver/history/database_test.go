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

package history

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"strings"
	"testing"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Each test creates and drops only a randomly named database/schema. CI supplies
// disposable services; no pre-existing tables are used or removed.
func externalDatabase(t *testing.T, driver, dsn string) *gorm.DB {
	t.Helper()
	name := "kfp_history_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	config := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)}
	var rootDialect, childDialect gorm.Dialector
	var createSQL, dropSQL string
	if driver == "mysql" {
		parsed, err := mysqldriver.ParseDSN(dsn)
		require.NoError(t, err)
		rootDialect = mysql.Open(dsn)
		parsed.DBName = name
		childDialect = mysql.Open(parsed.FormatDSN())
		createSQL, dropSQL = "CREATE DATABASE `"+name+"`", "DROP DATABASE `"+name+"`"
	} else {
		parsed, err := url.Parse(dsn)
		require.NoError(t, err)
		require.Contains(t, []string{"postgres", "postgresql"}, parsed.Scheme)
		rootDialect = postgres.Open(dsn)
		query := parsed.Query()
		query.Set("search_path", name)
		parsed.RawQuery = query.Encode()
		childDialect = postgres.Open(parsed.String())
		createSQL, dropSQL = "CREATE SCHEMA \""+name+"\"", "DROP SCHEMA \""+name+"\" CASCADE"
	}
	root, err := gorm.Open(rootDialect, config)
	require.NoError(t, err)
	sqlRoot, err := root.DB()
	require.NoError(t, err)
	t.Cleanup(func() { sqlRoot.Close() })
	require.NoError(t, root.Exec(createSQL).Error)
	t.Cleanup(func() { require.NoError(t, root.Exec(dropSQL).Error) })
	db, err := gorm.Open(childDialect, config)
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	t.Cleanup(func() { sqlDB.Close() })
	require.NoError(t, db.AutoMigrate(model.AllModels()...))
	return db
}

func TestHistoryDatabaseEngines(t *testing.T) {
	for _, test := range []struct{ driver, env string }{
		{"mysql", "KFP_HISTORY_MYSQL_TEST_DSN"},
		{"postgres", "KFP_HISTORY_POSTGRES_TEST_DSN"},
	} {
		t.Run(test.driver, func(t *testing.T) {
			dsn := os.Getenv(test.env)
			if dsn == "" {
				t.Skip("set " + test.env + " to run database integration tests")
			}
			source, dest := externalDatabase(t, test.driver, dsn), externalDatabase(t, test.driver, dsn)
			fixture(t, source)
			create(t, dest, &model.Experiment{UUID: "destination", Name: "Default", Namespace: "team"})
			create(t, dest, &model.Run{UUID: "native", Namespace: "team", ExperimentId: "destination"})
			bundle := exportFixture(t, source)
			opts := ImportOptions{NamePrefix: "retired-", DryRun: true}
			_, err := Import(context.Background(), dest, bundle, opts)
			require.NoError(t, err)
			require.Equal(t, int64(1), count(t, dest, &model.Run{}))
			opts.DryRun = false
			result, err := Import(context.Background(), dest, bundle, opts)
			require.NoError(t, err)
			require.Equal(t, 1, result.Imported)
			result, err = Import(context.Background(), dest, bundle, opts)
			require.NoError(t, err)
			require.Equal(t, 1, result.Skipped)
			require.Equal(t, int64(2), count(t, dest, &model.Run{}))
			var links []model.ArtifactTask
			require.NoError(t, dest.Find(&links).Error)
			require.Len(t, links, 1)
			require.Zero(t, links[0].Iteration)
			var artifacts []model.Artifact
			require.NoError(t, readRows(dest, &artifacts))
			require.NoError(t, insertHistoryArtifact(dest, &artifacts[0], bundle.Source))
			require.ErrorContains(t, insertHistoryArtifact(dest, &artifacts[0], "unrelated-source"), "conflicts")
			raw, err := json.Marshal(artifacts[0].Metadata)
			require.NoError(t, err)
			require.Contains(t, string(raw), "9007199254740993")
			bundle.Entries[0].Run.DisplayName = "conflicting revision"
			_, err = Import(context.Background(), dest, bundle, opts)
			require.ErrorContains(t, err, "conflicts")
			require.Equal(t, int64(2), count(t, dest, &model.Run{}))
		})
	}
}
