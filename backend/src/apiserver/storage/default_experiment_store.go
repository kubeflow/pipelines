// Copyright 2019 Google LLC
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
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

var (
	defaultExperimentDBValue = sq.Eq{"DefaultExperimentId": ""}
)

type DefaultExperimentStoreInterface interface {
	GetDefaultExperimentId() (string, error)
	SetDefaultExperimentId(id string) error
}

// Implementation of a DefaultExperimentStoreInterface. This stores the default experiment's ID,
// which is created the first time the API server is initialized.
type DefaultExperimentStore struct {
	db *DB
}

func (s *DefaultExperimentStore) initializeDefaultExperimentTable() error {
	// First check that the table is in fact empty
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create a new transaction to initialize default experiment table.")
	}
	rows, err := tx.Query("SELECT * FROM default_experiments")
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to get default experiment.")
	}
	next := rows.Next()
	rows.Close()

	// If the table is not initialized, then set the default value.
	if !next {
		sql, args, queryErr := sq.
			Insert("default_experiments").
			SetMap(defaultExperimentDBValue).
			ToSql()

		if queryErr != nil {
			tx.Rollback()
			return util.NewInternalServerError(queryErr, "Error creating query to initialize default experiment table.")
		}

		_, err = tx.Exec(sql, args...)
		if err != nil {
			tx.Rollback()
			return util.NewInternalServerError(err, "Error initializing the default experiment table.")
		}
	}
	err = tx.Commit()
	if err != nil {
		glog.Error("Failed to commit transaction to initialize default experiment table")
		return util.NewInternalServerError(err, "Failed to initializing the default experiment table.")
	}
	return nil
}

func (s *DefaultExperimentStore) SetDefaultExperimentId(id string) error {
	sql, args, err := sq.
		Update("default_experiments").
		SetMap(sq.Eq{"DefaultExperimentId": id}).
		Where(sq.Eq{"DefaultExperimentId": ""}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Error creating query to set default experiment ID.")
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err, "Error setting default experiment ID.")
	}
	return nil
}

func (s *DefaultExperimentStore) GetDefaultExperimentId() (string, error) {
	var defaultExperimentId string
	sql, args, err := sq.Select("DefaultExperimentId").From("default_experiments").ToSql()
	if err != nil {
		return "", util.NewInternalServerError(err, "Error creating query to get default experiment ID.")
	}

	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return "", util.NewInternalServerError(err, "Error when getting default experiment ID")
	}
	defer rows.Close()

	if rows.Next() {
		err = rows.Scan(&defaultExperimentId)
		if err != nil {
			return "", util.NewInternalServerError(err, "Error when scanning row to find default experiment ID")
		}
		return defaultExperimentId, nil
	}
	return "", nil
}

// Sets the default experiment ID stored in the DB to the empty string. This needs to happen if the
// experiment is deleted via the normal delete experiment API so that the server knows to create a
// new default.
// This is always done alongside the deletion of the actual experiment itself, so a transaction is
// needed as input.
// Update is used instead of delete so that we don't need to first check that the experiment ID is
// there.
func (s *DefaultExperimentStore) UnsetDefaultExperimentIdIfIdMatches(tx *sql.Tx, id string) error {
	sql, args, err := sq.
		Update("default_experiments").
		SetMap(sq.Eq{"DefaultExperimentId": ""}).
		Where(sq.Eq{"DefaultExperimentId": id}).
		ToSql()
	_, err = tx.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err, "Failed to clear default experiment with ID: %s", id)
	}
	return nil
}

// factory function for creating default experiment store
func NewDefaultExperimentStore(db *DB) *DefaultExperimentStore {
	s := &DefaultExperimentStore{db: db}
	// Initialize default experiment table
	s.initializeDefaultExperimentTable()
	return s
}
