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

// Package reconciler describes a Reconciler for working with Viewer CRDs. The
// reconciler takes care of the main logic for ensuring every Viewer CRD
// corresponds to a unique deployment and backing service. The service is
// annotated such that it is compatible with Ambassador managed routing.
// Currently, only supports Tensorboard viewers. Adding a new viewer CRD for
// tensorboard with the name 'abc123' will result in the tensorboard instance
// serving under the path '/tensorboard/abc123'.
package reconciler

import (
	"context"
	"fmt"
	"strings"

	"github.com/golang/glog"
	viewerV1beta1 "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/viewer/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

const viewerTargetPort = 6006

const defaultTensorflowImage = "tensorflow/tensorflow:1.13.2"

// Reconciler implements reconcile.Reconciler for the Viewer CRD.
type Reconciler struct {
	client.Client
	scheme *runtime.Scheme
	opts   *Options
}

// Options are the set of options to configure the behaviour of Reconciler.
type Options struct {
	// MaxNumViewers sets an upper bound on the number of viewer instances allowed.
	// When a user attempts to create one more viewer than this number, the oldest
	// existing viewer will be deleted.
	MaxNumViewers int
}

// New returns a new Reconciler.
func New(cli client.Client, scheme *runtime.Scheme, opts *Options) (*Reconciler, error) {
	if opts.MaxNumViewers < 1 {
		return nil, fmt.Errorf("MaxNumViewers should at least be 1. Got %d", opts.MaxNumViewers)
	}
	return &Reconciler{Client: cli, scheme: scheme, opts: opts}, nil
}

// Reconcile runs the main logic for reconciling the state of a viewer with a
// corresponding deployment and service allowing users to access the view under
// a specific path.
func (r *Reconciler) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	glog.Infof("Reconcile request: %+v", req)

	view := &viewerV1beta1.Viewer{}
	if err := r.Get(context.Background(), req.NamespacedName, view); err != nil {
		if errors.IsNotFound(err) {
			// No viewer instance, so this may be the result of a delete.
			// Nothing to do.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}
	glog.Infof("Got instance: %+v", view)

	// Ignore other viewer types for now.
	if view.Spec.Type != viewerV1beta1.ViewerTypeTensorboard {
		glog.Infof("Unsupported spec type: %q", view.Spec.Type)
		// Return nil to indicate nothing more to do here.
		return reconcile.Result{}, nil
	}

	if len(view.Spec.TensorboardSpec.TensorflowImage) == 0 {
		view.Spec.TensorboardSpec.TensorflowImage = defaultTensorflowImage
	}

	// Check and maybe delete the oldest viewer before creating the next one.
	if err := r.maybeDeleteOldestViewer(view.Spec.Type, view.Namespace); err != nil {
		// Couldn't delete. Requeue.
		return reconcile.Result{Requeue: true}, err
	}

	// Set up potential deployment.
	dpl, err := deploymentFrom(view)
	if err != nil {
		utilruntime.HandleError(err)
		// User error, don't requeue key.
		return reconcile.Result{}, nil
	}

	// Set the deployment to be owned by the view instance. This ensures that
	// deleting the viewer instance will delete the deployment as well.
	if err := controllerutil.SetControllerReference(view, dpl, r.scheme); err != nil {
		// Error means that the deployment is already owned by some other instance.
		utilruntime.HandleError(err)
		return reconcile.Result{}, err
	}

	foundDpl := &appsv1.Deployment{}
	nsn := types.NamespacedName{Name: dpl.Name, Namespace: dpl.Namespace}
	if err := r.Client.Get(context.Background(), nsn, foundDpl); err != nil {
		if errors.IsNotFound(err) {
			// Create a new instance.
			if createErr := r.Client.Create(context.Background(), dpl); createErr != nil {
				utilruntime.HandleError(fmt.Errorf("error creating deployment: %v", createErr))
				return reconcile.Result{}, createErr
			}
		} else {
			// Some other error.
			utilruntime.HandleError(err)
			return reconcile.Result{}, err
		}
	}
	glog.Infof("Created new deployment with spec: %+v", dpl)

	// Set up a service for the deployment above.
	svc := serviceFrom(view, dpl.Name)
	// Set the service to be owned by the view instance as well.
	if err := controllerutil.SetControllerReference(view, svc, r.scheme); err != nil {
		// Error means that the service is already owned by some other instance.
		utilruntime.HandleError(err)
		return reconcile.Result{}, err
	}

	foundSvc := &corev1.Service{}
	nsn = types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}
	if err := r.Client.Get(context.Background(), nsn, foundSvc); err != nil {
		if errors.IsNotFound(err) {
			// Create a new instance.
			if createErr := r.Client.Create(context.Background(), svc); createErr != nil {
				utilruntime.HandleError(fmt.Errorf("error creating service: %v", createErr))
				return reconcile.Result{}, createErr
			}
		} else {
			// Some other error.
			utilruntime.HandleError(err)
			return reconcile.Result{}, err
		}
	}
	glog.Infof("Created new service with spec: %+v", svc)

	return reconcile.Result{}, nil
}

func setPodSpecForTensorboard(view *viewerV1beta1.Viewer, s *corev1.PodSpec) {
	if len(s.Containers) == 0 {
		s.Containers = append(s.Containers, corev1.Container{})
	}

	c := &s.Containers[0]
	c.Name = view.Name + "-pod"
	c.Image = view.Spec.TensorboardSpec.TensorflowImage
	c.Args = []string{
		"tensorboard",
		fmt.Sprintf("--logdir=%s", view.Spec.TensorboardSpec.LogDir),
		fmt.Sprintf("--path_prefix=/tensorboard/%s/", view.Name),
	}

	tfImageVersion := strings.Split(view.Spec.TensorboardSpec.TensorflowImage, ":")[1]
	// This is needed for tf 2.0
	if !strings.HasPrefix(tfImageVersion, `1.`) {
		c.Args = append(c.Args, "--bind_all")
	}

	c.Ports = []corev1.ContainerPort{
		corev1.ContainerPort{ContainerPort: viewerTargetPort},
	}

}

func deploymentFrom(view *viewerV1beta1.Viewer) (*appsv1.Deployment, error) {
	name := view.Name + "-deployment"
	dpl := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: view.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"deployment": name,
					"app":        "viewer",
					"viewer":     view.Name,
				},
			},
			Template: view.Spec.PodTemplateSpec,
		},
	}

	// Add label so we can select this deployment with a service.
	if dpl.Spec.Template.ObjectMeta.Labels == nil {
		dpl.Spec.Template.ObjectMeta.Labels = make(map[string]string)
	}
	dpl.Spec.Template.ObjectMeta.Labels["deployment"] = name
	dpl.Spec.Template.ObjectMeta.Labels["app"] = "viewer"
	dpl.Spec.Template.ObjectMeta.Labels["viewer"] = view.Name

	switch view.Spec.Type {
	case viewerV1beta1.ViewerTypeTensorboard:
		setPodSpecForTensorboard(view, &dpl.Spec.Template.Spec)
	default:
		return nil, fmt.Errorf("unknown viewer type: %q", view.Spec.Type)
	}

	return dpl, nil
}

