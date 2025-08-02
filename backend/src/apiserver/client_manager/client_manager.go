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
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/cenkalti/backoff"
	mysqlStd "github.com/go-sql-driver/mysql"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/archive"
	"github.com/kubeflow/pipelines/backend/src/apiserver/auth"
	"github.com/kubeflow/pipelines/backend/src/apiserver/client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/apiserver/validation"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	k8sapi "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	"github.com/minio/minio-go/v7"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
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

var scheme *runtime.Scheme

func init() {
	scheme = runtime.NewScheme()

	err := k8sapi.AddToScheme(scheme)
	if err != nil {
		// Panic is okay here because it means there's a code issue and so the package shouldn't initialize.
		panic(fmt.Sprintf("Failed to initialize the Kubernetes API scheme: %v", err))
	}
}

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
	logArchive                archive.LogArchiveInterface
	time                      util.TimeInterface
	uuid                      util.UUIDGeneratorInterface
	authenticators            []auth.Authenticator
	controllerClient          ctrlclient.Client
	controllerClientNoCache   ctrlclient.Client
}

// Options to pass to Client Manager initialization
type Options struct {
	UsePipelineKubernetesStorage bool
	GlobalKubernetesWebhookMode  bool
	Context                      context.Context
	WaitGroup                    *sync.WaitGroup
}

func (c *ClientManager) TaskStore() storage.TaskStoreInterface {
	return c.taskStore
}

