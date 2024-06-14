// Copyright 2018-2023 The Kubeflow Authors
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

package clientmanager

import (
	"database/sql"
	"fmt"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"os"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	"github.com/kubeflow/pipelines/backend/src/apiserver/archive"
	"github.com/kubeflow/pipelines/backend/src/apiserver/auth"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/minio/minio-go/v6"
)

const (
	minioServiceHost   = "MINIO_SERVICE_SERVICE_HOST"
	minioServicePort   = "MINIO_SERVICE_SERVICE_PORT"
	minioServiceRegion = "MINIO_SERVICE_REGION"
	minioServiceSecure = "MINIO_SERVICE_SECURE"
	pipelineBucketName = "MINIO_PIPELINE_BUCKET_NAME"
	pipelinePath       = "MINIO_PIPELINE_PATH"

	mysqlServiceHost       = "DBConfig.MySQLConfig.Host"
	mysqlServicePort       = "DBConfig.MySQLConfig.Port"
	mysqlUser              = "DBConfig.MySQLConfig.User"
	mysqlPassword          = "DBConfig.MySQLConfig.Password"
	mysqlDBName            = "DBConfig.MySQLConfig.DBName"
	mysqlGroupConcatMaxLen = "DBConfig.MySQLConfig.GroupConcatMaxLen"
	mysqlExtraParams       = "DBConfig.MySQLConfig.ExtraParams"

	postgresHost     = "DBConfig.PostgreSQLConfig.Host"
	postgresPort     = "DBConfig.PostgreSQLConfig.Port"
	postgresUser     = "DBConfig.PostgreSQLConfig.User"
	postgresPassword = "DBConfig.PostgreSQLConfig.Password"
	postgresDBName   = "DBConfig.PostgreSQLConfig.DBName"

	archiveLogFileName   = "ARCHIVE_CONFIG_LOG_FILE_NAME"
	archiveLogPathPrefix = "ARCHIVE_CONFIG_LOG_PATH_PREFIX"
	dbConMaxLifeTime     = "DBConfig.ConMaxLifeTime"

	VisualizationServiceHost = "ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_HOST"
	VisualizationServicePort = "ML_PIPELINE_VISUALIZATIONSERVER_SERVICE_PORT"

	initConnectionTimeout = "InitConnectionTimeout"

	clientQPS   = "ClientQPS"
	clientBurst = "ClientBurst"
)

// Container for all service clients.
type ClientManager struct {
	db                        *storage.DB
	experimentStore           storage.ExperimentStoreInterface
	pipelineStore             storage.PipelineStoreInterface
	jobStore                  storage.JobStoreInterface
	runStore                  storage.RunStoreInterface
	taskStore                 storage.TaskStoreInterface
	resourceReferenceStore    storage.ResourceReferenceStoreInterface
	dBStatusStore             storage.DBStatusStoreInterface
	defaultExperimentStore    storage.DefaultExperimentStoreInterface
	objectStore               storage.ObjectStoreInterface
	execClient                util.ExecutionClient
	swfClient                 client.SwfClientInterface
	k8sCoreClient             client.KubernetesCoreInterface
	subjectAccessReviewClient client.SubjectAccessReviewInterface
	tokenReviewClient         client.TokenReviewInterface
	metadataClient            metadata.ClientInterface
	logArchive                archive.LogArchiveInterface
	time                      util.TimeInterface
	uuid                      util.UUIDGeneratorInterface
	authenticators            []auth.Authenticator
}

