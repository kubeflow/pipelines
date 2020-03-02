// Copyright 2020 Google LLC
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
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"pipelines/backend/src/cache/client"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/jinzhu/gorm"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const (
	TLSDir      string = "/etc/webhook/certs"
	TLSCertFile string = "cert.pem"
	TLSKeyFile  string = "key.pem"
)

const (
	MutateAPI   string = "/mutate"
	WebhookPort string = ":8443"
)

const (
	mysqlServiceHost       = "DBConfig.Host"
	mysqlServicePort       = "DBConfig.Port"
	mysqlUser              = "DBConfig.User"
	mysqlPassword          = "DBConfig.Password"
	mysqlDBName            = "DBConfig.DBName"
	mysqlGroupConcatMaxLen = "DBConfig.GroupConcatMaxLen"
	mysqlExtraParams       = "DBConfig.ExtraParams"

	initConnectionTimeout = "InitConnectionTimeout"

	mysqlDBDriverDefault = "mysql"
	mysqlDBHostDefault   = "mysql"
	mysqlDBPortDefault   = 3306
)

type WhSvrDBParameters struct {
	dbDriver string
	dbHost   string
	dbPort   int
	dbUser   string
	dbPwd    string
}

func main() {
	var params WhSvrDBParameters
	flag.StringVar(&params.dbDriver, "db_driver", mysqlDBDriverDefault, "Database driver name, mysql is the default value")
	flag.StringVar(&params.dbHost, "db_host", mysqlDBHostDefault, "Database host name.")
	flag.IntVar(&params.dbPort, "db_port", mysqlDBPortDefault, "Database port number.")
	flag.StringVar(&params.dbUser, "db_user", "root", "Database user name.")
	flag.StringVar(&params.dbPwd, "db_password", "", "Database password.")

	certPath := filepath.Join(TLSDir, TLSCertFile)
	keyPath := filepath.Join(TLSDir, TLSKeyFile)

	mux := http.NewServeMux()
	mux.Handle(MutateAPI, admitFuncHandler(mutatePodIfCached))
	server := &http.Server{
		// We listen on port 8443 such that we do not need root privileges or extra capabilities for this server.
		// The Service object will take care of mapping this port to the HTTPS port 443.
		Addr:    WebhookPort,
		Handler: mux,
	}
	log.Fatal(server.ListenAndServeTLS(certPath, keyPath))
}

func initDBClient(initConnectionTimeout time.Duration) {
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

	// Create table
	response := db.AutoMigrate(&ExecutionCache{})
	if response.Error != nil {
		glog.Fatalf("Failed to initialize the databases.")
	}
}

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
	err = backoff.Retry(operation, b)

	defer db.Close()
	util.TerminateIfError(err)

	// Use database
	dbName := common.GetStringConfig(mysqlDBName)
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
