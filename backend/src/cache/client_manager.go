// Copyright 2020 The Kubeflow Authors
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
	"log"
	"strconv"
	"strings"
	"time"

	"encoding/json"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common/sql/dialect"
	"github.com/kubeflow/pipelines/backend/src/cache/client"
	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/cache/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const (
	DefaultConnectionTimeout = "6m"
)

type ClientManager struct {
	db            *storage.DB
	cacheStore    storage.ExecutionCacheStoreInterface
	k8sCoreClient client.KubernetesCoreInterface
	time          util.TimeInterface
}

func (c *ClientManager) CacheStore() storage.ExecutionCacheStoreInterface {
	return c.cacheStore
}

func (c *ClientManager) KubernetesCoreClient() client.KubernetesCoreInterface {
	return c.k8sCoreClient
}

func (c *ClientManager) Close() {
	sqlDB, err := c.db.DB.DB()
	if err != nil {
		log.Printf("Failed to retrieve underlying sql.DB: %v", err)
		return
	}
	sqlDB.Close()
}

func (c *ClientManager) init(params WhSvrDBParameters, clientParams util.ClientParameters) {
	timeoutDuration, _ := time.ParseDuration(DefaultConnectionTimeout)
	db, d := initDBClient(params, timeoutDuration)

	c.time = util.NewRealTime()
	c.db = db
	c.cacheStore = storage.NewExecutionCacheStore(db, c.time, d)
	c.k8sCoreClient = client.CreateKubernetesCoreOrFatal(timeoutDuration, clientParams)
}

func initDBClient(params WhSvrDBParameters, initConnectionTimeout time.Duration) (*storage.DB, dialect.DBDialect) {
	driverName := params.dbDriver

	dbDialect := dialect.NewDBDialect(driverName)
	arg := initDBDriver(params, initConnectionTimeout)

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
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	util.TerminateIfError(err)

	// Create table
	err = gormDB.AutoMigrate(&model.ExecutionCache{})
	if err != nil {
		glog.Fatalf("Failed to initialize the databases.")
	}

	err = gormDB.Migrator().AlterColumn(&model.ExecutionCache{}, "ExecutionOutput")
	if err != nil {
		glog.Fatalf("Failed to update the execution output type. Error: %s", err)
	}
	err = gormDB.Migrator().AlterColumn(&model.ExecutionCache{}, "ExecutionTemplate")
	if err != nil {
		glog.Fatalf("Failed to update the execution template type. Error: %s", err)
	}

	var tableNames []string
	gormDB.Raw(`show tables`).Pluck("Tables_in_caches", &tableNames)
	for _, tableName := range tableNames {
		log.Printf("%s", tableName)
	}

	// TODO(yunkai): Migrate cache storage to dialect-aware initialization (similar to apiserver) and remove this NewDB wrapper when unified.
	return storage.NewDB(gormDB), dbDialect
}

func initDBDriver(params WhSvrDBParameters, initConnectionTimeout time.Duration) string {
	switch params.dbDriver {
	case "mysql":
		var mysqlExtraParams = map[string]string{}
		data := []byte(params.dbExtraParams)
		json.Unmarshal(data, &mysqlExtraParams)
		mysqlConfig := client.CreateMySQLConfig(
			params.dbUser,
			params.dbPwd,
			params.dbHost,
			params.dbPort,
			"",
			params.dbGroupConcatMaxLen,
			mysqlExtraParams,
		)

		var db *sql.DB
		var err error
		var operation = func() error {
			db, err = sql.Open(params.dbDriver, mysqlConfig.FormatDSN())
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
		dbName := params.dbName
		operation = func() error {
			_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))
			if err != nil {
				return err
			}
			log.Printf("Database created")
			return nil
		}
		b = backoff.NewExponentialBackOff()
		b.MaxElapsedTime = initConnectionTimeout
		err = backoff.Retry(operation, b)

		operation = func() error {
			_, err = db.Exec(fmt.Sprintf("USE %s", dbName))
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
	case "pgx":
		port, err := strconv.Atoi(params.dbPort)
		if err != nil {
			glog.Fatalf("Invalid port for PostgreSQL: %v", err)
		}
		// Connect without target DB first
		dsnNoDB := client.CreatePostgreSQLConfig(params.dbUser, params.dbPwd, params.dbHost, "postgres", uint16(port))
		var db *sql.DB
		var operation = func() error {
			db, err = sql.Open(params.dbDriver, dsnNoDB)
			if err != nil {
				return err
			}
			return db.Ping()
		}
		b := backoff.NewExponentialBackOff()
		b.MaxElapsedTime = initConnectionTimeout
		err = backoff.Retry(operation, b)
		util.TerminateIfError(err)

		// Create database, ignoring "already exists" error
		_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", params.dbName))
		if err != nil && !strings.Contains(err.Error(), "already exists") {
			db.Close()
			glog.Fatalf("Failed to create database: %v", err)
		}
		db.Close()

		// Return DSN with target DB
		return client.CreatePostgreSQLConfig(params.dbUser, params.dbPwd, params.dbHost, params.dbName, uint16(port))
	default:
		glog.Fatalf("Driver %v is not supported", params.dbDriver)
	}
	return ""
}

func NewClientManager(params WhSvrDBParameters, clientParams util.ClientParameters) ClientManager {
	clientManager := ClientManager{}
	clientManager.init(params, clientParams)

	return clientManager
}
