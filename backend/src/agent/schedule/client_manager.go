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

package main

import (
	"github.com/googleprivate/ml/backend/src/apiserver/client"
	"github.com/googleprivate/ml/backend/src/apiserver/storage"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/googleprivate/ml/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1alpha1"
	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

type ClientManager struct {
	db                 *gorm.DB
	packageStore       storage.PackageStoreInterface
	pipelineStore      storage.PipelineStoreInterface
	jobStore           storage.JobStoreInterface
	objectStore        storage.ObjectStoreInterface
	workflowClientFake *storage.FakeWorkflowClient
	time               util.TimeInterface
	uuid               util.UUIDGeneratorInterface
}

type ClientManagerParams struct {
	DBDriverName         string
	SqliteDatasourceName string
	User                 string
	MysqlServiceHost     string
	MysqlServicePort     string
	MysqlDBName          string
	MinioServiceHost     string
	MinioServicePort     string
	MinioAccessKey       string
	MinioSecretKey       string
	MinioBucketName      string
	Namespace            string
}

func NewClientManager(params *ClientManagerParams) (*ClientManager, error) {

	time := util.NewRealTime()
	db, err := client.CreateGormClient(
		params.DBDriverName,
		params.SqliteDatasourceName,
		params.User,
		params.MysqlServiceHost,
		params.MysqlServicePort,
		params.MysqlDBName)

	if err != nil {
		return nil, errors.Wrapf(err, "Error while creating GORM client: %+v: %+v", params, err)
	}

	workflowClient, err := client.CreateWorkflowClient(params.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while creating workflow client: %+v", err)
	}

	minioClient, err := client.CreateMinioClient(
		params.MinioServiceHost,
		params.MinioServicePort,
		params.MinioAccessKey,
		params.MinioSecretKey)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while create Minio client: %+v", err)
	}

	return &ClientManager{
		db:            db,
		packageStore:  storage.NewPackageStore(db, time),
		pipelineStore: storage.NewPipelineStore(db, time),
		jobStore:      storage.NewJobStore(db, workflowClient, time),
		objectStore: storage.NewMinioObjectStore(
			&storage.MinioClient{Client: minioClient},
			params.MinioBucketName),
		time: time,
		uuid: util.NewUUIDGenerator(),
	}, nil
}

func (c *ClientManager) PackageStore() storage.PackageStoreInterface {
	return c.packageStore
}

func (c *ClientManager) PipelineStore() storage.PipelineStoreInterface {
	return c.pipelineStore
}

func (c *ClientManager) JobStore() storage.JobStoreInterface {
	return c.jobStore
}

func (c *ClientManager) ObjectStore() storage.ObjectStoreInterface {
	return c.objectStore
}

func (c *ClientManager) Time() util.TimeInterface {
	return c.time
}

func (c *ClientManager) UUID() util.UUIDGeneratorInterface {
	return c.uuid
}

func (s *ClientManager) Close() error {
	return s.db.Close()
}

func (c *ClientManager) ScheduledWorkflow() v1alpha1.ScheduledWorkflowInterface {
	// It's intended to not implement the method, since these code will soon be deprecated.
	return nil
}

func (c *ClientManager) PipelineStoreV2() storage.PipelineStoreV2Interface {
	// It's intended to not implement the method, since these code will soon be deprecated.
	return nil
}

func (c *ClientManager) JobStoreV2() storage.JobStoreV2Interface {
	// It's intended to not implement the method, since these code will soon be deprecated.
	return nil
}
