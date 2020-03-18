// Copyright 2020 Google LLC
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
	"github.com/jinzhu/gorm"
	"github.com/kubeflow/pipelines/backend/src/cache/model"
	_ "github.com/mattn/go-sqlite3"
)

func NewFakeDb() (*DB, error) {
	// Initialize GORM
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("Could not create the GORM database: %v", err)
	}
	// Create tables
	db.AutoMigrate(&model.ExecutionCache{})

	return NewDB(db), nil
}

func NewFakeDbOrFatal() *DB {
	db, err := NewFakeDb()
	if err != nil {
		glog.Fatalf("The fake DB doesn't create successfully. Fail fast.")
	}
	return db
}
