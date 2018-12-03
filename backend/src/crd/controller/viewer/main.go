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
	"fmt"
	"context"
	"flag"
	"log"

	"github.com/golang/glog"

	viewerV1alpha1 "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/viewer/v1alpha1"
	kubeflowClientSet "github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned"
	"github.com/kubeflow/pipelines/backend/src/crd/pkg/client/clientset/versioned/typed/viewer/v1alpha1"
	"github.com/kubeflow/pipelines/backend/src/crd/pkg/signals"
	"github.com/kubernetes-sigs/controller-runtime/pkg/builder"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var (
	masterURL = flag.String("master_url", "", "Address of the Kubernetes API server.")
	kubecfg   = flag.String("kubecfg", "", "Path to a valid kubeconfig.")
)

type ViewerReconciler struct {
	client.Client
	vcli   v1alpha1.ViewerV1alpha1Interface
	scheme *runtime.Scheme
}

type Nothing struct{}

// var _ reconcile.Reconciler = &ViewerReconciler{}

func (v *ViewerReconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	glog.Infof("Got request: %+v", req)

	instance := &viewerV1alpha1.Viewer{}
	if err := v.Get(context.TODO(), req.NamespacedName, instance); err != nil {
		glog.Infof("Got error: %+v", err)
		if errors.IsNotFound(err) {
			// Object not found, return.
			// Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	glog.Infof("Got instance: %+v", instance)

	// Set up potential deployment.
	instanceName := instance.Namespace + "-" + instance.Name
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name + "-deployment",
			Namespace: instance.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"deployment": instanceName + "-deployment",
				},
			},
			Template: &corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "tensorboard",
						"instance": instanceName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							Name: instance.Name + "-pod",
							Image: "tensorflow/tensorflow:1.11.0",
							Args: []string{
								"tensorboard",
								fmt.Sprintf("--logdir=%s", instance.Spec.TensorboardSpec.LogDir),
								fmt.Sprintf("--path_prefix=/tensorboard/%s/", instanceName),
							},
							Ports: []corev1.ContainerPort{6006},
						},
					},
				},
			},


			},
		},
	}

	// Set the deployment to be owned by the instance.
	if err := controllerutil.SetControllerReference(instance, deployment, v.scheme); err != nil {
		// Error means that the deployment is already owned by some other instance.
		return reconcile.Result{}, err
	}

	if instance.Spec.Type != viewerV1alpha1.TensorboardViewer {
		glog.Infof("Unsupported spec type: %q", instance.Spec.Type)
		// Return nil to indicate nothing more to do here.
		return reconcile.Result{}, nil
	}
	instance.Spec.PodSpec.

	// Assume Tensorboard for now.

	// // i2 := &viewerV1alpha1.Viewer{}
	// i2, err := v.vcli.Viewers(req.Namespace).Get(req.Name, metav1.GetOptions{})
	// if err != nil {
	// 	glog.Infof("Got error: %+v", err)
	// } else {
	// 	glog.Infof("Got instance: %+v", *i2)
	// }

	// Enqueue the

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
	vcli := kubeflowClientSet.NewForConfigOrDie(cfg).ViewerV1alpha1()

	cli, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		log.Fatal(err)
	}

	mgr, err := builder.SimpleController().
		ForType(&viewerV1alpha1.Viewer{}).
		WithConfig(cfg).
		Build(&ViewerReconciler{Client: cli, vcli: vcli, scheme: scheme.Scheme})

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
