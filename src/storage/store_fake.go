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

	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
)

type FakeStore struct {
	gormDatabase *gorm.DB
	PipelineStore PipelineStoreInterface
	JobStore JobStoreInterface
	WorkflowClientFake *FakeWorkflowClient
	Time util.TimeInterface
}

func NewFakeStore(time util.TimeInterface) (*FakeStore, error) {

	// Initialize GORM
	gormDatabase, err := gorm.Open("sqlite3", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("Could not create the GORM database: %v", err)
	}

	// Create tables
	gormDatabase.AutoMigrate(&message.Package{}, &message.Pipeline{},
		&message.Parameter{}, &message.Job{})

	workflowClient := NewWorkflowClientFake()

	return &FakeStore{
		gormDatabase: gormDatabase,
		PipelineStore: NewPipelineStore(gormDatabase, time),
		JobStore: NewJobStore(gormDatabase, workflowClient, time),
		WorkflowClientFake: workflowClient,
		Time: time,
	}, nil
}

func (s *FakeStore) Close() error {
	return s.gormDatabase.Close()
}
