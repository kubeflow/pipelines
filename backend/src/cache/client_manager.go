package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"github.com/kubeflow/pipelines/backend/src/cache/client"
	"github.com/kubeflow/pipelines/backend/src/cache/model"

	"github.com/kubeflow/pipelines/backend/src/cache/storage"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const (
	DBName                   = "mlpipeline"
	DefaultConnectionTimeout = "6m"
)

type ClientManager struct {
	db         *storage.DB
	cacheStore storage.ExecutionCacheStoreInterface
	time       util.TimeInterface
}

func (c *ClientManager) CacheStore() storage.ExecutionCacheStoreInterface {
	return c.cacheStore
}

func (c *ClientManager) Close() {
	c.db.Close()
}

func (c *ClientManager) init(params WhSvrDBParameters) {
	timeoutDuration, _ := time.ParseDuration(DefaultConnectionTimeout)
	db := initDBClient(params, timeoutDuration)

	c.time = util.NewRealTime()
	c.db = db
	c.cacheStore = storage.NewExecutionCacheStore(db, c.time)

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

	return storage.NewDB(db.DB(), storage.NewMySQLDialect())
}

func initMysql(params WhSvrDBParameters, initConnectionTimeout time.Duration) string {
	mysqlConfig := client.CreateMySQLConfig(
		params.dbUser,
		params.dbPwd,
		params.dbHost,
		params.dbPort,
		"",
		params.dbGroupConcatMaxLen,
		map[string]string{},
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

	// Use database
	dbName := DBName
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

func NewClientManager(params WhSvrDBParameters) ClientManager {
	clientManager := ClientManager{}
	clientManager.init(params)

	return clientManager
}
