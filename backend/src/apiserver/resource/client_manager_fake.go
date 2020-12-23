// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resource

import (
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/archive"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const (
	DefaultFakeUUID = "123e4567-e89b-12d3-a456-426655440000"
	FakeUUIDOne     = "123e4567-e89b-12d3-a456-426655440001"
	NonDefaultFakeUUID     =   "123e4567-e89b-12d3-a456-426655441000"
)

type FakeClientManager struct {
	db                            *storage.DB
	experimentStore               storage.ExperimentStoreInterface
	pipelineStore                 storage.PipelineStoreInterface
	jobStore                      storage.JobStoreInterface
	runStore                      storage.RunStoreInterface
	resourceReferenceStore        storage.ResourceReferenceStoreInterface
	dBStatusStore                 storage.DBStatusStoreInterface
	defaultExperimentStore        storage.DefaultExperimentStoreInterface
	objectStore                   storage.ObjectStoreInterface
	ArgoClientFake                *client.FakeArgoClient
	swfClientFake                 *client.FakeSwfClient
	k8sCoreClientFake             *client.FakeKuberneteCoreClient
	SubjectAccessReviewClientFake client.SubjectAccessReviewInterface
	logArchive                    archive.LogArchiveInterface
	time                          util.TimeInterface
	uuid                          util.UUIDGeneratorInterface
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

	// TODO(neuromage): Pass in metadata.Store instance for tests as well.
	return &FakeClientManager{
		db:                            db,
		experimentStore:               storage.NewExperimentStore(db, time, uuid),
		pipelineStore:                 storage.NewPipelineStore(db, time, uuid),
		jobStore:                      storage.NewJobStore(db, time),
		runStore:                      storage.NewRunStore(db, time),
		ArgoClientFake:                client.NewFakeArgoClient(),
		resourceReferenceStore:        storage.NewResourceReferenceStore(db),
		dBStatusStore:                 storage.NewDBStatusStore(db),
		defaultExperimentStore:        storage.NewDefaultExperimentStore(db),
		objectStore:                   storage.NewFakeObjectStore(),
		swfClientFake:                 client.NewFakeSwfClient(),
		k8sCoreClientFake:             client.NewFakeKuberneteCoresClient(),
		SubjectAccessReviewClientFake: client.NewFakeSubjectAccessReviewClient(),
		logArchive:                    archive.NewLogArchive("/logs", "main.log"),
		time:                          time,
		uuid:                          uuid,
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

func (f *FakeClientManager) ExperimentStore() storage.ExperimentStoreInterface {
	return f.experimentStore
}

func (f *FakeClientManager) PipelineStore() storage.PipelineStoreInterface {
	return f.pipelineStore
}

func (f *FakeClientManager) ObjectStore() storage.ObjectStoreInterface {
	return f.objectStore
}

func (f *FakeClientManager) LogArchive() archive.LogArchiveInterface {
	return f.logArchive
}

func (f *FakeClientManager) Time() util.TimeInterface {
	return f.time
}

func (f *FakeClientManager) UUID() util.UUIDGeneratorInterface {
	return f.uuid
}

func (f *FakeClientManager) DB() *storage.DB {
	return f.db
}

func (f *FakeClientManager) ArgoClient() client.ArgoClientInterface {
	return f.ArgoClientFake
}

func (f *FakeClientManager) JobStore() storage.JobStoreInterface {
	return f.jobStore
}

func (f *FakeClientManager) RunStore() storage.RunStoreInterface {
	return f.runStore
}

func (f *FakeClientManager) ResourceReferenceStore() storage.ResourceReferenceStoreInterface {
	return f.resourceReferenceStore
}

func (f *FakeClientManager) DBStatusStore() storage.DBStatusStoreInterface {
	return f.dBStatusStore
}

func (f *FakeClientManager) DefaultExperimentStore() storage.DefaultExperimentStoreInterface {
	return f.defaultExperimentStore
}

func (f *FakeClientManager) SwfClient() client.SwfClientInterface {
	return f.swfClientFake
}

func (f *FakeClientManager) KubernetesCoreClient() client.KubernetesCoreInterface {
	return f.k8sCoreClientFake
}

func (f *FakeClientManager) SubjectAccessReviewClient() client.SubjectAccessReviewInterface {
	return f.SubjectAccessReviewClientFake
}

func (f *FakeClientManager) Close() error {
	return f.db.Close()
}

// Update the uuid used in this fake client manager
func (f *FakeClientManager) UpdateUUID(uuid util.UUIDGeneratorInterface) {
	f.uuid = uuid
	f.experimentStore = storage.NewExperimentStore(f.db, f.time, uuid)
	f.pipelineStore = storage.NewPipelineStore(f.db, f.time, uuid)
}
