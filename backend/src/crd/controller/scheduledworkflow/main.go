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
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	swfclientset "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	swfinformers "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/informers/externalversions"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/transport"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

var (
	logLevel                          string
	masterURL                         string
	kubeconfig                        string
	namespace                         string
	location                          *time.Location
	clientQPS                         float64
	clientBurst                       int
	mlPipelineAPIServerName           string
	mlPipelineServiceGRPCPort         string
	mlPipelineServiceTLSEnabled       bool
	caCertPath                        string
	metricsPort                       string
	recurringRunResyncIntervalSeconds int
	userIdentityHeader                string
	userIdentityValue                 string
	multiUser                         bool
)

const (
	// These flags match the persistence agent
	mlPipelineAPIServerBasePathFlagName   = "mlPipelineAPIServerBasePath"
	mlPipelineAPIServerNameFlagName       = "mlPipelineAPIServerName"
	mlPipelineAPIServerGRPCPortFlagName   = "mlPipelineServiceGRPCPort"
	mlPipelineAPIServerTLSEnabledFlagName = "mlPipelineServiceTLSEnabled"
	caCertPathFlagName                    = "caCertPath"
	apiTokenFile                          = "/var/run/secrets/kubeflow/tokens/scheduledworkflow-sa-token"
	leaseName                             = "kfp-scheduledworkflow"
)

func main() {
	flag.Parse()

	leaderElect, leaderElectError := leaderElectionEnabled(os.Getenv("LEADER_ELECT"))
	if leaderElectError != nil {
		log.Fatal(leaderElectError)
	}

	// Register before the informers and leader election start: they use the
	// scheme, and the controller is created later, when this replica leads.
	registerScheme()

	// set up signals so we handle the first shutdown signal gracefully
	shutdownContext := signals.SetupSignalHandler()
	stopCh := shutdownContext.Done()

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

	if recurringRunResyncIntervalSeconds <= 0 {
		log.Fatalf("recurringRunResyncInterval must be a positive number")
	}
	defaultResync := time.Second * time.Duration(recurringRunResyncIntervalSeconds)
	if namespace == "" {
		scheduleInformerFactory = swfinformers.NewSharedInformerFactory(scheduleClient, defaultResync)
	} else {
		scheduleInformerFactory = swfinformers.NewFilteredSharedInformerFactory(scheduleClient, defaultResync, namespace, nil)
	}

	grpcAddress := fmt.Sprintf("%s:%s", mlPipelineAPIServerName, mlPipelineServiceGRPCPort)

	log.Infof("Connecting the API server over GRPC at: %s", grpcAddress)
	var tlsCfg *tls.Config
	if mlPipelineServiceTLSEnabled {
		tlsCfg, err = commonutil.GetTLSConfig(caCertPath)
		if err != nil {
			log.Fatalf("Error creating tls config: %s", err.Error())
		}
	}

	apiConnection, err := commonutil.GetRPCConnectionWithTimeout(grpcAddress, tlsCfg, time.Now().Add(time.Minute))
	if err != nil {
		log.Fatalf("Error connecting to the API server after trying for one minute: %v", err)
	}

	var tokenSrc transport.ResettableTokenSource

	if _, err := os.Stat(apiTokenFile); err == nil {
		tokenSrc = transport.NewCachedFileTokenSource(apiTokenFile)
	}

	runClient := api.NewRunServiceClient(apiConnection)

	log.Info("Successfully connected to the API server")

	newController := func() *Controller {
		controller, controllerError := NewController(
			kubeClient,
			scheduleClient,
			execClient,
			runClient,
			scheduleInformerFactory,
			execInformer,
			commonutil.NewRealTime(),
			location,
			tokenSrc,
			userIdentityHeader,
			userIdentityValue,
			multiUser,
		)
		if controllerError != nil {
			log.Fatalf("Failed to instantiate the controller: %v", controllerError)
		}
		return controller
	}

	// Request both informers so that the factories start them now: with leader
	// election, the controller is created only when this replica becomes leader.
	scheduleInformerFactory.Scheduledworkflow().V1beta1().ScheduledWorkflows().Informer()
	execInformer.HasSynced()

	go scheduleInformerFactory.Start(stopCh)
	go execInformer.InformerFactoryStart(stopCh)

	go startMetricsServer()

	if !leaderElect {
		if runError := newController().Run(2, stopCh); runError != nil {
			log.Fatalf("Error running controller: %s", runError.Error())
		}
		return
	}

	leaseNamespace := os.Getenv("POD_NAMESPACE")
	if leaseNamespace == "" {
		log.Fatal("LEADER_ELECT is true but POD_NAMESPACE is not set: set POD_NAMESPACE from the Pod's metadata.namespace, or set LEADER_ELECT to false")
	}
	identity, identityError := commonutil.LeaderElectionIdentity()
	if identityError != nil {
		log.Fatalf("Error getting the leader election identity: %v", identityError)
	}
	electionError := commonutil.RunWithLeaderElection(shutdownContext, kubeClient, commonutil.LeaderElectionConfig{
		LeaseName: leaseName,
		Namespace: leaseNamespace,
		Identity:  identity,
		Run: func(leaderContext context.Context) {
			// A standby creates no controller, so it queues no work. Adding the
			// event handlers to the running informers replays every current
			// object, so the new leader starts with all schedules queued.
			if runError := newController().Run(2, leaderContext.Done()); runError != nil {
				log.Fatalf("Error running controller: %s", runError.Error())
			}
		},
		OnLost: func() {
			// Exit so that Kubernetes restarts the container as a standby. The
			// work queue has been shut down and cannot be reused, and workers
			// may still have requests in flight.
			log.Fatal("Lost the leader lease, exiting")
		},
	})
	if electionError != nil {
		log.Fatalf("Error running leader election: %v", electionError)
	}
}

