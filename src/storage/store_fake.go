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
	"fmt"
	"ml/src/message"
	"ml/src/util"

	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

type FakeStore struct {
	DB                 *gorm.DB
	PackageStore       PackageStoreInterface
	PipelineStore      PipelineStoreInterface
	JobStore           JobStoreInterface
	PackageManager     PackageManagerInterface
	WorkflowClientFake *FakeWorkflowClient
	Time               util.TimeInterface
}

func NewFakeStore(time util.TimeInterface) (*FakeStore, error) {

	// Initialize GORM
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("Could not create the GORM database: %v", err)
	}

	// Skip auto populating the CreatedAt/UpdatedAt/DeletedAt field to avoid unpredictable value.
	db.Callback().Create().Remove("gorm:update_time_stamp")
	db.Callback().Update().Remove("gorm:update_time_stamp")

	// Create tables
	db.AutoMigrate(&message.Package{}, &message.Pipeline{},
		&message.Parameter{}, &message.Job{})

	workflowClient := NewWorkflowClientFake()

	return &FakeStore{
		DB:                 db,
		PackageStore:       NewPackageStore(db),
		PipelineStore:      NewPipelineStore(db, time),
		JobStore:           NewJobStore(db, workflowClient, time),
		WorkflowClientFake: workflowClient,
		PackageManager:     NewFakePackageManager(),
		Time:               time,
	}, nil
}

func NewFakeStoreOrFatal(time util.TimeInterface) *FakeStore {
	fakeStore, err := NewFakeStore(time)
	if err != nil {
		glog.Exitf("The fake store doesn't create successfully. Fail fast.")
	}
	return fakeStore
}

func (s *FakeStore) Close() error {
	return s.DB.Close()
}
