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
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type SystemInfoStoreInterface interface {
	IsSampleLoaded() (bool, error)
	MarkSampleLoaded() error
}

type SystemInfoStore struct {
	db *DB
}

func (s *SystemInfoStore) InitializeSystemInfoTable() error {
	getSystemInfoSql, getSystemInfoArgs, err := sq.Select("*").From("system_infos").ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Error creating query to get system info.")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return util.NewInternalServerError(err, "Failed to create a new transaction to initialize system info.")
	}

	rows, err := tx.Query(getSystemInfoSql, getSystemInfoArgs...)
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to get load sample status")
	}

	// The table is not initialized
	if !rows.Next() {
		sql, args, err := sq.
			Insert("system_infos").
			SetMap(sq.Eq{"IsSampleLoaded": false}).
			ToSql()

		if err != nil {
			tx.Rollback()
			return util.NewInternalServerError(err, "Error creating query to initialize system info table.")
		}
		_, err = tx.Exec(sql, args...)
		if err != nil {
			tx.Rollback()
			return util.NewInternalServerError(err, "Error initializing the system info table.")
		}
	}
	err = tx.Commit()
	if err != nil {
		tx.Rollback()
		return util.NewInternalServerError(err, "Failed to initializing the system info table.")
	}
	return nil
}

func (s *SystemInfoStore) IsSampleLoaded() (bool, error) {
	var isSampleLoaded bool
	sql, args, err := sq.Select("*").From("system_infos").ToSql()
	if err != nil {
		return false, util.NewInternalServerError(err, "Error creating query to get load sample status.")
	}
	rows, err := s.db.Query(sql, args...)
	if err != nil {
		return false, util.NewInternalServerError(err, "Error when getting load sample status")
	}
	if rows.Next() {
		err = rows.Scan(&isSampleLoaded)
		if err != nil {
			return false, util.NewInternalServerError(err, "Error when scanning row to load sample status")
		}
		return isSampleLoaded, nil
	}
	return false, nil
}

func (s *SystemInfoStore) MarkSampleLoaded() error {
	sql, args, err := sq.
		Update("system_infos").
		SetMap(sq.Eq{"IsSampleLoaded": true}).
		ToSql()
	if err != nil {
		return util.NewInternalServerError(err, "Error creating query to mark samples as loaded.")
	}
	_, err = s.db.Exec(sql, args...)
	if err != nil {
		return util.NewInternalServerError(err, "Error mark samples as loaded.")
	}
	return nil
}

// factory function for system info store
func NewSystemInfoStore(db *DB) *SystemInfoStore {
	s := &SystemInfoStore{db: db}
	// Initialize system information table
	s.InitializeSystemInfoTable()
	return s
}
