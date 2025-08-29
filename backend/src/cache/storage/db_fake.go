// Copyright 2020 The Kubeflow Authors
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
	"fmt"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func NewFakeDB() (*DB, dialect.DBDialect, error) {
	// Initialize GORM
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		return nil, dialect.DBDialect{}, fmt.Errorf("could not create the GORM database: %v", err)
	}
	// Create tables
	err = db.AutoMigrate(&model.ExecutionCache{})
	if err != nil {
		return nil, dialect.DBDialect{}, fmt.Errorf("AutoMigrate failed: %v", err)
	}

	return NewDB(db), dialect.NewDBDialect("sqlite"), nil
}

func NewFakeDBOrFatal() (*DB, dialect.DBDialect) {
	db, dialect, err := NewFakeDB()
	if err != nil {
		glog.Fatalf("The fake DB doesn't create successfully. Fail fast.")
	}
	return db, dialect
}