// leaderElectionEnabled parses the LEADER_ELECT environment variable. It is
// read from the environment instead of a flag so that it also applies when a
// deployment overrides the image command, as the TLS install does.
func leaderElectionEnabled(value string) (bool, error) {
	if value == "" {
		return false, nil
	}
	enabled, parseError := strconv.ParseBool(value)
	if parseError != nil {
		return false, fmt.Errorf("invalid LEADER_ELECT value %q: set it to \"true\" or \"false\"", value)
	}
	return enabled, nil
}

func startMetricsServer() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	addr := ":" + metricsPort
	srv := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}

	log.Infof("Starting metrics server at %s...", addr)

	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("Metrics server failed: %v", err)
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
	flag.StringVar(&metricsPort, "metricsPort", "9090", "The port for the metrics endpoint.")
	flag.StringVar(&mlPipelineAPIServerName, mlPipelineAPIServerNameFlagName, "ml-pipeline", "Name of the ML pipeline API server.")
	flag.StringVar(&mlPipelineServiceGRPCPort, mlPipelineAPIServerGRPCPortFlagName, "8887", "GRPC Port of the ML pipeline API server.")
	flag.BoolVar(&mlPipelineServiceTLSEnabled, mlPipelineAPIServerTLSEnabledFlagName, false, "Set to true if ML pipeline API server serves over TLS.")
	flag.StringVar(&caCertPath, caCertPathFlagName, "", "CA cert to connect to the ML pipeline API server.")
	flag.IntVar(&clientBurst, "clientBurst", 10, "Maximum burst for throttle from this client.")
	flag.IntVar(&recurringRunResyncIntervalSeconds, "recurringRunResyncIntervalSeconds", 30, "The full resync interval in seconds for recurring run reconciliations.")
	flag.BoolVar(&multiUser, "multiUser", false, "Submit all scheduled runs through the API server using persisted recurring-run inputs.")
	flag.StringVar(&userIdentityHeader, "userIdentityHeader", "", "User identity metadata key for multi-user mode (e.g. kubeflow-userid). If set, the controller injects this key into the outgoing gRPC metadata when calling the API server. The key is normalized to lowercase and must be a valid gRPC metadata key.")
	flag.StringVar(&userIdentityValue, "userIdentityValue", "", "Value for the user identity gRPC metadata key (e.g. system:serviceaccount:kubeflow:ml-pipeline-scheduledworkflow).")
	var err error
	location, err = util.GetLocation()
	if err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
	log.Infof("Location: %s", location.String())
}
