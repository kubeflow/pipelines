// Copyright 2018 The Kubeflow Authors
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
	"os"
	"time"

	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfclientset "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"
	"github.com/kubeflow/pipelines/backend/src/crd/pkg/signals"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	masterURL                     string
	logLevel                      string
	kubeconfig                    string
	initializeTimeout             time.Duration
	timeout                       time.Duration
	mlPipelineAPIServerName       string
	mlPipelineAPIServerPort       string
	mlPipelineAPIServerBasePath   string
	mlPipelineServiceHttpPort     string
	mlPipelineServiceGRPCPort     string
	namespace                     string
	ttlSecondsAfterWorkflowFinish int64
	numWorker                     int
	clientQPS                     float64
	clientBurst                   int
	executionType                 string
	saTokenRefreshIntervalInSecs  int64
)

const (
	logLevelFlagName                      = "logLevel"
	kubeconfigFlagName                    = "kubeconfig"
	masterFlagName                        = "master"
	initializationTimeoutFlagName         = "initializeTimeout"
	timeoutFlagName                       = "timeout"
	mlPipelineAPIServerBasePathFlagName   = "mlPipelineAPIServerBasePath"
	mlPipelineAPIServerNameFlagName       = "mlPipelineAPIServerName"
	mlPipelineAPIServerHttpPortFlagName   = "mlPipelineServiceHttpPort"
	mlPipelineAPIServerGRPCPortFlagName   = "mlPipelineServiceGRPCPort"
	namespaceFlagName                     = "namespace"
	ttlSecondsAfterWorkflowFinishFlagName = "ttlSecondsAfterWorkflowFinish"
	numWorkerName                         = "numWorker"
	clientQPSFlagName                     = "clientQPS"
	clientBurstFlagName                   = "clientBurst"
	executionTypeFlagName                 = "executionType"
	saTokenRefreshIntervalFlagName        = "saTokenRefreshIntervalInSecs"
)

const (
	DefaultConnectionTimeout              = 6 * time.Minute
	DefaultSATokenRefresherIntervalInSecs = 60 * 60 // 1 Hour in seconds
)

var (
	persistenceAgentFlags = flag.NewFlagSet("persistence_agent",flag.ContinueOnError) 
)

func main() {
	if err := persistenceAgentFlags.Parse(os.Args[1:]); err != nil {
        log.Fatalf("Failed to parse flags: %v", err)
    }

	flag.Parse()
	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	// Use the util to store the ExecutionType
	util.SetExecutionType(util.ExecutionType(executionType))

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	cfg.QPS = float32(clientQPS)
	cfg.Burst = clientBurst

	swfClient, err := swfclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building schedule clientset: %s", err.Error())
	}

	if logLevel == "" {
		logLevel = "info"
	}

	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatal("Invalid log level:", err)
	}
	log.SetLevel(level)

	clientParam := util.ClientParameters{QPS: float64(cfg.QPS), Burst: cfg.Burst}
	execInformer := util.NewExecutionInformerOrFatal(util.CurrentExecutionType(), namespace, time.Second*30, clientParam)

	var swfInformerFactory swfinformers.SharedInformerFactory
	if namespace == "" {
		swfInformerFactory = swfinformers.NewSharedInformerFactory(swfClient, time.Second*30)
	} else {
		swfInformerFactory = swfinformers.NewFilteredSharedInformerFactory(swfClient, time.Second*30, namespace, nil)
	}

	tokenRefresher := client.NewTokenRefresher(time.Duration(saTokenRefreshIntervalInSecs)*time.Second, nil)
	err = tokenRefresher.StartTokenRefreshTicker()
	if err != nil {
		log.Fatalf("Error starting Service Account Token Refresh Ticker due to: %v", err)
	}

	pipelineClient, err := client.NewPipelineClient(
		initializeTimeout,
		timeout,
		tokenRefresher,
		mlPipelineAPIServerBasePath,
		mlPipelineAPIServerName,
		mlPipelineServiceHttpPort,
		mlPipelineServiceGRPCPort)
	if err != nil {
		log.Fatalf("Error creating ML pipeline API Server client: %v", err)
	}

	mgr, err := NewPersistenceAgent(
		swfInformerFactory,
		execInformer,
		pipelineClient,
		util.NewRealTime())
	if err != nil {
		log.Fatalf("Failed to instantiate the controller: %v", err)
	}
	go swfInformerFactory.Start(stopCh)
	go execInformer.InformerFactoryStart(stopCh)

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		log.Fatalf("problem running manager: %v", err)
	}

}

func init() {
	persistenceAgentFlags.StringVar(&kubeconfig, kubeconfigFlagName, "", "Path to a kubeconfig. Only required if out-of-cluster.")
	persistenceAgentFlags.StringVar(&masterURL, masterFlagName, "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	persistenceAgentFlags.StringVar(&logLevel, logLevelFlagName, "", "Defines the log level for the application.")
	persistenceAgentFlags.DurationVar(&initializeTimeout, initializationTimeoutFlagName, 2*time.Minute, "Duration to wait for initialization of the ML pipeline API server.")
	persistenceAgentFlags.DurationVar(&timeout, timeoutFlagName, 1*time.Minute, "Duration to wait for calls to complete.")
	persistenceAgentFlags.StringVar(&mlPipelineAPIServerName, mlPipelineAPIServerNameFlagName, "ml-pipeline", "Name of the ML pipeline API server.")
	persistenceAgentFlags.StringVar(&mlPipelineServiceHttpPort, mlPipelineAPIServerHttpPortFlagName, "8888", "Http Port of the ML pipeline API server.")
	persistenceAgentFlags.StringVar(&mlPipelineServiceGRPCPort, mlPipelineAPIServerGRPCPortFlagName, "8887", "GRPC Port of the ML pipeline API server.")
	persistenceAgentFlags.StringVar(&mlPipelineAPIServerBasePath, mlPipelineAPIServerBasePathFlagName,
		"/apis/v1beta1", "The base path for the ML pipeline API server.")
	persistenceAgentFlags.StringVar(&namespace, namespaceFlagName, "", "The namespace name used for Kubernetes informers to obtain the listers.")
	persistenceAgentFlags.Int64Var(&ttlSecondsAfterWorkflowFinish, ttlSecondsAfterWorkflowFinishFlagName, 604800 /* 7 days */, "The TTL for Argo workflow to persist after workflow finish.")
	persistenceAgentFlags.IntVar(&numWorker, numWorkerName, 2, "Number of worker for sync job.")
	// Use default value of client QPS (5) & burst (10) defined in
	// k8s.io/client-go/rest/config.go#RESTClientFor
	persistenceAgentFlags.Float64Var(&clientQPS, clientQPSFlagName, 5, "The maximum QPS to the master from this client.")
	persistenceAgentFlags.IntVar(&clientBurst, clientBurstFlagName, 10, "Maximum burst for throttle from this client.")
	persistenceAgentFlags.StringVar(&executionType, executionTypeFlagName, "Workflow", "Custom Resource's name of the backend Orchestration Engine")
	// TODO use viper/config file instead. Sync `saTokenRefreshIntervalFlagName` with the value from manifest file by using ENV var.
	persistenceAgentFlags.Int64Var(&saTokenRefreshIntervalInSecs, saTokenRefreshIntervalFlagName, DefaultSATokenRefresherIntervalInSecs, "Persistence agent service account token read interval in seconds. "+
		"Defines how often `/var/run/secrets/kubeflow/tokens/kubeflow-persistent_agent-api-token` to be read")

}
