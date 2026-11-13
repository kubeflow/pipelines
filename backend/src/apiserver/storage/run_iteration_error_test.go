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
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type runIterationErrorConnector struct{ err error }

func (c *runIterationErrorConnector) Connect(context.Context) (driver.Conn, error) {
	return &runIterationErrorConnection{err: c.err}, nil
}
func (c *runIterationErrorConnector) Driver() driver.Driver { return nil }

type runIterationErrorConnection struct{ err error }

func (c *runIterationErrorConnection) Prepare(string) (driver.Stmt, error) {
	return nil, errors.New("unused")
}
func (c *runIterationErrorConnection) Close() error              { return nil }
func (c *runIterationErrorConnection) Begin() (driver.Tx, error) { return nil, errors.New("unused") }
func (c *runIterationErrorConnection) QueryContext(context.Context, string, []driver.NamedValue) (driver.Rows, error) {
	return &runIterationErrorRows{err: c.err}, nil
}

type runIterationErrorRows struct{ err error }

func (r *runIterationErrorRows) Columns() []string         { return []string{"UUID"} }
func (r *runIterationErrorRows) Close() error              { return nil }
func (r *runIterationErrorRows) Next([]driver.Value) error { return r.err }

func queryRunIterationErrorRows(t *testing.T, streamErr error) *sql.Rows {
	t.Helper()
	db := sql.OpenDB(&runIterationErrorConnector{err: streamErr})
	t.Cleanup(func() { _ = db.Close() })
	rows, err := db.Query("SELECT 1")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rows.Close() })
	return rows
}
