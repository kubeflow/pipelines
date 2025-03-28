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
	"fmt"
	"os"
	"strings"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	swfclientset "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	logLevel                  string
	masterURL                 string
	kubeconfig                string
	namespace                 string
	location                  *time.Location
	clientQPS                 float64
	clientBurst               int
	mlPipelineAPIServerName   string
	mlPipelineServiceGRPCPort string
)

const (
	// These flags match the persistence agent
	mlPipelineAPIServerBasePathFlagName = "mlPipelineAPIServerBasePath"
	mlPipelineAPIServerNameFlagName     = "mlPipelineAPIServerName"
	mlPipelineAPIServerGRPCPortFlagName = "mlPipelineServiceGRPCPort"
	apiTokenFile                        = "/var/run/secrets/kubeflow/tokens/scheduledworkflow-sa-token"
)

func main() {
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler().Done()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}
	cfg.QPS = float32(clientQPS)
	cfg.Burst = clientBurst

	if logLevel == "" {
		logLevel = "info"
	}

	level, err := log.ParseLevel(logLevel)
	if err != nil {
		log.Fatal("Invalid log level:", err)
	}
	log.SetLevel(level)

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	scheduleClient, err := swfclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building schedule clientset: %s", err.Error())
	}

	clientParam := commonutil.ClientParameters{QPS: float64(cfg.QPS), Burst: cfg.Burst}
	execClient := commonutil.NewExecutionClientOrFatal(commonutil.ArgoWorkflow, time.Second*30, clientParam)

	var scheduleInformerFactory swfinformers.SharedInformerFactory
	execInformer := commonutil.NewExecutionInformerOrFatal(commonutil.ArgoWorkflow, namespace, time.Second*30, clientParam)
	if namespace == "" {
		scheduleInformerFactory = swfinformers.NewSharedInformerFactory(scheduleClient, time.Second*30)
	} else {
		scheduleInformerFactory = swfinformers.NewFilteredSharedInformerFactory(scheduleClient, time.Second*30, namespace, nil)
	}

	grpcAddress := fmt.Sprintf("%s:%s", mlPipelineAPIServerName, mlPipelineServiceGRPCPort)

	log.Infof("Connecting the API server over GRPC at: %s", grpcAddress)
	apiConnection, err := commonutil.GetRpcConnectionWithTimeout(grpcAddress, time.Now().Add(time.Minute))
	if err != nil {
		log.Fatalf("Error connecting to the API server after trying for one minute: %v", err)
	}

	var tokenSrc transport.ResettableTokenSource

	if _, err := os.Stat(apiTokenFile); err == nil {
		tokenSrc = transport.NewCachedFileTokenSource(apiTokenFile)
	}

	runClient := api.NewRunServiceClient(apiConnection)

	log.Info("Successfully connected to the API server")

	controller, err := NewController(
		kubeClient,
		scheduleClient,
		execClient,
		runClient,
		scheduleInformerFactory,
		execInformer,
		commonutil.NewRealTime(),
		location,
		tokenSrc,
	)
	if err != nil {
		log.Fatalf("Failed to instantiate the controller: %v", err)
	}

	go scheduleInformerFactory.Start(stopCh)
	go execInformer.InformerFactoryStart(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
}

func initEnv() {
	// Import environment variable, support nested vars e.g. OBJECTSTORECONFIG_ACCESSKEY
	replacer := strings.NewReplacer(".", "_")
	viper.SetEnvKeyReplacer(replacer)
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)
}

func init() {
	initEnv()

	flag.StringVar(&logLevel, "logLevel", "", "Defines the log level for the application.")
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&namespace, "namespace", "", "The namespace name used for Kubernetes informers to obtain the listers.")
	// Use default value of client QPS (5) & burst (10) defined in
	// k8s.io/client-go/rest/config.go#RESTClientFor
	flag.Float64Var(&clientQPS, "clientQPS", 5, "The maximum QPS to the master from this client.")
	flag.StringVar(&mlPipelineAPIServerName, mlPipelineAPIServerNameFlagName, "ml-pipeline", "Name of the ML pipeline API server.")
	flag.StringVar(&mlPipelineServiceGRPCPort, mlPipelineAPIServerGRPCPortFlagName, "8887", "GRPC Port of the ML pipeline API server.")
	flag.IntVar(&clientBurst, "clientBurst", 10, "Maximum burst for throttle from this client.")
	var err error
	location, err = util.GetLocation()
	if err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
	log.Infof("Location: %s", location.String())
}
