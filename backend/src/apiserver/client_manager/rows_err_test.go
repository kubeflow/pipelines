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

package clientmanager

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type cmMockRowsSpec struct {
	cols      []string
	rows      [][]driver.Value
	streamErr error
}

type cmMockConnector struct {
	queryHandler func(query string) cmMockRowsSpec
	execQueries  *[]string
}

func (c *cmMockConnector) Connect(context.Context) (driver.Conn, error) {
	return &cmMockConn{queryHandler: c.queryHandler, execQueries: c.execQueries}, nil
}

func (c *cmMockConnector) Driver() driver.Driver { return nil }

type cmMockConn struct {
	queryHandler func(query string) cmMockRowsSpec
	execQueries  *[]string
}

func (c *cmMockConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("unused") }
func (c *cmMockConn) Close() error                        { return nil }
func (c *cmMockConn) Begin() (driver.Tx, error)           { return cmMockTx{}, nil }

func (c *cmMockConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	if c.execQueries != nil {
		*c.execQueries = append(*c.execQueries, query)
	}
	return driver.RowsAffected(0), nil
}

func (c *cmMockConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	spec := c.queryHandler(query)
	return &cmMockRows{spec: spec}, nil
}

type cmMockTx struct{}

func (cmMockTx) Commit() error   { return nil }
func (cmMockTx) Rollback() error { return nil }

type cmMockRows struct {
	spec cmMockRowsSpec
	idx  int
}

func (r *cmMockRows) Columns() []string { return r.spec.cols }
func (r *cmMockRows) Close() error      { return nil }
func (r *cmMockRows) Next(dest []driver.Value) error {
	if r.idx < len(r.spec.rows) {
		copy(dest, r.spec.rows[r.idx])
		r.idx++
		return nil
	}
	if r.spec.streamErr != nil {
		return r.spec.streamErr
	}
	return io.EOF
}

func openMockMySQLGormDB(t *testing.T, queryHandler func(query string) cmMockRowsSpec) (*gorm.DB, *[]string) {
	t.Helper()
	var execQueries []string
	sqlDB := sql.OpenDB(&cmMockConnector{
		queryHandler: queryHandler,
		execQueries:  &execQueries,
	})
	t.Cleanup(func() { _ = sqlDB.Close() })

	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	require.NoError(t, err)
	return db, &execQueries
}

func TestBackfillExperimentIDToRunTable_RowsErr(t *testing.T) {
	streamErr := errors.New("simulated backfill rows failure")

	t.Run("immediate rows error stops before UPDATE", func(t *testing.T) {
		db, execQueries := openMockMySQLGormDB(t, func(string) cmMockRowsSpec {
			return cmMockRowsSpec{
				cols:      []string{"ExperimentUUID"},
				streamErr: streamErr,
			}
		})
		err := backfillExperimentIDToRunTable(db, dialect.NewDBDialect("mysql"))
		require.ErrorIs(t, err, streamErr)
		require.Empty(t, *execQueries, "backfill must not UPDATE when row check fails")
	})

	t.Run("clean empty result skips UPDATE", func(t *testing.T) {
		db, execQueries := openMockMySQLGormDB(t, func(string) cmMockRowsSpec {
			return cmMockRowsSpec{
				cols: []string{"ExperimentUUID"},
			}
		})
		err := backfillExperimentIDToRunTable(db, dialect.NewDBDialect("mysql"))
		require.NoError(t, err)
		require.Empty(t, *execQueries)
	})

	t.Run("non-empty result runs UPDATE", func(t *testing.T) {
		db, execQueries := openMockMySQLGormDB(t, func(string) cmMockRowsSpec {
			return cmMockRowsSpec{
				cols: []string{"ExperimentUUID"},
				rows: [][]driver.Value{{""}},
			}
		})
		err := backfillExperimentIDToRunTable(db, dialect.NewDBDialect("mysql"))
		require.NoError(t, err)
		require.Len(t, *execQueries, 1)
		require.Contains(t, (*execQueries)[0], "UPDATE")
	})
}

func TestDropLegacyIndexesMySQL_RowsErr(t *testing.T) {
	streamErr := errors.New("simulated SHOW INDEX failure")

	baseHandler := func(showIdxSpec cmMockRowsSpec) func(string) cmMockRowsSpec {
		return func(query string) cmMockRowsSpec {
			switch {
			case strings.Contains(query, "information_schema.STATISTICS"):
				return cmMockRowsSpec{cols: []string{"INDEX_NAME"}}
			case strings.Contains(query, "information_schema.tables") || strings.Contains(query, "SELECT count(*)"):
				return cmMockRowsSpec{cols: []string{"count(*)"}, rows: [][]driver.Value{{int64(1)}}}
			case strings.Contains(query, "SHOW INDEX FROM pipeline_versions"):
				return showIdxSpec
			default:
				return cmMockRowsSpec{cols: []string{"1"}}
			}
		}
	}

	t.Run("SHOW INDEX rows error stops before DROP INDEX", func(t *testing.T) {
		db, execQueries := openMockMySQLGormDB(t, baseHandler(cmMockRowsSpec{
			cols:      []string{"Key_name"},
			streamErr: streamErr,
		}))
		err := dropLegacyIndexesMySQL(db)
		require.ErrorIs(t, err, streamErr)
		require.ErrorContains(t, err, "iterate pipeline_versions indices failed")
		require.Empty(t, *execQueries, "cleanup must not DROP INDEX when SHOW INDEX iteration fails")
	})

	t.Run("clean empty SHOW INDEX skips DROP INDEX", func(t *testing.T) {
		db, execQueries := openMockMySQLGormDB(t, baseHandler(cmMockRowsSpec{
			cols: []string{"Key_name"},
		}))
		err := dropLegacyIndexesMySQL(db)
		require.NoError(t, err)
		require.Empty(t, *execQueries)
	})

	t.Run("legacy index present runs DROP INDEX", func(t *testing.T) {
		db, execQueries := openMockMySQLGormDB(t, baseHandler(cmMockRowsSpec{
			cols: []string{"Key_name"},
			rows: [][]driver.Value{{"idx_pipeline_version_uuid_name"}},
		}))
		err := dropLegacyIndexesMySQL(db)
		require.NoError(t, err)
		require.Len(t, *execQueries, 1)
		require.Equal(t, "DROP INDEX `idx_pipeline_version_uuid_name` ON `pipeline_versions`", (*execQueries)[0])
	})
}
