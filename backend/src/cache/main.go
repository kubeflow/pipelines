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
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/cache/server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

const (
	TLSDir             string = "/etc/webhook/certs"
	TLSCertFileDefault string = "cert.pem"
	TLSKeyFileDefault  string = "key.pem"
)

const (
	MutateAPI   string = "/mutate"
	WebhookPort string = ":8443"
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

type S3Params struct {
	serviceHost      string
	servicePort      string
	region           string
	accessKey        string
	secretKey        string
	isServiceSecure  bool
	bucketName       string
	pipelinePath     string
	disableMultipart bool
}

func main() {
	var params WhSvrDBParameters
	var clientParams util.ClientParameters
	var s3params S3Params
	var certFile string
	var keyFile string

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

	s3params = parseS3Flags()
	//s3 endpoint params
	flag.Parse()

	log.Println("Initing client manager....")
	clientManager := NewClientManager(params, clientParams, s3params)
	ctx := context.Background()
	go server.WatchPods(ctx, params.namespaceToWatch, &clientManager)

	certPath := filepath.Join(TLSDir, certFile)
	keyPath := filepath.Join(TLSDir, keyFile)

	mux := http.NewServeMux()
	mux.Handle(MutateAPI, server.AdmitFuncHandler(server.MutatePodIfCached, &clientManager))
	server := &http.Server{
		// We listen on port 8443 such that we do not need root privileges or extra capabilities for this server.
		// The Service object will take care of mapping this port to the HTTPS port 443.
		Addr:    WebhookPort,
		Handler: mux,
	}
	log.Fatal(server.ListenAndServeTLS(certPath, keyPath))
}

func parseS3Flags() S3Params {
	var params S3Params
	flag.StringVar(&params.serviceHost, "s3_service_host", common.GetFromStringWithDefault(os.Getenv("SERVICE_HOST"), "minio-service.kubeflow.svc"), "hostname of S3.")
	flag.StringVar(&params.servicePort, "s3_service_port", common.GetFromStringWithDefault(os.Getenv("SERVICE_PORT"), "9000"), "port of S3.")
	flag.StringVar(&params.region, "s3_region", common.GetFromStringWithDefault(os.Getenv("SERVICE_REGION"), ""), "Region of S3 service.")
	flag.StringVar(&params.accessKey, "s3_access_key", os.Getenv("OBJECTSTORECONFIG_ACCESSKEY"), "accessKey of S3.")
	flag.StringVar(&params.secretKey, "s3_secret_key", os.Getenv("OBJECTSTORECONFIG_SECRETACCESSKEY"), "secretKey of S3.")
	flag.BoolVar(&params.isServiceSecure, "s3_is_service_secure",
		common.GetBoolFromStringWithDefault(os.Getenv("SERVICE_SECURE"), false), "is ssl enabled on S3.")
	flag.StringVar(&params.bucketName, "s3_bucketname", os.Getenv("PIPELINE_BUCKET_NAME"), "default bucketname on S3.")
	flag.StringVar(&params.pipelinePath, "s3_pipelinepath", "pipelines", "pipelinepath of S3.")
	flag.BoolVar(&params.disableMultipart, "s3_disable_multipart",
		common.GetBoolConfigWithDefault(os.Getenv("PIPELINE_BUCKET_DISABLE_MULTIPART"), true), "if multipart request are disabled on S3.")
	log.Printf("S3Params %v\n", params)
	return params
}