func (c *ClientManager) TaskStore() storage.TaskStoreInterface {
	return c.taskStore
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

func (c *ClientManager) ExecClient() util.ExecutionClient {
	return c.execClient
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

func (c *ClientManager) TokenReviewClient() client.TokenReviewInterface {
	return c.tokenReviewClient
}

func (c *ClientManager) MetadataClient() metadata.ClientInterface {
	return c.metadataClient
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

func (c *ClientManager) Authenticators() []auth.Authenticator {
	return c.authenticators
}

func (c *ClientManager) init() {
	glog.Info("Initializing client manager")
	glog.Info("Initializing DB client...")
	db := InitDBClient(common.GetDurationConfig(initConnectionTimeout))
	db.SetConnMaxLifetime(common.GetDurationConfig(dbConMaxLifeTime))
	glog.Info("DB client initialized successfully")
	// time
	c.time = util.NewRealTime()

	// UUID generator
	c.uuid = util.NewUUIDGenerator()

	c.db = db
	c.experimentStore = storage.NewExperimentStore(db, c.time, c.uuid)
	c.pipelineStore = storage.NewPipelineStore(db, c.time, c.uuid)
	c.jobStore = storage.NewJobStore(db, c.time)
	c.taskStore = storage.NewTaskStore(db, c.time, c.uuid)
	c.resourceReferenceStore = storage.NewResourceReferenceStore(db)
	c.dBStatusStore = storage.NewDBStatusStore(db)
	c.defaultExperimentStore = storage.NewDefaultExperimentStore(db)
	glog.Info("Initializing Minio client...")
	c.objectStore = initMinioClient(common.GetDurationConfig(initConnectionTimeout))
	glog.Info("Minio client initialized successfully")
	// Use default value of client QPS (5) & burst (10) defined in
	// k8s.io/client-go/rest/config.go#RESTClientFor
	clientParams := util.ClientParameters{
		QPS:   common.GetFloat64ConfigWithDefault(clientQPS, 5),
		Burst: common.GetIntConfigWithDefault(clientBurst, 10),
	}

	c.execClient = util.NewExecutionClientOrFatal(util.ArgoWorkflow, common.GetDurationConfig(initConnectionTimeout), clientParams)

	c.swfClient = client.NewScheduledWorkflowClientOrFatal(common.GetDurationConfig(initConnectionTimeout), clientParams)

	c.k8sCoreClient = client.CreateKubernetesCoreOrFatal(common.GetDurationConfig(initConnectionTimeout), clientParams)

	newClient, err := metadata.NewClient(common.GetMetadataGrpcServiceServiceHost(), common.GetMetadataGrpcServiceServicePort())
	if err != nil {
		glog.Fatalf("Failed to create metadata client. Error: %v", err)
	}
	c.metadataClient = newClient

	runStore := storage.NewRunStore(db, c.time)
	c.runStore = runStore

	// Log archive
	c.logArchive = initLogArchive()

	if common.IsMultiUserMode() {
		c.subjectAccessReviewClient = client.CreateSubjectAccessReviewClientOrFatal(common.GetDurationConfig(initConnectionTimeout), clientParams)
		c.tokenReviewClient = client.CreateTokenReviewClientOrFatal(common.GetDurationConfig(initConnectionTimeout), clientParams)
		c.authenticators = auth.GetAuthenticators(c.tokenReviewClient)
	}
	glog.Infof("Client manager initialized successfully")
}

func (c *ClientManager) Close() {
	c.db.Close()
}

func InitDBClient(initConnectionTimeout time.Duration) *storage.DB {
	// Allowed driverName values:
	// 1) To use MySQL, use `mysql`
	// 2) To use PostgreSQL, use `pgx`
	driverName := common.GetStringConfig("DBDriverName")
	arg := initDBDriver(driverName, initConnectionTimeout)

	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	db, err := gorm.Open(driverName, arg)
	util.TerminateIfError(err)

	// If pipeline_versions table is introduced into DB for the first time,
	// it needs initialization or data backfill.
	var tableNames []string
	initializePipelineVersions := true
	db.Raw(`show tables`).Pluck("Tables_in_mlpipeline", &tableNames)
	for _, tableName := range tableNames {
		if tableName == "pipeline_versions" {
			initializePipelineVersions = false
			break
		}
	}

	// Create table
	response := db.AutoMigrate(
		&model.DBStatus{},
		&model.DefaultExperiment{},
		&model.Experiment{},
		&model.Pipeline{},
		&model.PipelineVersion{},
		&model.Job{},
		&model.Run{},
		&model.RunMetric{},
		&model.Task{},
		&model.ResourceReference{},
	)

	if ignoreAlreadyExistError(driverName, response.Error) != nil {
		glog.Fatalf("Failed to initialize the databases. Error: %s", response.Error)
	}

	var textFormat string
	switch driverName {
	case "mysql":
		textFormat = client.MYSQL_TEXT_FORMAT
	case "pgx":
		textFormat = client.PGX_TEXT_FORMAT
	default:
		glog.Fatalf("Unsupported database driver %s, please use `mysql` for MySQL, or `pgx` for PostgreSQL.", driverName)
	}

	response = db.Model(&model.Experiment{}).RemoveIndex("Name")
	if response.Error != nil {
		glog.Fatalf("Failed to drop unique key on experiment name. Error: %s", response.Error)
	}

	response = db.Model(&model.Pipeline{}).RemoveIndex("Name")
	if response.Error != nil {
		glog.Fatalf("Failed to drop unique key on pipeline name. Error: %s", response.Error)
	}

	response = db.Model(&model.ResourceReference{}).ModifyColumn("Payload", textFormat)
	if response.Error != nil {
		glog.Fatalf("Failed to update the resource reference payload type. Error: %s", response.Error)
	}

	response = db.Model(&model.Run{}).AddIndex("experimentuuid_createatinsec", "ExperimentUUID", "CreatedAtInSec")
	if ignoreAlreadyExistError(driverName, response.Error) != nil {
		glog.Fatalf("Failed to create index experimentuuid_createatinsec on run_details. Error: %s", response.Error)
	}

	response = db.Model(&model.Run{}).AddIndex("experimentuuid_conditions_finishedatinsec", "ExperimentUUID", "Conditions", "FinishedAtInSec")
	if ignoreAlreadyExistError(driverName, response.Error) != nil {
		glog.Fatalf("Failed to create index experimentuuid_conditions_finishedatinsec on run_details. Error: %s", response.Error)
	}

	response = db.Model(&model.Run{}).AddIndex("namespace_createatinsec", "Namespace", "CreatedAtInSec")
	if ignoreAlreadyExistError(driverName, response.Error) != nil {
		glog.Fatalf("Failed to create index namespace_createatinsec on run_details. Error: %s", response.Error)
	}

	response = db.Model(&model.Run{}).AddIndex("namespace_conditions_finishedatinsec", "Namespace", "Conditions", "FinishedAtInSec")
	if ignoreAlreadyExistError(driverName, response.Error) != nil {
		glog.Fatalf("Failed to create index namespace_conditions_finishedatinsec on run_details. Error: %s", response.Error)
	}

	response = db.Model(&model.Pipeline{}).AddUniqueIndex("name_namespace_index", "Name", "Namespace")
	if ignoreAlreadyExistError(driverName, response.Error) != nil {
		glog.Fatalf("Failed to create index name_namespace_index on run_details. Error: %s", response.Error)
	}

	switch driverName {
	case "pgx":
		response = db.Model(&model.RunMetric{}).
			AddForeignKey("\"RunUUID\"", "run_details(\"UUID\")", "CASCADE" /* onDelete */, "CASCADE" /* onUpdate */)
		if ignoreAlreadyExistError(driverName, response.Error) != nil {
			glog.Fatalf("Failed to create a foreign key for RunUUID in run_metrics table. Error: %s", response.Error)
		}
		response = db.Model(&model.PipelineVersion{}).
			AddForeignKey("\"PipelineId\"", "pipelines(\"UUID\")", "CASCADE" /* onDelete */, "CASCADE" /* onUpdate */)
		if ignoreAlreadyExistError(driverName, response.Error) != nil {
			glog.Fatalf("Failed to create a foreign key for PipelineId in pipeline_versions table. Error: %s", response.Error)
		}
		response = db.Model(&model.Task{}).
			AddForeignKey("\"RunUUID\"", "run_details(\"UUID\")", "CASCADE" /* onDelete */, "CASCADE" /* onUpdate */)
		if ignoreAlreadyExistError(driverName, response.Error) != nil {
			glog.Fatalf("Failed to create a foreign key for RunUUID in task table. Error: %s", response.Error)
		}
	case "mysql":
		response = db.Model(&model.RunMetric{}).
			AddForeignKey("RunUUID", "run_details(UUID)", "CASCADE" /* onDelete */, "CASCADE" /* onUpdate */)
		if ignoreAlreadyExistError(driverName, response.Error) != nil {
			glog.Fatalf("Failed to create a foreign key for RunUUID in run_metrics table. Error: %s", response.Error)
		}
		response = db.Model(&model.PipelineVersion{}).
			AddForeignKey("PipelineId", "pipelines(UUID)", "CASCADE" /* onDelete */, "CASCADE" /* onUpdate */)
		if ignoreAlreadyExistError(driverName, response.Error) != nil {
			glog.Fatalf("Failed to create a foreign key for PipelineId in pipeline_versions table. Error: %s", response.Error)
		}
		response = db.Model(&model.Task{}).
			AddForeignKey("RunUUID", "run_details(UUID)", "CASCADE" /* onDelete */, "CASCADE" /* onUpdate */)
		if ignoreAlreadyExistError(driverName, response.Error) != nil {
			glog.Fatalf("Failed to create a foreign key for RunUUID in task table. Error: %s", response.Error)
		}
	default:
		glog.Fatalf("Driver %v is not supported, use \"mysql\" for MySQL, or \"pgx\" for PostgreSQL", driverName)
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

	response = db.Model(&model.Pipeline{}).ModifyColumn("Description", textFormat)
	if response.Error != nil {
		glog.Fatalf("Failed to update pipeline description type. Error: %s", response.Error)
	}

	// Because PostgreSQL was supported later, there's no need to delete the relic index
	if driverName == "mysql" {
		// If the old unique index idx_pipeline_version_uuid_name on pipeline_versions exists, remove it.
		rows, err := db.Raw(`show index from pipeline_versions where Key_name='idx_pipeline_version_uuid_name'`).Rows()
		if err != nil {
			glog.Fatalf("Failed to query pipeline_version table's indices. Error: %s", err)
		}
		if err := rows.Err(); err != nil {
			glog.Fatalf("Failed to query pipeline_version table's indices. Error: %s", err)
		}
		if rows.Next() {
			db.Exec(`drop index idx_pipeline_version_uuid_name on pipeline_versions`)
		}
		defer rows.Close()
	}

	return storage.NewDB(db.DB(), storage.NewMySQLDialect())
}

// Initializes Database driver. Use `driverName` to indicate which type of DB to use:
// 1) "mysql" for MySQL
// 2) "pgx" for PostgreSQL
func initDBDriver(driverName string, initConnectionTimeout time.Duration) string {
	var sqlConfig, dbName string
	var mysqlConfig *mysql.Config
	switch driverName {
	case "mysql":
		mysqlConfig = client.CreateMySQLConfig(
			common.GetStringConfigWithDefault(mysqlUser, "root"),
			common.GetStringConfigWithDefault(mysqlPassword, ""),
			common.GetStringConfigWithDefault(mysqlServiceHost, "mysql"),
			common.GetStringConfigWithDefault(mysqlServicePort, "3306"),
			"",
			common.GetStringConfigWithDefault(mysqlGroupConcatMaxLen, "1024"),
			common.GetMapConfig(mysqlExtraParams),
		)
		sqlConfig = mysqlConfig.FormatDSN()
		dbName = common.GetStringConfig(mysqlDBName)
	case "pgx":
		sqlConfig = client.CreatePostgreSQLConfig(
			common.GetStringConfigWithDefault(postgresUser, "user"),
			common.GetStringConfigWithDefault(postgresPassword, "password"),
			common.GetStringConfigWithDefault(postgresHost, "postgresql"),
			"postgres",
			uint16(common.GetIntConfigWithDefault(postgresPort, 5432)),
		)
		dbName = common.GetStringConfig(postgresDBName)
	default:
		glog.Fatalf("Driver %v is not supported, use \"mysql\" for MySQL, or \"pgx\" for PostgreSQL", driverName)
	}

	var db *sql.DB
	var err error
	operation := func() error {
		db, err = sql.Open(driverName, sqlConfig)
		if err != nil {
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.RetryNotify(operation, b, func(e error, duration time.Duration) {
		glog.Errorf("%v", e)
	})

	defer db.Close()
	util.TerminateIfError(err)

	// Create database if not exist
	operation = func() error {
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if ignoreAlreadyExistError(driverName, err) != nil {
			return err
		}
		return nil
	}
	b = backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	util.TerminateIfError(err)

	switch driverName {
	case "mysql":
		mysqlConfig.DBName = dbName
		// When updating, return rows matched instead of rows affected. This counts rows that are being
		// set as the same values as before. If updating using a primary key and rows matched is 0, then
		// it means this row is not found.
		// Config reference: https://github.com/go-sql-driver/mysql#clientfoundrows
		mysqlConfig.ClientFoundRows = true
		sqlConfig = mysqlConfig.FormatDSN()
	case "pgx":
		// Note: postgreSQL does not have the option `ClientFoundRows`
		// Config reference: https://www.postgresql.org/docs/current/libpq-connect.html
		sqlConfig = client.CreatePostgreSQLConfig(
			common.GetStringConfigWithDefault(postgresUser, "root"),
			common.GetStringConfigWithDefault(postgresPassword, ""),
			common.GetStringConfigWithDefault(postgresHost, "postgresql"),
			dbName,
			uint16(common.GetIntConfigWithDefault(postgresPort, 5432)),
		)
	default:
		glog.Fatalf("Driver %v is not supported, use \"mysql\" for MySQL, or \"pgx\" for PostgreSQL", driverName)
	}
	return sqlConfig
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

// NewClientManager creates and Init a new instance of ClientManager.
func NewClientManager() ClientManager {
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
	// resources using the same ID is OK.
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

func backfillExperimentIDToRunTable(db *gorm.DB) error {
	// check if there is any row in the run table has experiment ID being empty
	rows, err := db.CommonDB().Query("SELECT \"ExperimentUUID\" FROM run_details WHERE \"ExperimentUUID\" = '' LIMIT 1")
	if err != nil {
		return err
	}
	if err := rows.Err(); err != nil {
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

// Returns the same error, if it's not "already exists" related.
// Otherwise, return nil.
func ignoreAlreadyExistError(driverName string, err error) error {
	if driverName == "pgx" && err != nil && strings.Contains(err.Error(), client.PGX_EXIST_ERROR) {
		return nil
	}
	if driverName == "mysql" && err != nil && strings.Contains(err.Error(), client.MYSQL_EXIST_ERROR) {
		return nil
	}
	return err
}
