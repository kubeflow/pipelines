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
	"fmt"
	"ml/apiserver/src/message/pipelinemanager"
	"ml/apiserver/src/storage"
	"ml/apiserver/src/util"

	argoclient "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/minio/minio-go"
	k8sclient "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

const (
	minioServiceHost = "MINIO_SERVICE_SERVICE_HOST"
	minioServicePort = "MINIO_SERVICE_SERVICE_PORT"
)

// Container for all service clients
type ClientManager struct {
	db             *gorm.DB
	packageStore   storage.PackageStoreInterface
	pipelineStore  storage.PipelineStoreInterface
	jobStore       storage.JobStoreInterface
	packageManager storage.PackageManagerInterface
}

func (clientManager *ClientManager) Init() {
	glog.Infof("Initializing client manager")

	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	db, err := gorm.Open(getConfig("DBConfig.DriverName"), getConfig("DBConfig.DataSourceName"))
	util.TerminateIfError(err)

	// Create table
	db.AutoMigrate(&pipelinemanager.Package{}, &pipelinemanager.Pipeline{}, &pipelinemanager.Parameter{})

	// Initialize package store
	clientManager.db = db
	clientManager.packageStore = storage.NewPackageStore(db)

	// Initialize pipeline store
	clientManager.db = db
	clientManager.pipelineStore = storage.NewPipelineStore(db)

	// Initialize job store
	wfClient := initWorkflowClient()
	clientManager.jobStore = storage.NewJobStore(wfClient)

	// Initialize package manager.
	clientManager.packageManager = initMinioClient()

	glog.Infof("Client manager initialized successfully")
}

func (clientManager *ClientManager) End() {
	clientManager.db.Close()
}

func initMinioClient() storage.PackageManagerInterface {
	minioServiceHost := getConfig(minioServiceHost)
	minioServicePort := getConfig(minioServicePort)
	minioClient, err := minio.New(
		fmt.Sprintf("%s:%s", minioServiceHost, minioServicePort),
		getConfig("PackageManagerConfig.AccessKey"),
		getConfig("PackageManagerConfig.SecretAccessKey"),
		false /* Secure connection */)
	if err != nil {
		glog.Fatalf("Failed to create Minio client. Error: %v", err)
	}

	bucketName := getConfig("PackageManagerConfig.BucketName")
	err = minioClient.MakeBucket(bucketName, "")
	if err != nil {
		// Check to see if we already own this bucket.
		exists, err := minioClient.BucketExists(bucketName)
		if err == nil && exists {
			glog.Infof("We already own %s\n", bucketName)
		} else {
			glog.Fatalf("Failed to create Minio bucket. Error: %v", err)
		}
	}
	glog.Infof("Successfully created %s\n", bucketName)
	return storage.NewMinioPackageManager(&storage.MinioClient{Client: minioClient}, bucketName)
}

// creates a new client for the Kubernetes Workflow CRD.
func initWorkflowClient() v1alpha1.WorkflowInterface {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		glog.Fatalf("Failed to initialize workflow client. Error: %v", err)
	}
	wfClientSet := argoclient.NewForConfigOrDie(restConfig)
	wfClient := wfClientSet.ArgoprojV1alpha1().Workflows(k8sclient.NamespaceDefault)
	return wfClient
}

// newClientManager creates and Init a new instance of ClientManager
func newClientManager() ClientManager {
	clientManager := ClientManager{}
	clientManager.Init()

	return clientManager
}
