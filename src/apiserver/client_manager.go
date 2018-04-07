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
	"ml/src/client"
	"ml/src/message"
	"ml/src/storage"
	"ml/src/util"

	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
)

const (
	minioServiceHost = "MINIO_SERVICE_SERVICE_HOST"
	minioServicePort = "MINIO_SERVICE_SERVICE_PORT"
	mysqlServiceHost = "MYSQL_SERVICE_HOST"
	mysqlServicePort = "MYSQL_SERVICE_PORT"
	dbName           = "mlpipeline"
)

// Container for all service clients
type ClientManager struct {
	db             *gorm.DB
	packageStore   storage.PackageStoreInterface
	pipelineStore  storage.PipelineStoreInterface
	jobStore       storage.JobStoreInterface
	packageManager storage.PackageManagerInterface
	time           util.TimeInterface
}

func (clientManager *ClientManager) Init() {
	glog.Infof("Initializing client manager")

	db := initDBClient()

	// time
	clientManager.time = util.NewRealTime()

	// Initialize package store
	clientManager.db = db
	clientManager.packageStore = storage.NewPackageStore(db)

	// Initialize pipeline store
	clientManager.db = db
	clientManager.pipelineStore = storage.NewPipelineStore(db, clientManager.time)

	// Initialize job store
	wfClient := client.CreateWorkflowClientOrFatal()
	clientManager.jobStore = storage.NewJobStore(db, wfClient, clientManager.time)

	// Initialize package manager.
	clientManager.packageManager = initMinioClient()

	glog.Infof("Client manager initialized successfully")
}

func (clientManager *ClientManager) End() {
	clientManager.db.Close()
}

func initDBClient() *gorm.DB {
	driverName := getConfig("DBConfig.DriverName")
	var arg string
	var db *gorm.DB

	switch driverName {
	case "mysql":
		arg = initMysql(driverName)
	case "sqlite3":
		arg = getConfig("DBConfig.DataSourceName")
	default:
		glog.Fatalf("Driver %v is not supported", driverName)
	}

	// db is safe for concurrent use by multiple goroutines
	// and maintains its own pool of idle connections.
	db, err := gorm.Open(driverName, arg)
	util.TerminateIfError(err)

	// Create table
	db.AutoMigrate(&message.Package{}, &message.Pipeline{},
		&message.Parameter{}, &message.Job{})
	return db
}

// Initialize the connection string for connecting to Mysql database
// Format would be something like root@tcp(ip:port)/dbname?charset=utf8&loc=Local&parseTime=True
func initMysql(driverName string) string {
	mysqlConfig := client.CreateMySQLConfig(
		"root",
		getConfig(mysqlServiceHost),
		getConfig(mysqlServicePort),
		"")
	db, err := sql.Open(driverName, mysqlConfig.FormatDSN())
	defer db.Close()
	util.TerminateIfError(err)

	// Create database if not exist
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))
	util.TerminateIfError(err)
	mysqlConfig.DBName = dbName
	return mysqlConfig.FormatDSN()
}

func initMinioClient() storage.PackageManagerInterface {
	// Create minio client.
	minioServiceHost := getConfig(minioServiceHost)
	minioServicePort := getConfig(minioServicePort)
	accessKey := getConfig("PackageManagerConfig.AccessKey")
	secretKey := getConfig("PackageManagerConfig.SecretAccessKey")
	bucketName := getConfig("PackageManagerConfig.BucketName")

	minioClient := client.CreateMinioClientOrFatal(minioServiceHost, minioServicePort, accessKey,
		secretKey)

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
	return storage.NewMinioPackageManager(&storage.MinioClient{Client: minioClient}, bucketName)
}

// newClientManager creates and Init a new instance of ClientManager
func newClientManager() ClientManager {
	clientManager := ClientManager{}
	clientManager.Init()

	return clientManager
}
