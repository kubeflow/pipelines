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

package config

import (
	"flag"
	"fmt"
)

var (
	Namespace                     = flag.String("namespace", "kubeflow", "The namespace Kubeflow Pipelines is deployed to")
	RunProxyTests                 = flag.Bool("useProxy", false, "Whether to run the proxy tests")
	UploadPipelinesWithKubernetes = flag.Bool("uploadPipelinesWithKubernetes", false, "Whether to use Kubernetes for uploading pipelines or use the REST API")
	MultiUserMode                 = flag.Bool("multiUserMode", false, "Is your deployment a multi-user mode deployment?")
	DefaultServiceAccountName     = flag.String("serviceAccountName", "pipeline-runner", "Name of the service account to use for authentication")
	UserServiceAccountName        = flag.String("userServiceAccountName", "default-editor", "Name of the service account to use for authentication")
	ApiScheme                     = flag.String("apiScheme", "http", "The scheme for the apiserver to use to talk")
	ApiHost                       = flag.String("apiHost", "localhost", "The hostname of the API server")
	ApiPort                       = flag.String("apiPort", "8888", "The port on which the API server is listening")
	ApiUrl                        = flag.String("apiUrl", fmt.Sprintf("%s:%s", *ApiHost, *ApiPort), "The URL of the API server (without the scheme)")
	DisableTlsCheck               = flag.Bool("disableTlsCheck", false, "Whether to use HTTPS when connecting to the API server")
	InClusterRun                  = flag.Bool("runInCluster", false, "Whether to run your tests from within the K8s cluster")
)

var IsDebugMode = flag.Bool("isDebugMode", false, "Whether to enable debug mode. Debug mode will log more diagnostics messages.")
var PodLogLimit = flag.Int64("podLogLimit", 50000000, "Limit the pod logs size to this limit")

var (
	KubeflowMode  = flag.Bool("kubeflowMode", false, "Runs tests in full Kubeflow mode")
	UserNamespace = flag.String("userNamespace", "kubeflow-user-example-com", "The namespace that will store the test resources in Kubeflow mode")
)
