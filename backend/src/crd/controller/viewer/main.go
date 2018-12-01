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

	viewerV1alpha1 "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/viewer/v1alpha1"
	kubeflowClientSet "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/backend/src/crd/pkg/signals"
	"github.com/kubernetes-sigs/controller-runtime/pkg/builder"
	_ "k8s.io/api/apps/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	_ "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var (
	masterURL = flag.String("master_url", "", "Address of the Kubernetes API server.")
	kubecfg   = flag.String("kubecfg", "", "Path to a valid kubeconfig.")
)

type ViewerReconciler struct {
}

type Nothing struct{}

func (v *ViewerReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	glog.Infof("Got request: %+v", req)
	return reconcile.Result{}, nil
}

func main() {
	flag.Parse()

	viewerV1alpha1.AddToScheme(scheme.Scheme)

	cfg, err := clientcmd.BuildConfigFromFlags(*masterURL, *kubecfg)
	if err != nil {
		log.Fatalf("Failed to build valid config from supplied flags: %v", err)
	}

	_ = kubernetes.NewForConfigOrDie(cfg)
	_ = kubeflowClientSet.NewForConfigOrDie(cfg).ViewerV1alpha1()

	mgr, err := builder.SimpleController().
		ForType(&viewerV1alpha1.Viewer{}).
		WithConfig(cfg).
		Build(&ViewerReconciler{})

	if err != nil {
		log.Fatal(err)
	}

	// mgr, err := manager.New(cfg, manager.Options{})
	// if err != nil {
	// 	log.Fatal(err)
	// }

	glog.Info("Starting controller!")
	if err := mgr.Start(signals.SetupSignalHandler()); err != nil {
		log.Fatalf("Failed to start viewer controller: %v", err)
	}
}
