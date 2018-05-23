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
	"ml/backend/src/model"
	"ml/backend/src/util"

	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

const (
	DefaultFakeUUID = "123e4567-e89b-12d3-a456-426655440000"
)

type FakeClientManager struct {
	db                 *gorm.DB
	packageStore       PackageStoreInterface
	pipelineStore      PipelineStoreInterface
	jobStore           JobStoreInterface
	objectStore        ObjectStoreInterface
	workflowClientFake *FakeWorkflowClient
	time               util.TimeInterface
	uuid               util.UUIDGeneratorInterface
}

func NewFakeClientManager(time util.TimeInterface, uuid util.UUIDGeneratorInterface) (
	*FakeClientManager, error) {

	if time == nil {
		glog.Fatalf("The time parameter must not be null.") // Must never happen
	}

	if uuid == nil {
		glog.Fatalf("The UUID generator must not be null.") // Must never happen
	}

	// Initialize GORM
	db, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("Could not create the GORM database: %v", err)
	}

	// Create tables
	db.AutoMigrate(&model.Package{}, &model.Pipeline{}, &model.Job{})
	workflowClient := NewWorkflowClientFake()

	return &FakeClientManager{
		db:                 db,
		packageStore:       NewPackageStore(db, time),
		pipelineStore:      NewPipelineStore(db, time),
		jobStore:           NewJobStore(db, workflowClient, time),
		workflowClientFake: workflowClient,
		objectStore:        NewFakeObjectStore(),
		time:               time,
		uuid:               uuid,
	}, nil
}

func NewFakeClientManagerOrFatal(time util.TimeInterface) *FakeClientManager {
	uuid := util.NewFakeUUIDGeneratorOrFatal(DefaultFakeUUID, nil)
	fakeStore, err := NewFakeClientManager(time, uuid)
	if err != nil {
		glog.Fatalf("The fake store doesn't create successfully. Fail fast.")
	}
	return fakeStore
}

func (f *FakeClientManager) PackageStore() PackageStoreInterface {
	return f.packageStore
}

func (f *FakeClientManager) PipelineStore() PipelineStoreInterface {
	return f.pipelineStore
}

func (f *FakeClientManager) JobStore() JobStoreInterface {
	return f.jobStore
}

func (f *FakeClientManager) ObjectStore() ObjectStoreInterface {
	return f.objectStore
}

func (f *FakeClientManager) Time() util.TimeInterface {
	return f.time
}

func (c *FakeClientManager) UUID() util.UUIDGeneratorInterface {
	return c.uuid
}

func (f *FakeClientManager) DB() *gorm.DB {
	return f.db
}

func (f *FakeClientManager) WorkflowClientFake() *FakeWorkflowClient {
	return f.workflowClientFake
}

func (f *FakeClientManager) Close() error {
	return f.db.Close()
}
