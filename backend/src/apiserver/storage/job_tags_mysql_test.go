// Copyright 2026 The Kubeflow Authors
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
	"database/sql"
	"fmt"
	"os"
	"testing"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// TestJobTagsMySQL creates and removes only its own uniquely named database.
// The optional DSN must point to a disposable server with CREATE DATABASE permission.
func TestJobTagsMySQL(t *testing.T) {
	dsn := os.Getenv("KFP_JOB_TAGS_MYSQL_TEST_DSN")
	if dsn == "" {
		t.Skip("set KFP_JOB_TAGS_MYSQL_TEST_DSN to run MySQL tag coverage")
	}
	config, err := mysqldriver.ParseDSN(dsn)
	require.NoError(t, err)
	config.DBName = ""
	admin, err := sql.Open("mysql", config.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, admin.Close()) })
	d := dialect.NewDBDialect("mysql")
	name := fmt.Sprintf("kfp_job_tags_%d", time.Now().UnixNano())
	_, err = admin.Exec("CREATE DATABASE " + d.QuoteIdentifier(name))
	require.NoError(t, err)
	t.Cleanup(func() { _, err := admin.Exec("DROP DATABASE " + d.QuoteIdentifier(name)); require.NoError(t, err) })
	config.DBName = name
	db, err := sql.Open("mysql", config.FormatDSN())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	orm, err := gorm.Open(mysql.New(mysql.Config{Conn: db}), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, orm.AutoMigrate(model.AllModels()...))
	t.Run("lifecycle", func(t *testing.T) { testJobTagsLifecycle(t, db, d) })
	t.Run("filters", func(t *testing.T) { testListJobTagsPaginationAndIsolation(t, db, d) })
}
