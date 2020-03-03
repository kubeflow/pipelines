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
	"flag"
	"log"
	"net/http"
	"path/filepath"

	"github.com/kubeflow/pipelines/backend/src/cache/model"
	"github.com/kubeflow/pipelines/backend/src/cache/server"
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

	mysqlDBDriverDefault            = "mysql"
	mysqlDBHostDefault              = "mysql"
	mysqlDBPortDefault              = "3306"
	mysqlDBGroupConcatMaxLenDefault = "4194304"
)

type WhSvrDBParameters struct {
	dbDriver            string
	dbHost              string
	dbPort              string
	dbUser              string
	dbPwd               string
	dbGroupConcatMaxLen string
}

func main() {
	var params WhSvrDBParameters
	flag.StringVar(&params.dbDriver, "db_driver", mysqlDBDriverDefault, "Database driver name, mysql is the default value")
	flag.StringVar(&params.dbHost, "db_host", mysqlDBHostDefault, "Database host name.")
	flag.StringVar(&params.dbPort, "db_port", mysqlDBPortDefault, "Database port number.")
	flag.StringVar(&params.dbUser, "db_user", "root", "Database user name.")
	flag.StringVar(&params.dbPwd, "db_password", "", "Database password.")
	flag.StringVar(&params.dbGroupConcatMaxLen, "db_group_concat_max_len", mysqlDBGroupConcatMaxLenDefault, "Database password.")

	log.Println("Initing client manager....")
	clientManager := NewClientManager(params)
	testCache := model.ExecutionCache{
		ExecutionCacheKey: "test123456",
		ExecutionOutput:   "testoutput",
	}
	_, err := clientManager.cacheStore.CreateExecutionCache(&testCache)
	if err != nil {
		log.Println(err.Error())
	}

	cacheResult, err := clientManager.cacheStore.GetExecutionCache("test123456")
	if err != nil {
		log.Printf(err.Error())
	}
	log.Println(cacheResult.GetExecutionOutput())

	certPath := filepath.Join(TLSDir, TLSCertFile)
	keyPath := filepath.Join(TLSDir, TLSKeyFile)

	mux := http.NewServeMux()
	mux.Handle(MutateAPI, server.AdmitFuncHandler(server.MutatePodIfCached))
	server := &http.Server{
		// We listen on port 8443 such that we do not need root privileges or extra capabilities for this server.
		// The Service object will take care of mapping this port to the HTTPS port 443.
		Addr:    WebhookPort,
		Handler: mux,
	}
	log.Fatal(server.ListenAndServeTLS(certPath, keyPath))
}
