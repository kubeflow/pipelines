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
	"time"
)

var (
	Namespace                     = flag.String("namespace", "kubeflow", "The namespace ml pipeline deployed to")
	InitializeTimeout             = flag.Duration("initializeTimeout", 2*time.Minute, "Duration to wait for test initialization")
	RunK8sStoreIntegrationTests   = flag.Bool("runK8sStoreIntegrationTests", false, "Whether to run integration tests with K8s Native API Support")
	UseProxy                      = flag.Bool("useProxy", false, "Whether to run the proxy tests")
	CacheEnabled                  = flag.Bool("cacheEnabled", true, "Whether cache is enabled tests")
	UploadPipelinesWithKubernetes = flag.Bool("uploadPipelinesWithKubernetes", false, "Whether to use Kubernetes for uploading pipelines or use the REST API")
)

/**
 * IsDevMode
 * Differences in dev mode:
 * 1. Resources are not cleaned up when a test finishes, so that developer can debug manually.
 * 2. One step that doesn't work locally is skipped.
 */
var IsDevMode = flag.Bool("isDevMode", false, "Dev mode helps local development of integration tests")

var IsDebugMode = flag.Bool("isDebugMode", false, "Whether to enable debug mode. Debug mode will log more diagnostics messages.")
var PodLogLimit = flag.Int64("podLogLimit", 50000000, "Limit the pod logs size to this limit")

var (
	IsKubeflowMode    = flag.Bool("isKubeflowMode", false, "Runs tests in full Kubeflow mode")
	ResourceNamespace = flag.String("resourceNamespace", "", "The namespace that will store the test resources in Kubeflow mode")
)
