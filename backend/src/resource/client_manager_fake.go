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

package resource

import (
	"github.com/golang/glog"
	"github.com/googleprivate/ml/backend/src/storage"
	"github.com/googleprivate/ml/backend/src/util"
	"github.com/jinzhu/gorm"
	"github.com/kubeflow/pipelines/pkg/client/clientset/versioned/typed/scheduledworkflow/v1alpha1"
)

const (
	DefaultFakeUUID = "123e4567-e89b-12d3-a456-426655440000"
)

type FakeClientManager struct {
	db                          *gorm.DB
	packageStore                storage.PackageStoreInterface
	pipelineStore               storage.PipelineStoreInterface
	jobStore                    storage.JobStoreInterface
	pipelineStoreV2             storage.PipelineStoreV2Interface
	jobStoreV2                  storage.JobStoreV2Interface
	objectStore                 storage.ObjectStoreInterface
	workflowClientFake          *storage.FakeWorkflowClient
	scheduledWorkflowClientFake *FakeScheduledWorkflowClient
	time                        util.TimeInterface
	uuid                        util.UUIDGeneratorInterface
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
	db, err := storage.NewFakeDb()
	if err != nil {
		return nil, err
	}
	workflowClient := storage.NewWorkflowClientFake()

	return &FakeClientManager{
		db:                          db,
		packageStore:                storage.NewPackageStore(db, time),
		pipelineStore:               storage.NewPipelineStore(db, time),
		jobStore:                    storage.NewJobStore(db, workflowClient, time),
		pipelineStoreV2:             storage.NewPipelineStoreV2(db, time),
		jobStoreV2:                  storage.NewJobStoreV2(db, time),
		workflowClientFake:          workflowClient,
		objectStore:                 storage.NewFakeObjectStore(),
		scheduledWorkflowClientFake: NewScheduledWorkflowClientFake(),
		time: time,
		uuid: uuid,
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

func (f *FakeClientManager) PackageStore() storage.PackageStoreInterface {
	return f.packageStore
}

func (f *FakeClientManager) PipelineStore() storage.PipelineStoreInterface {
	return f.pipelineStore
}

func (f *FakeClientManager) JobStore() storage.JobStoreInterface {
	return f.jobStore
}

func (f *FakeClientManager) ObjectStore() storage.ObjectStoreInterface {
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

func (f *FakeClientManager) WorkflowClientFake() *storage.FakeWorkflowClient {
	return f.workflowClientFake
}

func (f *FakeClientManager) PipelineStoreV2() storage.PipelineStoreV2Interface {
	return f.pipelineStoreV2
}

func (f *FakeClientManager) JobStoreV2() storage.JobStoreV2Interface {
	return f.jobStoreV2
}

func (f *FakeClientManager) ScheduledWorkflow() v1alpha1.ScheduledWorkflowInterface {
	return f.scheduledWorkflowClientFake
}

func (f *FakeClientManager) Close() error {
	return f.db.Close()
}
