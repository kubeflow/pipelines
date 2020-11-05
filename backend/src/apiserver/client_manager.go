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
	"os"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/kubeflow/pipelines/backend/src/apiserver/archive"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/minio/minio-go"
)

const (
	minioServiceHost       = "MINIO_SERVICE_SERVICE_HOST"
	minioServicePort       = "MINIO_SERVICE_SERVICE_PORT"
	minioServiceRegion     = "MINIO_SERVICE_REGION"
	minioServiceSecure     = "MINIO_SERVICE_SECURE"
	pipelineBucketName     = "MINIO_PIPELINE_BUCKET_NAME"
	pipelinePath           = "MINIO_PIPELINE_PATH"
	mysqlServiceHost       = "DBConfig.Host"
	mysqlServicePort       = "DBConfig.Port"
	mysqlUser              = "DBConfig.User"
	mysqlPassword          = "DBConfig.Password"
	mysqlDBName            = "DBConfig.DBName"
	mysqlGroupConcatMaxLen = "DBConfig.GroupConcatMaxLen"
	mysqlExtraParams       = "DBConfig.ExtraParams"
	archiveLogFileName     = "ARCHIVE_CONFIG_LOG_FILE_NAME"
	archiveLogPathPrefix   = "ARCHIVE_CONFIG_LOG_PATH_PREFIX"

	visualizationServiceHost = "ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_HOST"
	visualizationServicePort = "ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_PORT"

	initConnectionTimeout = "InitConnectionTimeout"
)

