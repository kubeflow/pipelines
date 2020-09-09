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
	"flag"
	"time"

	workflowclientSet "github.com/argoproj/argo/pkg/client/clientset/versioned"
	workflowinformers "github.com/argoproj/argo/pkg/client/informers/externalversions"
	"github.com/kubeflow/pipelines/backend/src/agent/persistence/client"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	swfclientset "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"
	"github.com/kubeflow/pipelines/backend/src/crd/pkg/signals"
	log "github.com/sirupsen/logrus"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterURL                     string
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
)

const (
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
)

func main() {
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	swfClient, err := swfclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building schedule clientset: %s", err.Error())
	}

	workflowClient, err := workflowclientSet.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building workflow clientset: %s", err.Error())
	}

	var swfInformerFactory swfinformers.SharedInformerFactory
	var workflowInformerFactory workflowinformers.SharedInformerFactory
	if namespace == "" {
		swfInformerFactory = swfinformers.NewSharedInformerFactory(swfClient, time.Second*30)
		workflowInformerFactory = workflowinformers.NewSharedInformerFactory(workflowClient, time.Second*30)
	} else {
		swfInformerFactory = swfinformers.NewFilteredSharedInformerFactory(swfClient, time.Second*30, namespace, nil)
		workflowInformerFactory = workflowinformers.NewFilteredSharedInformerFactory(workflowClient, time.Second*30, namespace, nil)
	}

	pipelineClient, err := client.NewPipelineClient(
		initializeTimeout,
		timeout,
		mlPipelineAPIServerBasePath,
		mlPipelineAPIServerName,
		mlPipelineServiceHttpPort,
		mlPipelineServiceGRPCPort)
	if err != nil {
		log.Fatalf("Error creating ML pipeline API Server client: %v", err)
	}

	controller := NewPersistenceAgent(
		swfInformerFactory,
		workflowInformerFactory,
		pipelineClient,
		util.NewRealTime())

	go swfInformerFactory.Start(stopCh)
	go workflowInformerFactory.Start(stopCh)

	if err = controller.Run(numWorker, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, kubeconfigFlagName, "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, masterFlagName, "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.DurationVar(&initializeTimeout, initializationTimeoutFlagName, 2*time.Minute, "Duration to wait for initialization of the ML pipeline API server.")
	flag.DurationVar(&timeout, timeoutFlagName, 1*time.Minute, "Duration to wait for calls to complete.")
	flag.StringVar(&mlPipelineAPIServerName, mlPipelineAPIServerNameFlagName, "ml-pipeline", "Name of the ML pipeline API server.")
	flag.StringVar(&mlPipelineServiceHttpPort, mlPipelineAPIServerHttpPortFlagName, "8888", "Http Port of the ML pipeline API server.")
	flag.StringVar(&mlPipelineServiceGRPCPort, mlPipelineAPIServerGRPCPortFlagName, "8887", "GRPC Port of the ML pipeline API server.")
	flag.StringVar(&mlPipelineAPIServerBasePath, mlPipelineAPIServerBasePathFlagName,
		"/apis/v1beta1", "The base path for the ML pipeline API server.")
	flag.StringVar(&namespace, namespaceFlagName, "", "The namespace name used for Kubernetes informers to obtain the listers.")
	flag.Int64Var(&ttlSecondsAfterWorkflowFinish, ttlSecondsAfterWorkflowFinishFlagName, 604800 /* 7 days */, "The TTL for Argo workflow to persist after workflow finish.")
	flag.IntVar(&numWorker, numWorkerName, 2, "Number of worker for sync job.")
}
