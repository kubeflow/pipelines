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

// Package main is the main binary for running the Viewer Kubernetes CRD
// controller.
package main

import (
	"flag"
	"log"

	"github.com/golang/glog"

	"github.com/kubeflow/pipelines/backend/src/crd/controller/viewer/reconciler"
	"github.com/kubeflow/pipelines/backend/src/crd/pkg/signals"

	viewerV1beta1 "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/viewer/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp" // Needed for GCP authentication.
)

var (
	masterURL     = flag.String("master_url", "", "Address of the Kubernetes API server.")
	kubecfg       = flag.String("kubecfg", "", "Path to a valid kubeconfig.")
	maxNumViewers = flag.Int("max_num_viewers", 50,
		"Maximum number of viewer instances allowed within "+
			"the cluster before the controller starts deleting the oldest one.")
	namespace = flag.String("namespace", "kubeflow",
		"Namespace within which CRD controller is running. Default is "+
			"kubeflow.")
)

func main() {
	flag.Parse()

	cfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubecfg)
	if err != nil {
		log.Fatalf("Failed to build valid config from supplied flags: %v", err)
	}

	cli, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		log.Fatalf("Failed to build Kubernetes runtime client: %v", err)
	}

	viewerV1beta1.AddToScheme(scheme.Scheme)
	opts := &reconciler.Options{MaxNumViewers: *maxNumViewers}
	reconciler, err := reconciler.New(cli, scheme.Scheme, opts)
	if err != nil {
		log.Fatalf("Failed to create a Viewer Controller: %v", err)

	}

	// Create a controller that is in charge of Viewer types, and also responds to
	// changes to any deployment and services that is owned by any Viewer instance.
	mgr, err := manager.New(cfg, manager.Options{Namespace: *namespace})
	if err != nil {
		log.Fatal(err)
	}

	_, err = builder.ControllerManagedBy(mgr).
		ForType(&viewerV1beta1.Viewer{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		WithConfig(cfg).
		Build(reconciler)

	if err != nil {
		log.Fatal(err)
	}

	glog.Info("Starting controller for the Viewer CRD")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatalf("Failed to start controller: %v", err)
	}
}