func (c *ClientManager) ControllerClient(cacheEnabled bool) ctrlclient.Client {
	if cacheEnabled {
		return c.controllerClient
	}

	return c.controllerClientNoCache
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

func (c *ClientManager) init(options *Options) error {
	// time
	c.time = util.NewRealTime()

	// UUID generator
	c.uuid = util.NewUUIDGenerator()

	var pipelineStoreForRef storage.PipelineStoreInterface

	if options.UsePipelineKubernetesStorage || options.GlobalKubernetesWebhookMode {
		glog.Info("Initializing controller client...")
		restConfig, err := util.GetKubernetesConfig()
		if err != nil {
			return err
		}

		var cacheConfig map[string]cache.Config

		if !common.IsMultiUserMode() && common.GetPodNamespace() != "" && !options.GlobalKubernetesWebhookMode {
			cacheConfig = map[string]cache.Config{common.GetPodNamespace(): {}}
		}

		k8sAPICache, err := cache.New(restConfig,
			cache.Options{
				DefaultNamespaces: cacheConfig,
				Scheme:            scheme,
			},
		)
		if err != nil {
			return err
		}

		options.WaitGroup.Add(1)
		go func() {
			defer options.WaitGroup.Done()

			err := k8sAPICache.Start(options.Context)
			if err != nil {
				panic(fmt.Sprintf("Failed to start the cache to the cluster: %v", err))
			}
		}()

		controllerClient, err := ctrlclient.New(
			restConfig, ctrlclient.Options{Scheme: scheme, Cache: &ctrlclient.CacheOptions{Reader: k8sAPICache}},
		)
		if err != nil {
			return fmt.Errorf("failed to initialize the controller client: %w", err)
		}

		controllerClientNoCache, err := ctrlclient.New(restConfig, ctrlclient.Options{Scheme: scheme})
		if err != nil {
			return fmt.Errorf("failed to initialize the no cache controller client: %w", err)
		}

		glog.Info("Controller client initialized successfully.")

		c.controllerClient = controllerClient
		c.controllerClientNoCache = controllerClientNoCache
		if options.GlobalKubernetesWebhookMode {
			return nil
		}

		c.pipelineStore = storage.NewPipelineStoreKubernetes(controllerClient, controllerClientNoCache)
		pipelineStoreForRef = c.pipelineStore
	}

	glog.Info("Initializing client manager")
	glog.Info("Initializing DB client...")
	db := InitDBClient(common.GetDurationConfig(initConnectionTimeout))
	db.SetConnMaxLifetime(common.GetDurationConfig(dbConMaxLifeTime))
	glog.Info("DB client initialized successfully")

	c.db = db
	if !options.UsePipelineKubernetesStorage {
		c.pipelineStore = storage.NewPipelineStore(db, c.time, c.uuid)
	}
	c.experimentStore = storage.NewExperimentStore(db, c.time, c.uuid)
	c.jobStore = storage.NewJobStore(db, c.time, pipelineStoreForRef)
	c.taskStore = storage.NewTaskStore(db, c.time, c.uuid)
	c.resourceReferenceStore = storage.NewResourceReferenceStore(db, pipelineStoreForRef)
	c.dBStatusStore = storage.NewDBStatusStore(db)
	c.defaultExperimentStore = storage.NewDefaultExperimentStore(db)
	glog.Info("Initializing Object store client...")
	c.objectStore = initMinioClient(options.Context, common.GetDurationConfig(initConnectionTimeout))
	glog.Info("Object store client initialized successfully")
	// Use default value of client QPS (5) & burst (10) defined in
	// k8s.io/client-go/rest/config.go#RESTClientFor
	clientParams := util.ClientParameters{
		QPS:   common.GetFloat64ConfigWithDefault(clientQPS, 5),
		Burst: common.GetIntConfigWithDefault(clientBurst, 10),
	}

	c.execClient = util.NewExecutionClientOrFatal(util.CurrentExecutionType(), common.GetDurationConfig(initConnectionTimeout), clientParams)

	c.swfClient = client.NewScheduledWorkflowClientOrFatal(common.GetDurationConfig(initConnectionTimeout), clientParams)

	c.k8sCoreClient = client.CreateKubernetesCoreOrFatal(common.GetDurationConfig(initConnectionTimeout), clientParams)

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

	return nil
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

	var dialector gorm.Dialector
	switch driverName {
	case "mysql":
		// DefaultStringSize dictates non-indexable string fields map to VARCHAR(255) for backward compatibility with GORM v1.
		dialector = mysql.New(mysql.Config{
			DSN:               arg,
			DefaultStringSize: 255,
		})
	case "pgx":
		dialector = postgres.Open(arg)
	default:
		glog.Fatalf("Unsupported driver %v", driverName)
	}

	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	db, err := gorm.Open(dialector, &gorm.Config{})
	util.TerminateIfError(err)

	dialect := GetDialect(driverName)

	legacy, err := isLegacySchema(db)
	if err != nil {
		glog.Fatalf("failed to detect schema version: %v", err)
	}
	if legacy {
		util.TerminateIfError(runLegacyUpgradeFlow(db, dialect))
	} else {
		util.TerminateIfError(runFreshInstallFlow(db))
	}

	newdb, err := db.DB()
	if err != nil {
		glog.Fatalf("Failed to retrieve *sql.DB from gorm.DB. Error: %v", err)
	}
	return storage.NewDB(newdb, storage.NewMySQLDialect())
}

// Initializes Database driver. Use `driverName` to indicate which type of DB to use:
// 1) "mysql" for MySQL
// 2) "pgx" for PostgreSQL
func initDBDriver(driverName string, initConnectionTimeout time.Duration) string {
	var sqlConfig, dbName string
	var mysqlConfig *mysqlStd.Config
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
	dialect := GetDialect(driverName)
	operation = func() error {
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
		if ignoreAlreadyExistError(dialect, err) != nil {
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

func isLegacySchema(db *gorm.DB) (bool, error) {
	if !db.Migrator().HasTable(&model.Pipeline{}) {
		glog.Infof("Pipelines table not found. Assuming fresh install.")
		return false, nil
	}
	length, ok, err := getColumnLength(db, &model.Pipeline{}, "UUID")
	if err != nil {
		return false, fmt.Errorf("detect schema version: %w", err)
	}
	return !ok || length > 64, nil
}

func runLegacyUpgradeFlow(db *gorm.DB, dialect SQLDialect) error {
	glog.Infof("Detected legacy schema. Running upgrade flow.")
	// Step 1: decide whether to backfill pipeline_versions
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
	// Step 2: block upgrade if legacy data too long
	if err := runPreflightLengthChecks(db, dialect, validation.LengthSpecs); err != nil {
		return fmt.Errorf("preflight length check failed: %w", err)
	}

	// Step 3: drop all indexes and constraints except primary key which blocks shrinking columns
	if err := DropAllConstraintsAndIndexes(db, dialect.Name); err != nil {
		return fmt.Errorf("drop constraints/indexes failed: %w", err)
	}

	// Step 4: shrink fields to meet new length constraints
	// NOTE: In GORM v2, AutoMigrate performs full reconciliation for most fields,
	// including type, size, and nullability. However, it will silently skip
	// primary key columns due to database constraints.
	//
	// Therefore, shrinkColumns() is retained to ensure primary key fields like UUID
	// are explicitly resized. While redundant for non-primary fields, shrinkColumns()
	// shares a common metadata source (LengthSpecs) with API-layer validation,
	// which helps avoid drift between schema and runtime logic.

	if err := shrinkColumns(db, validation.LengthSpecs); err != nil {
		return fmt.Errorf("shrink columns failed: %w", err)
	}

	// Step 5: automigrate will add DisplayName and all constraints and indices.
	err := db.AutoMigrate(
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

	if ignoreAlreadyExistError(dialect, err) != nil {
		return fmt.Errorf("failed to initialize the databases. Error: %w", err)
	}

	// Step 6: data backfill
	// Data backfill for pipeline_versions if this is the first time for
	// pipeline_versions to enter mlpipeline DB.
	if initializePipelineVersions {
		initPipelineVersionsFromPipelines(db)
	}
	err = backfillExperimentIDToRunTable(db)
	if err != nil {
		return fmt.Errorf("failed to backfill experiment UUID in run_details table: %s", err)
	}

	if err := db.Migrator().AlterColumn(&model.Pipeline{}, "Description"); err != nil {
		return fmt.Errorf("failed to update pipeline description type. Error: %s", err)
	}

	return nil
}

func runFreshInstallFlow(db *gorm.DB) error {
	glog.Infof("Detected fresh install. Running AutoMigrate.")

	if err := db.AutoMigrate(
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
	); err != nil {
		return fmt.Errorf("AutoMigrate failed: %w", err)
	}

	return nil
}

// getColumnLength returns the declared length for a column using GORM ColumnTypes.
// If the dialect/type doesn't report a length (e.g., TEXT), ok=false.
func getColumnLength(db *gorm.DB, mdl interface{}, column string) (length int64, ok bool, err error) {
	colTypes, err := db.Migrator().ColumnTypes(mdl)
	if err != nil {
		return 0, false, err
	}
	for _, ct := range colTypes {
		if strings.EqualFold(ct.Name(), column) {
			l, okLen := ct.Length()
			return l, okLen, nil
		}
	}
	return 0, false, nil
}

// runPreflightLengthChecks scans existing data and aborts upgrade if any row exceeds the new Max length.
// It must be called BEFORE AutoMigrate/DDL that shrinks column definitions.
func runPreflightLengthChecks(db *gorm.DB, dialect SQLDialect, specs []validation.ColLenSpec) error {
	quote := dialect.QuoteIdentifier

	for _, s := range specs {
		if !db.Migrator().HasTable(s.Model) {
			continue
		}
		tableName, dbCol, err := FieldMeta(db, s.Model, s.Field)
		if err != nil {
			return fmt.Errorf("failed to resolve meta for %T.%s: %w", s.Model, s.Field, err)
		}

		var cnt int64
		lengthFn := dialect.LengthFunc
		where := fmt.Sprintf("%s(%s) > ?", lengthFn, quote(dbCol))
		if err := db.Table(tableName).Where(where, s.Max).Count(&cnt).Error; err != nil {
			return fmt.Errorf("preflight length check failed for %s.%s (count): %w", tableName, dbCol, err)
		}
		if cnt == 0 {
			continue
		}

		type rowSample struct {
			Val string
		}
		var samples []rowSample
		if err := db.Table(tableName).
			Select(dbCol+" as Val").
			Where(where, s.Max).
			Limit(5).
			Scan(&samples).Error; err != nil {
			return fmt.Errorf("preflight length check failed for %s.%s (sample): %w", tableName, dbCol, err)
		}

		var preview []string
		for _, sm := range samples {
			if len(sm.Val) > 50 {
				preview = append(preview, sm.Val[:50]+"…")
			} else {
				preview = append(preview, sm.Val)
			}
		}

		return fmt.Errorf(`[Preflight] %s.%s has %d rows with length > %d.
		Reason: This column must stay indexable (e.g. MySQL utf8mb4 index key ≤ 767 bytes).Thus, KFP enforces a max of %d chars.
		Action: Shorten these values before upgrading.
		Find offenders with:
		SELECT UUID, CHAR_LENGTH(%[2]s) AS L FROM %[1]s WHERE CHAR_LENGTH(%[2]s) > %[3]d;
		Examples: %v`,
			tableName, dbCol, cnt, s.Max, s.Max, preview)
	}
	return nil
}

// FieldMeta returns the table name and DB column name for the given model+field.
func FieldMeta(db *gorm.DB, mdl interface{}, field string) (table string, dbCol string, err error) {
	stmt := &gorm.Statement{DB: db}
	if err = stmt.Parse(mdl); err != nil {
		return "", "", err
	}
	f, ok := stmt.Schema.FieldsByName[field]
	if !ok {
		return stmt.Table, "", fmt.Errorf("field %s not found in %T", field, mdl)
	}
	return stmt.Table, f.DBName, nil
}

func DropAllConstraintsAndIndexes(db *gorm.DB, driverName string) error {
	switch driverName {
	case "mysql":
		return dropAllMySQLConstraintsAndIndexes(db)
	case "pgx":
		// PostgreSQL not yet supported. No-op for now.
		return nil
	default:
		return fmt.Errorf("DropAllConstraintsAndIndexes not supported for driver: %s", driverName)
	}
}

// dropAllMySQLConstraintsAndIndexes drops all foreign key constraints, unique constraints (except PRIMARY), and non-primary indexes from all tables in the current MySQL database.
func dropAllMySQLConstraintsAndIndexes(db *gorm.DB) error {
	tables := []string{}
	if err := db.Raw("SHOW TABLES").Scan(&tables).Error; err != nil {
		return fmt.Errorf("failed to list tables: %w", err)
	}

	for _, table := range tables {
		// Drop foreign key constraints
		var foreignKeys []struct {
			ConstraintName string `gorm:"column:CONSTRAINT_NAME"`
		}
		err := db.Raw(fmt.Sprintf(`
			SELECT CONSTRAINT_NAME
			FROM information_schema.TABLE_CONSTRAINTS
			WHERE TABLE_SCHEMA = DATABASE()
			  AND TABLE_NAME = '%s'
			  AND CONSTRAINT_TYPE = 'FOREIGN KEY'`, table)).
			Scan(&foreignKeys).Error
		if err != nil {
			glog.Warningf("failed to list foreign keys for table %s: %v", table, err)
			continue
		}

		// FK constraints
		for _, fk := range foreignKeys {
			glog.Infof("Dropping foreign key %s on table %s", fk.ConstraintName, table)
			if err := db.Exec(fmt.Sprintf(
				"ALTER TABLE `%s` DROP FOREIGN KEY `%s`", table, fk.ConstraintName,
			)).Error; err != nil {
				return fmt.Errorf("failed to drop foreign key %s on table %s: %w", fk.ConstraintName, table, err)
			}
		}
	}

	// Drop UNIQUE constraints except PRIMARY KEY
	rows2, err := db.Raw(`
		SELECT constraint_name, table_name
		FROM information_schema.table_constraints
		WHERE constraint_schema = DATABASE()
		  AND constraint_type = 'UNIQUE'
		  AND constraint_name != 'PRIMARY'
	`).Rows()
	if err != nil {
		return fmt.Errorf("failed to list unique constraints: %w", err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var constraintName, tableName string
		if err := rows2.Scan(&constraintName, &tableName); err != nil {
			return fmt.Errorf("failed to scan unique constraint row: %w", err)
		}
		dropStmt := fmt.Sprintf("ALTER TABLE `%s` DROP INDEX `%s`", tableName, constraintName)
		glog.Infof("Dropping unique constraint: %s", dropStmt)
		if err := db.Exec(dropStmt).Error; err != nil {
			return fmt.Errorf("failed to drop unique constraint %s on table %s: %w", constraintName, tableName, err)
		}
	}

	// Drop non-primary indexes
	for _, table := range tables {
		var indexes []struct {
			KeyName string `gorm:"column:Key_name"`
		}
		err = db.Raw(fmt.Sprintf("SHOW INDEX FROM `%s`", table)).Scan(&indexes).Error
		if err != nil {
			glog.Warningf("failed to list indexes for table %s: %v", table, err)
			continue
		}
		seen := make(map[string]bool)
		for _, idx := range indexes {
			if idx.KeyName == "PRIMARY" || seen[idx.KeyName] {
				continue
			}
			seen[idx.KeyName] = true

			glog.Infof("Dropping index %s on table %s", idx.KeyName, table)
			if err := db.Exec(fmt.Sprintf(
				"DROP INDEX `%s` ON `%s`", idx.KeyName, table,
			)).Error; err != nil {
				return fmt.Errorf("failed to drop index %s on table %s: %w", idx.KeyName, table, err)
			}
		}
	}
	return nil
}

func shrinkColumns(db *gorm.DB, specs []validation.ColLenSpec) error {
	for _, s := range specs {
		if !db.Migrator().HasTable(s.Model) {
			continue
		}
		if err := ensureColumnLength(db, s); err != nil {
			return err
		}
	}
	return nil
}

func ensureColumnLength(db *gorm.DB, spec validation.ColLenSpec) error {

	tableName, dbCol, err := FieldMeta(db, spec.Model, spec.Field)
	if err != nil {
		return fmt.Errorf("failed to resolve meta for %T.%s: %w", spec.Model, spec.Field, err)
	}

	// Current length
	curLen, haveLen, err := getColumnLength(db, spec.Model, dbCol)
	if err != nil {
		return fmt.Errorf("columnTypes read failed for %s.%s: %w", tableName, dbCol, err)
	}
	if haveLen && curLen <= int64(spec.Max) {
		return nil
	}

	// Alter via GORM
	if err := db.Migrator().AlterColumn(spec.Model, spec.Field); err != nil {
		return fmt.Errorf("AlterColumn failed for %s.%s (field=%s): %w", tableName, dbCol, spec.Field, err)
	}

	// Verify after alter
	newLen, haveLen2, err := getColumnLength(db, spec.Model, dbCol)
	if err != nil {
		return fmt.Errorf("post-AlterColumn columnTypes read failed for %s.%s: %w", tableName, dbCol, err)
	}
	if haveLen2 && newLen > int64(spec.Max) {
		return fmt.Errorf("after AlterColumn, %s.%s length=%d (> %d)", tableName, dbCol, newLen, spec.Max)
	}
	return nil
}

func initMinioClient(ctx context.Context, initConnectionTimeout time.Duration) storage.ObjectStoreInterface {
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
	createMinioBucket(ctx, minioClient, bucketName, minioServiceRegion)

	return storage.NewMinioObjectStore(&storage.MinioClient{Client: minioClient}, bucketName, pipelinePath, disableMultipart)
}

func createMinioBucket(ctx context.Context, minioClient *minio.Client, bucketName, region string) {
	// Check to see if it exists, and we have permission to access it.
	exists, err := minioClient.BucketExists(ctx, bucketName)
	if err != nil {
		glog.Fatalf("Failed to check if object store bucket exists. Error: %v", err)
	}
	if exists {
		glog.Infof("We already own %s\n", bucketName)
		return
	}
	// Create bucket if it does not exist
	err = minioClient.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{Region: region})
	if err != nil {
		glog.Fatalf("Failed to create object store bucket. Error: %v", err)
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
func NewClientManager(options *Options) (*ClientManager, error) {
	clientManager := &ClientManager{}
	err := clientManager.init(options)
	if err != nil {
		return nil, err
	}

	return clientManager, nil
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
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	rows, err := sqlDB.Query("SELECT \"ExperimentUUID\" FROM run_details WHERE \"ExperimentUUID\" = '' LIMIT 1")
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

	_, err = sqlDB.Exec(`
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
func ignoreAlreadyExistError(dialect SQLDialect, err error) error {
	if err != nil && strings.Contains(err.Error(), dialect.ExistDatabaseErrHint) {
		return nil
	}
	return err
}
