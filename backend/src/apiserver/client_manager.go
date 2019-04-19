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
	"database/sql"
	"fmt"
	"strconv"
	"time"

	workflowclient "github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/golang/protobuf/proto"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/metadata"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	scheduledworkflowclient "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1beta1"
	minio "github.com/minio/minio-go"

	"ml_metadata/metadata_store/mlmetadata"
	mlpb "ml_metadata/proto/metadata_store_go_proto"
)

const (
	minioServiceHost      = "MINIO_SERVICE_SERVICE_HOST"
	minioServicePort      = "MINIO_SERVICE_SERVICE_PORT"
	mysqlServiceHost      = "MYSQL_SERVICE_HOST"
	mysqlServicePort      = "MYSQL_SERVICE_PORT"
	podNamespace          = "POD_NAMESPACE"
	dbName                = "mlpipeline"
	initConnectionTimeout = "InitConnectionTimeout"
)

// Container for all service clients
type ClientManager struct {
	db                     *storage.DB
	experimentStore        storage.ExperimentStoreInterface
	pipelineStore          storage.PipelineStoreInterface
	jobStore               storage.JobStoreInterface
	runStore               storage.RunStoreInterface
	resourceReferenceStore storage.ResourceReferenceStoreInterface
	dBStatusStore          storage.DBStatusStoreInterface
	defaultExperimentStore storage.DefaultExperimentStoreInterface
	objectStore            storage.ObjectStoreInterface
	wfClient               workflowclient.WorkflowInterface
	swfClient              scheduledworkflowclient.ScheduledWorkflowInterface
	time                   util.TimeInterface
	uuid                   util.UUIDGeneratorInterface

	MetadataStore *mlmetadata.Store
}

func (c *ClientManager) ExperimentStore() storage.ExperimentStoreInterface {
	return c.experimentStore
}

func (c *ClientManager) PipelineStore() storage.PipelineStoreInterface {
	return c.pipelineStore
}

func (c *ClientManager) JobStore() storage.JobStoreInterface {
	return c.jobStore
}

func (c *ClientManager) RunStore() storage.RunStoreInterface {
	return c.runStore
}

func (c *ClientManager) ResourceReferenceStore() storage.ResourceReferenceStoreInterface {
	return c.resourceReferenceStore
}

func (c *ClientManager) DBStatusStore() storage.DBStatusStoreInterface {
	return c.dBStatusStore
}

func (c *ClientManager) DefaultExperimentStore() storage.DefaultExperimentStoreInterface {
	return c.defaultExperimentStore
}

func (c *ClientManager) ObjectStore() storage.ObjectStoreInterface {
	return c.objectStore
}

func (c *ClientManager) Workflow() workflowclient.WorkflowInterface {
	return c.wfClient
}

func (c *ClientManager) ScheduledWorkflow() scheduledworkflowclient.ScheduledWorkflowInterface {
	return c.swfClient
}

func (c *ClientManager) Time() util.TimeInterface {
	return c.time
}

func (c *ClientManager) UUID() util.UUIDGeneratorInterface {
	return c.uuid
}

func (c *ClientManager) init() {
	glog.Infof("Initializing client manager")

	db := initDBClient(getDurationConfig(initConnectionTimeout))

	// time
	c.time = util.NewRealTime()

	// UUID generator
	c.uuid = util.NewUUIDGenerator()

	c.db = db
	c.experimentStore = storage.NewExperimentStore(db, c.time, c.uuid)
	c.pipelineStore = storage.NewPipelineStore(db, c.time, c.uuid)
	c.jobStore = storage.NewJobStore(db, c.time)
	c.resourceReferenceStore = storage.NewResourceReferenceStore(db)
	c.dBStatusStore = storage.NewDBStatusStore(db)
	c.defaultExperimentStore = storage.NewDefaultExperimentStore(db)
	c.objectStore = initMinioClient(getDurationConfig(initConnectionTimeout))

	c.wfClient = client.CreateWorkflowClientOrFatal(
		getStringConfig(podNamespace), getDurationConfig(initConnectionTimeout))

	c.swfClient = client.CreateScheduledWorkflowClientOrFatal(
		getStringConfig(podNamespace), getDurationConfig(initConnectionTimeout))

	metadataStore := initMetadataStore()
	runStore := storage.NewRunStore(db, c.time, metadataStore)
	c.runStore = runStore

	glog.Infof("Client manager initialized successfully")
}

func (c *ClientManager) Close() {
	c.db.Close()
}