// Container for all service clients
type ClientManager struct {
	db                        *storage.DB
	experimentStore           storage.ExperimentStoreInterface
	pipelineStore             storage.PipelineStoreInterface
	jobStore                  storage.JobStoreInterface
	runStore                  storage.RunStoreInterface
	resourceReferenceStore    storage.ResourceReferenceStoreInterface
	dBStatusStore             storage.DBStatusStoreInterface
	defaultExperimentStore    storage.DefaultExperimentStoreInterface
	objectStore               storage.ObjectStoreInterface
	argoClient                client.ArgoClientInterface
	swfClient                 client.SwfClientInterface
	k8sCoreClient             client.KubernetesCoreInterface
	subjectAccessReviewClient client.SubjectAccessReviewInterface
	logArchive                archive.LogArchiveInterface
	time                      util.TimeInterface
	uuid                      util.UUIDGeneratorInterface
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

func (c *ClientManager) ArgoClient() client.ArgoClientInterface {
	return c.argoClient
}

func (c *ClientManager) SwfClient() client.SwfClientInterface {
	return c.swfClient
}

func (c *ClientManager) KubernetesCoreClient() client.KubernetesCoreInterface {
	return c.k8sCoreClient
}

func (c *ClientManager) SubjectAccessReviewClient() client.SubjectAccessReviewInterface {
	return c.subjectAccessReviewClient
}

func (c *ClientManager) LogArchive() archive.LogArchiveInterface {
	return c.logArchive
}

func (c *ClientManager) Time() util.TimeInterface {
	return c.time
}

func (c *ClientManager) UUID() util.UUIDGeneratorInterface {
	return c.uuid
}

func (c *ClientManager) init() {
	glog.Info("Initializing client manager")
	db := initDBClient(common.GetDurationConfig(initConnectionTimeout))

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
	c.objectStore = initMinioClient(common.GetDurationConfig(initConnectionTimeout))

	c.argoClient = client.NewArgoClientOrFatal(common.GetDurationConfig(initConnectionTimeout))

	c.swfClient = client.NewScheduledWorkflowClientOrFatal(common.GetDurationConfig(initConnectionTimeout))

	c.k8sCoreClient = client.CreateKubernetesCoreOrFatal(common.GetDurationConfig(initConnectionTimeout))

	runStore := storage.NewRunStore(db, c.time)
	c.runStore = runStore

	// Log archive
	c.logArchive = initLogArchive()

	if common.IsMultiUserMode() {
		c.subjectAccessReviewClient = client.CreateSubjectAccessReviewClientOrFatal(common.GetDurationConfig(initConnectionTimeout))
	}
	glog.Infof("Client manager initialized successfully")
}

func (c *ClientManager) Close() {
	c.db.Close()
}

func initDBClient(initConnectionTimeout time.Duration) *storage.DB {
	driverName := common.GetStringConfig("DBConfig.DriverName")
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

	// If pipeline_versions table is introduced into DB for the first time,
	// it needs initialization or data backfill.
	var tableNames []string
	var initializePipelineVersions = true
	db.Raw(`show tables`).Pluck("Tables_in_mlpipeline", &tableNames)
	for _, tableName := range tableNames {
		if tableName == "pipeline_versions" {
			initializePipelineVersions = false
			break
		}
	}

	// Create table
	response := db.AutoMigrate(
		&model.Experiment{},
		&model.Job{},
		&model.Pipeline{},
		&model.PipelineVersion{},
		&model.ResourceReference{},
		&model.RunDetail{},
		&model.RunMetric{},
		&model.DBStatus{},
		&model.DefaultExperiment{})

	if response.Error != nil {
		glog.Fatalf("Failed to initialize the databases.")
	}

	response = db.Model(&model.Experiment{}).RemoveIndex("Name")
	if response.Error != nil {
		glog.Fatalf("Failed to drop unique key on experiment name. Error: %s", response.Error)
	}

	response = db.Model(&model.ResourceReference{}).ModifyColumn("Payload", "longtext not null")
	if response.Error != nil {
		glog.Fatalf("Failed to update the resource reference payload type. Error: %s", response.Error)
	}

	response = db.Model(&model.RunDetail{}).AddIndex("experimentuuid_createatinsec", "ExperimentUUID", "CreatedAtInSec")
	if response.Error != nil {
		glog.Fatalf("Failed to create index experimentuuid_createatinsec on run_details. Error: %s", response.Error)
	}

	response = db.Model(&model.RunDetail{}).AddIndex("experimentuuid_conditions_finishedatinsec", "ExperimentUUID", "Conditions", "FinishedAtInSec")
	if response.Error != nil {
		glog.Fatalf("Failed to create index experimentuuid_conditions_finishedatinsec on run_details. Error: %s", response.Error)
	}

	response = db.Model(&model.RunMetric{}).
		AddForeignKey("RunUUID", "run_details(UUID)", "CASCADE" /* onDelete */, "CASCADE" /* update */)
	if response.Error != nil {
		glog.Fatalf("Failed to create a foreign key for RunID in run_metrics table. Error: %s", response.Error)
	}
	response = db.Model(&model.PipelineVersion{}).
		AddForeignKey("PipelineId", "pipelines(UUID)", "CASCADE" /* onDelete */, "CASCADE" /* update */)
	if response.Error != nil {
		glog.Fatalf("Failed to create a foreign key for PipelineId in pipeline_versions table. Error: %s", response.Error)
	}

	// Data backfill for pipeline_versions if this is the first time for
	// pipeline_versions to enter mlpipeline DB.
	if initializePipelineVersions {
		initPipelineVersionsFromPipelines(db)
	}
	err = backfillExperimentIDToRunTable(db)
	if err != nil {
		glog.Fatalf("Failed to backfill experiment UUID in run_details table: %s", err)
	}

	response = db.Model(&model.Pipeline{}).ModifyColumn("Description", "longtext not null")
	if response.Error != nil {
		glog.Fatalf("Failed to update pipeline description type. Error: %s", response.Error)
	}

	// If the old unique index idx_pipeline_version_uuid_name on pipeline_versions exists, remove it.
	rows, err := db.Raw(`show index from pipeline_versions where Key_name="idx_pipeline_version_uuid_name"`).Rows()
	if err != nil {
		glog.Fatalf("Failed to query pipeline_version table's indices. Error: %s", err)
	}
	if rows.Next() {
		db.Exec(`drop index idx_pipeline_version_uuid_name on pipeline_versions`)
	}
	rows.Close()

	return storage.NewDB(db.DB(), storage.NewMySQLDialect())
}

// Initialize the connection string for connecting to Mysql database
// Format would be something like root@tcp(ip:port)/dbname?charset=utf8&loc=Local&parseTime=True
func initMysql(driverName string, initConnectionTimeout time.Duration) string {
	mysqlConfig := client.CreateMySQLConfig(
		common.GetStringConfigWithDefault(mysqlUser, "root"),
		common.GetStringConfigWithDefault(mysqlPassword, ""),
		common.GetStringConfigWithDefault(mysqlServiceHost, "mysql"),
		common.GetStringConfigWithDefault(mysqlServicePort, "3306"),
		"",
		common.GetStringConfigWithDefault(mysqlGroupConcatMaxLen, "1024"),
		common.GetMapConfig(mysqlExtraParams),
	)

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
	//err = backoff.Retry(operation, b)
	backoff.RetryNotify(operation, b, func(e error, duration time.Duration) {
		glog.Errorf("%v", e)
	})

	defer db.Close()
	util.TerminateIfError(err)

	// Create database if not exist
	dbName := common.GetStringConfig(mysqlDBName)
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
	// When updating, return rows matched instead of rows affected. This counts rows that are being
	// set as the same values as before. If updating using a primary key and rows matched is 0, then
	// it means this row is not found.
	// Config reference: https://github.com/go-sql-driver/mysql#clientfoundrows
	mysqlConfig.ClientFoundRows = true
	return mysqlConfig.FormatDSN()
}

func initMinioClient(initConnectionTimeout time.Duration) storage.ObjectStoreInterface {
	// Create minio client.
	minioServiceHost := common.GetStringConfigWithDefault(
		"ObjectStoreConfig.Host", os.Getenv(minioServiceHost))
	minioServicePort := common.GetStringConfigWithDefault(
		"ObjectStoreConfig.Port", os.Getenv(minioServicePort))
	minioServiceRegion := common.GetStringConfigWithDefault(
		"ObjectStoreConfig.Region", os.Getenv(minioServiceRegion))
	minioServiceSecure := common.GetBoolConfigWithDefault(
		"ObjectStoreConfig.Secure", common.GetBoolFromStringWithDefault(os.Getenv(minioServiceSecure), false))
	accessKey := common.GetStringConfigWithDefault("ObjectStoreConfig.AccessKey", "")
	secretKey := common.GetStringConfigWithDefault("ObjectStoreConfig.SecretAccessKey", "")
	bucketName := common.GetStringConfigWithDefault("ObjectStoreConfig.BucketName", os.Getenv(pipelineBucketName))
	pipelinePath := common.GetStringConfigWithDefault("ObjectStoreConfig.PipelinePath", os.Getenv(pipelinePath))
	disableMultipart := common.GetBoolConfigWithDefault("ObjectStoreConfig.Multipart.Disable", true)

	minioClient := client.CreateMinioClientOrFatal(minioServiceHost, minioServicePort, accessKey,
		secretKey, minioServiceSecure, minioServiceRegion, initConnectionTimeout)
	createMinioBucket(minioClient, bucketName, minioServiceRegion)

	return storage.NewMinioObjectStore(&storage.MinioClient{Client: minioClient}, bucketName, pipelinePath, disableMultipart)
}

func createMinioBucket(minioClient *minio.Client, bucketName, region string) {
	// Check to see if we already own this bucket.
	exists, err := minioClient.BucketExists(bucketName)
	if err != nil {
		glog.Fatalf("Failed to check if Minio bucket exists. Error: %v", err)
	}
	if exists {
		glog.Infof("We already own %s\n", bucketName)
		return
	}
	// Create bucket if it does not exist
	err = minioClient.MakeBucket(bucketName, region)
	if err != nil {
		glog.Fatalf("Failed to create Minio bucket. Error: %v", err)
	}
	glog.Infof("Successfully created bucket %s\n", bucketName)
}

func initLogArchive() (logArchive archive.LogArchiveInterface) {
	logFileName := common.GetStringConfigWithDefault(archiveLogFileName, "")
	logPathPrefix := common.GetStringConfigWithDefault(archiveLogPathPrefix, "")

	if logFileName != "" && logPathPrefix != "" {
		logArchive = archive.NewLogArchive(logPathPrefix, logFileName)
	}

	return
}

// newClientManager creates and Init a new instance of ClientManager
func newClientManager() ClientManager {
	clientManager := ClientManager{}
	clientManager.init()

	return clientManager
}

// Data migration in 2 steps to introduce pipeline_versions table. This
// migration shall be called only once when pipeline_versions table is created
// for the first time in DB.
func initPipelineVersionsFromPipelines(db *gorm.DB) {
	tx := db.Begin()

	// Step 1: duplicate pipelines to pipeline versions.
	// The pipeline versions created here are not through KFP pipeine version
	// API, and are only for the legacy pipelines that are created
	// before pipeline version API is introduced.
	// For those legacy pipelines, who don't have versions before, we create one
	// implicit version for each of them. Given a legacy pipeline, the implicit
	// version created here is assigned an ID the same as the pipeline ID. This
	// way we don't need to move the minio file of pipeline package around,
	// since the minio file's path is based on the pipeline ID (and now on the
	// implicit version ID too). Meanwhile, IDs are required to be unique inside
	// the same resource type, so pipeline and pipeline version as two different
	// resources useing the same ID is OK.
	// On the other hand, pipeline and its pipeline versions created after
	// pipeline version API is introduced will have different Ids; and the minio
	// file will be put directly into the directories for pipeline versions.
	tx.Exec(`INSERT INTO
	pipeline_versions (UUID, Name, CreatedAtInSec, Parameters, Status, PipelineId)
	SELECT UUID, Name, CreatedAtInSec, Parameters, Status, UUID FROM pipelines;`)

	// Step 2: modifiy pipelines table after pipeline_versions are populated.
	tx.Exec("update pipelines set DefaultVersionId=UUID;")

	tx.Commit()
}

func backfillExperimentIDToRunTable(db *gorm.DB) (retError error) {
	// check if there is any row in the run table has experiment ID being empty
	rows, err := db.CommonDB().Query(`SELECT ExperimentUUID FROM run_details WHERE ExperimentUUID = '' LIMIT 1`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// no row in run_details table has empty ExperimentUUID
	if !rows.Next() {
		return nil
	}

	_, err = db.CommonDB().Exec(`
		UPDATE
			run_details, resource_references
		SET
			run_details.ExperimentUUID = resource_references.ReferenceUUID
		WHERE
			run_details.UUID = resource_references.ResourceUUID
			AND resource_references.ResourceType = 'Run'
			AND resource_references.ReferenceType = 'Experiment'
			AND run_details.ExperimentUUID = ''
	`)
	return err
}