const mappingTpl = `
---
apiVersion: ambassador/v0
kind: Mapping
name: viewer-mapping-%s
prefix: %s
rewrite: %s
service: %s`

func serviceFrom(v *viewerV1beta1.Viewer, deploymentName string) *corev1.Service {
	name := v.Name + "-service"
	path := fmt.Sprintf("/%s/%s/", v.Spec.Type, v.Name)
	mapping := fmt.Sprintf(mappingTpl, v.Name, path, path, name)

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   v.Namespace,
			Annotations: map[string]string{"getambassador.io/config": mapping},
			Labels: map[string]string{
				"app":    "viewer",
				"viewer": v.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"deployment": deploymentName,
				"app":        "viewer",
				"viewer":     v.Name,
			},
			Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name:       "http",
					Protocol:   corev1.ProtocolTCP,
					Port:       80,
					TargetPort: intstr.IntOrString{IntVal: viewerTargetPort}},
			},
		},
	}
}

func (r *Reconciler) maybeDeleteOldestViewer(t viewerV1beta1.ViewerType, namespace string) error {
	list := &viewerV1beta1.ViewerList{}

	if err := r.Client.List(context.Background(), list, &client.ListOptions{Namespace: namespace}); err != nil {
		return fmt.Errorf("failed to list viewers: %v", err)
	}

	if len(list.Items) <= r.opts.MaxNumViewers {
		return nil
	}

	// Delete the oldest viewer by creation timestamp.
	oldest := &list.Items[0] // MaxNumViewers must be at least one.
	for i := range list.Items {
		if list.Items[i].CreationTimestamp.Time.Before(oldest.CreationTimestamp.Time) {
			oldest = &list.Items[i]
		}
	}

	return r.Client.Delete(context.Background(), oldest)
}
