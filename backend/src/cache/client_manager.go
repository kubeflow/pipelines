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
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	apiserver_client "github.com/kubeflow/pipelines/backend/src/apiserver/client"
	apiserver_storage "github.com/kubeflow/pipelines/backend/src/apiserver/storage"
	"github.com/kubeflow/pipelines/backend/src/cache/client"
	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/cache/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const (
	DefaultConnectionTimeout = "6m"
)

type ClientManager struct {
	db            *storage.DB
	cacheStore    storage.ExecutionCacheStoreInterface
	k8sCoreClient client.KubernetesCoreInterface
	minioClient   apiserver_storage.ObjectStoreInterface
	time          util.TimeInterface
}

func (c *ClientManager) CacheStore() storage.ExecutionCacheStoreInterface {
	return c.cacheStore
}

func (c *ClientManager) KubernetesCoreClient() client.KubernetesCoreInterface {
	return c.k8sCoreClient
}

func (c *ClientManager) MinioClient() apiserver_storage.ObjectStoreInterface {
	return c.minioClient
}

func (c *ClientManager) Close() {
	c.db.Close()
}

func (c *ClientManager) init(params WhSvrDBParameters, clientParams util.ClientParameters, s3params S3Params) {
	timeoutDuration, _ := time.ParseDuration(DefaultConnectionTimeout)
	db := initDBClient(params, timeoutDuration)

	c.time = util.NewRealTime()
	c.db = db
	c.cacheStore = storage.NewExecutionCacheStore(db, c.time)
	c.k8sCoreClient = client.CreateKubernetesCoreOrFatal(timeoutDuration, clientParams)
	c.minioClient = NewMinioClient(timeoutDuration, s3params)
}

func NewMinioClient(initConnectionTimeout time.Duration, params S3Params) apiserver_storage.ObjectStoreInterface {
	glog.Infof("NewMinioClient %v", params)
	minioClient := apiserver_client.CreateMinioClientOrFatal(params.serviceHost, params.servicePort,
		params.accessKey, params.secretKey, params.isServiceSecure, params.region, initConnectionTimeout)
	return apiserver_storage.NewMinioObjectStore(&apiserver_storage.MinioClient{Client: minioClient},
		params.region, params.bucketName, params.pipelinePath, params.disableMultipart)
}

func initDBClient(params WhSvrDBParameters, initConnectionTimeout time.Duration) *storage.DB {
	driverName := params.dbDriver
	var arg string

	switch driverName {
	case mysqlDBDriverDefault:
		arg = initMysql(params, initConnectionTimeout)
	default:
		glog.Fatalf("Driver %v is not supported", driverName)
	}

	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	db, err := gorm.Open(driverName, arg)
	util.TerminateIfError(err)

	// Create table
	response := db.AutoMigrate(&model.ExecutionCache{})
	if response.Error != nil {
		glog.Fatalf("Failed to initialize the databases.")
	}

	response = db.Model(&model.ExecutionCache{}).ModifyColumn("ExecutionOutput", "longtext")
	if response.Error != nil {
		glog.Fatalf("Failed to update the execution output type. Error: %s", response.Error)
	}
	response = db.Model(&model.ExecutionCache{}).ModifyColumn("ExecutionTemplate", "longtext not null")
	if response.Error != nil {
		glog.Fatalf("Failed to update the execution template type. Error: %s", response.Error)
	}

	var tableNames []string
	db.Raw(`show tables`).Pluck("Tables_in_caches", &tableNames)
	for _, tableName := range tableNames {
		log.Printf(tableName)
	}

	return storage.NewDB(db)
}

func initMysql(params WhSvrDBParameters, initConnectionTimeout time.Duration) string {

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
}

func NewClientManager(params WhSvrDBParameters, clientParams util.ClientParameters, s3params S3Params) ClientManager {
	clientManager := ClientManager{}
	clientManager.init(params, clientParams, s3params)

	return clientManager
}
