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
	commonutil "github.com/googleprivate/ml/backend/src/common/util"
	swfclientset "github.com/googleprivate/ml/backend/src/crd/pkg/client/clientset/versioned"
	swfinformers "github.com/googleprivate/ml/backend/src/crd/pkg/client/informers/externalversions"
	"github.com/googleprivate/ml/backend/src/crd/pkg/signals"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	masterURL  string
	kubeconfig string
)

func main() {
	flag.Parse()

	// set up signals so we handle the first shutdown signal gracefully
	stopCh := signals.SetupSignalHandler()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		log.Fatalf("Error building kubeconfig: %s", err.Error())
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building kubernetes clientset: %s", err.Error())
	}

	scheduleClient, err := swfclientset.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building schedule clientset: %s", err.Error())
	}

	workflowClient, err := workflowclientSet.NewForConfig(cfg)
	if err != nil {
		log.Fatalf("Error building workflow clientset: %s", err.Error())
	}

	scheduleInformerFactory := swfinformers.NewSharedInformerFactory(scheduleClient, time.Second*30)
	workflowInformerFactory := workflowinformers.NewSharedInformerFactory(workflowClient, time.Second*30)

	controller := NewController(
		kubeClient,
		scheduleClient,
		workflowClient,
		scheduleInformerFactory,
		workflowInformerFactory,
		commonutil.NewRealTime())

	go scheduleInformerFactory.Start(stopCh)
	go workflowInformerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		log.Fatalf("Error running controller: %s", err.Error())
	}
}

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	flag.StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")
}
