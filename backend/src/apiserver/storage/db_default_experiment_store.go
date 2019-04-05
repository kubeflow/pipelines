// Copyright 2018 Google LLC
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
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

var (
	defaultDBValue = sq.Eq{"DefaultExperimentId": ""}
)

type DefaultExperimentStoreInterface interface {
	GetDefaultExperimentId() (string, error)
	SetDefaultExperimentId(id string) error
}

// Implementation of a DefaultExperimentStoreInterface. This stores the default experiment's ID, if
// there is one.
type DefaultExperimentStore struct {
	db *DB
}

func (s *DefaultExperimentStore) InitializeDefaultExperimentTable() error {
	getDefaultExperimentSql, getDefaultExperimentArgs, err := sq.Select("*").From("default_experiment").ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Error creating query to get default experiment ID.")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create a new transaction to initialize default experiment table.")
	}

	rows, err := tx.Query(getDefaultExperimentSql, getDefaultExperimentArgs...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to get default experiment.")
	}

	// The table is not initialized
	if !rows.Next() {
		sql, args, queryErr := sq.
			Insert("default_experiment").
			SetMap(defaultDBValue).
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
		Update("default_experiment").
		SetMap(sq.Eq{"DefaultExperimentId": id}).
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
	sql, args, err := sq.Select("DefaultExperimentId").From("default_experiment").Where(sq.Eq{}).ToSql()
	if err != nil {
		return "", util.NewInternalServerError(err, "Error creating query to get default experiment ID.")
	}
	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return "", util.NewInternalServerError(err, "Error when getting default experiment ID")
	}
	if rows.Next() {
		err = rows.Scan(&defaultExperimentId)
		if err != nil {
			return "", util.NewInternalServerError(err, "Error when scanning row to find default experiment ID")
		}
		return defaultExperimentId, nil
	}
	return "", nil
}

// factory function for creating default experiment store
func NewDefaultExperimentStore(db *DB) *DefaultExperimentStore {
	s := &DefaultExperimentStore{db: db}
	// Initialize default experiment table
	s.InitializeDefaultExperimentTable()
	return s
}
