// Copyright 2018 The Kubeflow Authors
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
	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type DBStatusStoreInterface interface {
	HaveSamplesLoaded() (bool, error)
	MarkSampleLoaded() error
}

var dbStatusStoreColumns = []string{
	"HaveSamplesLoaded",
}

// Implementation of a DBStatusStoreInterface. This store read/write state of the database.
// For now we store status like whether sample is loaded.
type DBStatusStore struct {
	db      *DB
	dialect dialect.DBDialect
}

func (s *DBStatusStore) InitializeDBStatusTable() error {
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create a new transaction to initialize database status")
	}

	sqlSelect, argsSelect, err := qb.Select("*").From(q("db_statuses")).ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create query to check db_statuses table")
	}
	rows, err := tx.Query(sqlSelect, argsSelect...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to load database status")
	}
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to load database status")
	}
	next := rows.Next()
	defer rows.Close() // "rows" shouldn't be used after this point.

	// The table is not initialized
	if !next {
		sqlInsert, argsInsert, queryErr := qb.
			Insert(q("db_statuses")).
			SetMap(sq.Eq{q("HaveSamplesLoaded"): false}).
			ToSql()

		if queryErr != nil {
			tx.Rollback()
			return util.NewInternalServerError(queryErr, "Error creating query to initialize database status table")
		}

		_, err = tx.Exec(sqlInsert, argsInsert...)
		if err != nil {
			tx.Rollback()
			return util.NewInternalServerError(err, "Error initializing the database status table")
		}
	}
	err = tx.Commit()
	if err != nil {
		glog.Error("Failed to commit transaction to initialize database status table")
		return util.NewInternalServerError(err, "Failed to initializing the database status table")
	}
	return nil
}

func (s *DBStatusStore) HaveSamplesLoaded() (bool, error) {
	var haveSamplesLoaded bool
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	quotedCols := make([]string, len(dbStatusStoreColumns))
	for i, c := range dbStatusStoreColumns {
		quotedCols[i] = q(c)
	}
	sql, args, err := qb.Select(quotedCols...).From(q("db_statuses")).ToSql()
	if err != nil {
		return false, util.NewInternalServerError(err, "Error creating query to get load sample status")
	}
	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return false, util.NewInternalServerError(err, "Error when getting load sample status")
	}
	if err := rows.Err(); err != nil {
		return false, util.NewInternalServerError(err, "Error when getting load sample status")
	}
	defer rows.Close()
	if rows.Next() {
		err = rows.Scan(&haveSamplesLoaded)
		if err != nil {
			return false, util.NewInternalServerError(err, "Error when scanning row to load sample status")
		}
		return haveSamplesLoaded, nil
	}
	return false, nil
}

func (s *DBStatusStore) MarkSampleLoaded() error {
	q := s.dialect.QuoteIdentifier
	qb := s.dialect.QueryBuilder()
	sql, args, err := qb.
		Update(q("db_statuses")).
		SetMap(sq.Eq{q("HaveSamplesLoaded"): true}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Error creating query to mark samples as loaded")
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err, "Error mark samples as loaded")
	}
	return nil
}

// factory function for database status store.
func NewDBStatusStore(db *DB, d dialect.DBDialect) *DBStatusStore {
	s := &DBStatusStore{db: db, dialect: d}
	// Initialize database status table
	s.InitializeDBStatusTable()
	return s
}
