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
	"database/sql"
	"fmt"

	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	_ "github.com/mattn/go-sqlite3"
)

func NewInMemoryTestDB() (*DB, error) {
	// Initialize GORM
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, util.Wrap(err, "Could not create the GORM database")
	}
	// Create tables
	dbWithMigration, err := migrate(db)
	if err != nil {
		return nil, util.Wrap(err, "Could not migrate the database")
	}
	return NewDB(dbWithMigration, NewSQLiteDialect()), nil
}

func migrate(db *gorm.DB) (*sql.DB, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	// Create tables
	db.AutoMigrate(
		&model.Experiment{},
		&model.Job{},
		&model.Pipeline{},
		&model.PipelineVersion{},
		&model.ResourceReference{},
		&model.Run{},
		&model.RunMetric{},
		&model.Task{},
		&model.DBStatus{},
		&model.DefaultExperiment{},
	)
	return db.DB(), nil
}

func NewFakeDBOrFatal() *DB {
	db, err := NewInMemoryTestDB()
	if err != nil {
		glog.Fatalf("The fake DB doesn't create successfully. Fail fast: %v", err)
	}
	return db
}
