// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"github.com/kubeflow/pipelines/backend/src/common/util"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
)

const (
	unsetProxyArgValue = "unset"
	RootDag            = "ROOT_DAG"
	DAG                = "DAG"
	CONTAINER          = "CONTAINER"
)

var (
	logLevel   = flag.String("log_level", "1", "The verbosity level to log.")
	serverPort = flag.String("server_port", ":8080", "Server port")
)

func main() {
	flag.Parse()

	glog.Infof("Setting log level to: '%s'", *logLevel)
	err := flag.Set("v", *logLevel)
	if err != nil {
		glog.Warningf("Failed to set log level: %s", err.Error())
	}

	http.HandleFunc("/api/v1/template.execute", ExecutePlugin)
	glog.Infof("Server started at http://localhost%v", *serverPort)
	err = http.ListenAndServe(*serverPort, nil)
	if err != nil {
		glog.Warningf("Failed to start http server: %s", err.Error())
	}
}

// Use WARNING default logging level to facilitate troubleshooting.
func init() {
	flag.Set("logtostderr", "true")
	// Change the WARNING to INFO level for debugging.
	flag.Set("stderrthreshold", "WARNING")
}

func parseExecConfigJSON(k8sExecConfigJSON *string) (*kubernetesplatform.KubernetesExecutorConfig, error) {
	var k8sExecCfg *kubernetesplatform.KubernetesExecutorConfig
	if *k8sExecConfigJSON != "" {
		k8sExecCfg = &kubernetesplatform.KubernetesExecutorConfig{}
		if err := util.UnmarshalString(*k8sExecConfigJSON, k8sExecCfg); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Kubernetes config, error: %w\nKubernetesConfig: %v", err, k8sExecConfigJSON)
		}
	}
	return k8sExecCfg, nil
}

func prettyPrint(jsonStr string) string {
	var prettyJSON bytes.Buffer
	err := json.Indent(&prettyJSON, []byte(jsonStr), "", "  ")
	if err != nil {
		return jsonStr
	}
	return prettyJSON.String()
}

func newMlmdClient(mlmdServerAddress string, mlmdServerPort string, tlsCfg *tls.Config) (*metadata.Client, error) {
	return metadata.NewClient(mlmdServerAddress, mlmdServerPort, tlsCfg)
}