func initMetadataStore() *metadata.Store {
	port, err := strconv.Atoi(getStringConfig(mysqlServicePort))
	if err != nil {
		glog.Fatalf("Failed to parse valid MySQL service port from %q: %v", getStringConfig(mysqlServicePort), err)
	}

	cfg := &mlpb.ConnectionConfig{
		Config: &mlpb.ConnectionConfig_Mysql{
			&mlpb.MySQLDatabaseConfig{
				Host:     proto.String(getStringConfig(mysqlServiceHost)),
				Port:     proto.Uint32(uint32(port)),
				Database: proto.String("mlmetadata"),
				User:     proto.String("root"),
			},
		},
	}

	mlmdStore, err := mlmetadata.NewStore(cfg)
	if err != nil {
		glog.Fatalf("Failed to create ML Metadata store: %v", err)
	}
	return metadata.NewStore(mlmdStore)
}

func initDBClient(initConnectionTimeout time.Duration) *storage.DB {
	driverName := getStringConfig("DBConfig.DriverName")
	var arg string

	switch driverName {
	case "mysql":
		arg = initMysql(driverName, initConnectionTimeout)
	default:
		glog.Fatalf("Driver %v is not supported", driverName)
	}

	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	db, err := gorm.Open(driverName, arg)
	util.TerminateIfError(err)

	// Create table
	response := db.AutoMigrate(
		&model.Experiment{},
		&model.Job{},
		&model.Pipeline{},
		&model.ResourceReference{},
		&model.RunDetail{},
		&model.RunMetric{},
		&model.DBStatus{},
		&model.DefaultExperiment{})

	if response.Error != nil {
		glog.Fatalf("Failed to initialize the databases.")
	}
	response = db.Model(&model.RunMetric{}).
		AddForeignKey("RunUUID", "run_details(UUID)", "CASCADE" /* onDelete */, "CASCADE" /* update */)
	if response.Error != nil {
		glog.Fatalf("Failed to create a foreign key for RunID in run_metrics table. Error: %s", response.Error)
	}
	return storage.NewDB(db.DB(), storage.NewMySQLDialect())
}

// Initialize the connection string for connecting to Mysql database
// Format would be something like root@tcp(ip:port)/dbname?charset=utf8&loc=Local&parseTime=True
func initMysql(driverName string, initConnectionTimeout time.Duration) string {
	mysqlConfig := client.CreateMySQLConfig(
		"root",
		getStringConfig(mysqlServiceHost),
		getStringConfig(mysqlServicePort),
		"")

	var db *sql.DB
	var err error
	var operation = func() error {
		db, err = sql.Open(driverName, mysqlConfig.FormatDSN())
		if err != nil {
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	defer db.Close()
	util.TerminateIfError(err)

	// Create database if not exist
	operation = func() error {
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))
		if err != nil {
			return err
		}
		return nil
	}
	b = backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	util.TerminateIfError(err)
	mysqlConfig.DBName = dbName
	return mysqlConfig.FormatDSN()
}

func initMinioClient(initConnectionTimeout time.Duration) storage.ObjectStoreInterface {
	// Create minio client.
	minioServiceHost := getStringConfig(minioServiceHost)
	minioServicePort := getStringConfig(minioServicePort)
	accessKey := getStringConfig("ObjectStoreConfig.AccessKey")
	secretKey := getStringConfig("ObjectStoreConfig.SecretAccessKey")
	bucketName := getStringConfig("ObjectStoreConfig.BucketName")

	minioClient := client.CreateMinioClientOrFatal(minioServiceHost, minioServicePort, accessKey,
		secretKey, initConnectionTimeout)
	createMinioBucket(minioClient, bucketName)

	return storage.NewMinioObjectStore(&storage.MinioClient{Client: minioClient}, bucketName)
}

func createMinioBucket(minioClient *minio.Client, bucketName string) {
	// Create bucket if it does not exist
	err := minioClient.MakeBucket(bucketName, "")
	if err != nil {
		// Check to see if we already own this bucket.
		exists, err := minioClient.BucketExists(bucketName)
		if err == nil && exists {
			glog.Infof("We already own %s\n", bucketName)
		} else {
			glog.Fatalf("Failed to create Minio bucket. Error: %v", err)
		}
	}
	glog.Infof("Successfully created bucket %s\n", bucketName)
}

// newClientManager creates and Init a new instance of ClientManager
func newClientManager() ClientManager {
	clientManager := ClientManager{}
	clientManager.init()

	return clientManager
}
