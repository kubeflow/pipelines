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
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/kubeflow/pipelines/backend/src/cache/server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const (
	TLSDir             string = "/etc/webhook/certs"
	TLSCertFileDefault string = "cert.pem"
	TLSKeyFileDefault  string = "key.pem"
)

const (
	MutateAPI          string = "/mutate"
	DefaultWebhookPort int    = 8443
)

const (
	initConnectionTimeout = "6m"

	mysqlDBDriverDefault            = "mysql"
	mysqlDBHostDefault              = "mysql"
	mysqlDBPortDefault              = "3306"
	mysqlDBGroupConcatMaxLenDefault = "4194304"
)

type WhSvrDBParameters struct {
	dbDriver            string
	dbHost              string
	dbPort              string
	dbName              string
	dbUser              string
	dbPwd               string
	dbGroupConcatMaxLen string
	dbExtraParams       string
	namespaceToWatch    string
}

func main() {
	var params WhSvrDBParameters
	var clientParams util.ClientParameters
	var certFile string
	var keyFile string
	var webhookPort int

	flag.StringVar(&params.dbDriver, "db_driver", mysqlDBDriverDefault, "Database driver name, mysql is the default value")
	flag.StringVar(&params.dbHost, "db_host", mysqlDBHostDefault, "Database host name.")
	flag.StringVar(&params.dbPort, "db_port", mysqlDBPortDefault, "Database port number.")
	flag.StringVar(&params.dbName, "db_name", "cachedb", "Database name.")
	flag.StringVar(&params.dbUser, "db_user", "root", "Database user name.")
	flag.StringVar(&params.dbPwd, "db_password", "", "Database password.")
	flag.StringVar(&params.dbGroupConcatMaxLen, "db_group_concat_max_len", mysqlDBGroupConcatMaxLenDefault, "Database group concat max length.")
	flag.StringVar(&params.dbExtraParams, "db_extra_params", "", "Database extra parameters.")
	flag.StringVar(&params.namespaceToWatch, "namespace_to_watch", "kubeflow", "Namespace to watch.")
	// Use default value of client QPS (5) & burst (10) defined in
	// k8s.io/client-go/rest/config.go#RESTClientFor
	flag.Float64Var(&clientParams.QPS, "kube_client_qps", 5, "The maximum QPS to the master from this client.")
	flag.IntVar(&clientParams.Burst, "kube_client_burst", 10, "Maximum burst for throttle from this client.")
	// If you are NOT using cache deployer to create the certificate then you can use these two parameters to specify the TLS filenames
	// Eg: If you have created the certificate using cert-manager then specify tls_cert_filename=tls.crt and tls_key_filename=tls.key
	flag.StringVar(&certFile, "tls_cert_filename", TLSCertFileDefault, "The TLS certificate filename.")
	flag.StringVar(&keyFile, "tls_key_filename", TLSKeyFileDefault, "The TLS key filename.")
	flag.IntVar(&webhookPort, "listen_port", DefaultWebhookPort, "Port number on which the webhook listens.")

	flag.Parse()

	log.Println("Initing client manager....")
	clientManager := NewClientManager(params, clientParams)
	ctx := context.Background()
	go server.WatchPods(ctx, params.namespaceToWatch, &clientManager)

	certPath := filepath.Join(TLSDir, certFile)
	keyPath := filepath.Join(TLSDir, keyFile)

	mux := http.NewServeMux()
	mux.Handle(MutateAPI, server.AdmitFuncHandler(server.MutatePodIfCached, &clientManager))
	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", webhookPort),
		Handler: mux,
	}
	log.Fatal(server.ListenAndServeTLS(certPath, keyPath))
}
