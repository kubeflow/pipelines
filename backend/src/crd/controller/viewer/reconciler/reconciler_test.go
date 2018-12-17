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

package reconciler

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	_ "github.com/google/go-cmp/cmp/cmpopts"
	viewerV1alpha1 "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/viewer/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var viewer *Reconciler

func TestMain(m *testing.M) {
	viewerV1alpha1.AddToScheme(scheme.Scheme)
	os.Exit(m.Run())
}

func getDeployments(t *testing.T, c client.Client) []*appsv1.Deployment {
	dplList := &appsv1.DeploymentList{}

	if err := c.List(context.Background(), &client.ListOptions{}, dplList); err != nil {
		t.Fatalf("Failed to list deployments from Fake client: %v", err)
	}

	var dpls []*appsv1.Deployment
	for _, dpl := range dplList.Items {
		dpl := dpl
		dpls = append(dpls, &dpl)
	}
	return dpls
}

func getServices(t *testing.T, c client.Client) []*corev1.Service {
	svcList := &corev1.ServiceList{}

	if err := c.List(context.Background(), &client.ListOptions{}, svcList); err != nil {
		t.Fatalf("Failed to list services from Fake client: %v", err)
	}

	var svcs []*corev1.Service
	for _, svc := range svcList.Items {
		svc := svc
		svcs = append(svcs, &svc)
	}
	return svcs
}

func deploymentNames(dpls []*appsv1.Deployment) []string {
	var ns []string

	for _, d := range dpls {
		ns = append(ns, d.ObjectMeta.Namespace+"/"+d.ObjectMeta.Name)
	}
	return ns
}

func serviceNames(svcs []*corev1.Service) []string {
	var ns []string

	for _, s := range svcs {
		ns = append(ns, s.ObjectMeta.Namespace+"/"+s.ObjectMeta.Name)
	}
	return ns
}

func boolRef(val bool) *bool {
	return &val
}

func TestReconcile_EachViewerCreatesADeployment(t *testing.T) {
	viewer := &viewerV1alpha1.Viewer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "viewer-123",
			Namespace: "kubeflow",
		},
		Spec: viewerV1alpha1.ViewerSpec{
			Type: viewerV1alpha1.ViewerTypeTensorboard,
			TensorboardSpec: viewerV1alpha1.TensorboardSpec{
				LogDir: "gs://tensorboard/logdir",
			},
		},
	}

	cli := fake.NewFakeClient(viewer)
	reconciler := New(cli, scheme.Scheme, &Options{MaxNumViewers: 10})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "viewer-123", Namespace: "kubeflow"},
	}
	_, err := reconciler.Reconcile(req)

	if err != nil {
		t.Fatalf("Reconcile(%+v) = %v; Want nil error", req, err)
	}

	wantDpls := []*appsv1.Deployment{{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "viewer-123-deployment",
			Namespace: "kubeflow",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "kubeflow.org/v1alpha1",
				Name:               "viewer-123",
				Kind:               "Viewer",
				Controller:         boolRef(true),
				BlockOwnerDeletion: boolRef(true),
			}}},
		Spec: appsv1.DeploymentSpec{
			Replicas: nil,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"deployment": "viewer-123-deployment",
				}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deployment": "viewer-123-deployment",
					}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "viewer-123-pod",
						Image: "tensorflow/tensorflow:1.11.0",
						Args: []string{
							"tensorboard",
							"--logdir=gs://tensorboard/logdir",
							"--path_prefix=/tensorboard/viewer-123/"},
						Ports: []corev1.ContainerPort{{ContainerPort: 6006}},
					}}}}}}}

	gotDpls := getDeployments(t, cli)

	if !cmp.Equal(gotDpls, wantDpls) {
		t.Errorf("Created viewer CRD %+v\nWant deployment: %+v\nGot deployment: %+v\nDiff: %s",
			viewer, gotDpls, wantDpls, cmp.Diff(wantDpls, gotDpls))

	}
}

func TestReconcile_EachViewerCreatesAService(t *testing.T) {
	viewer := &viewerV1alpha1.Viewer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "viewer-123",
			Namespace: "kubeflow",
		},
		Spec: viewerV1alpha1.ViewerSpec{
			Type: viewerV1alpha1.ViewerTypeTensorboard,
			TensorboardSpec: viewerV1alpha1.TensorboardSpec{
				LogDir: "gs://tensorboard/logdir",
			},
		},
	}

	cli := fake.NewFakeClient(viewer)
	reconciler := New(cli, scheme.Scheme, &Options{MaxNumViewers: 10})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "viewer-123", Namespace: "kubeflow"},
	}
	_, err := reconciler.Reconcile(req)

	if err != nil {
		t.Fatalf("Reconcile(%+v) = %v; Want nil error", req, err)
	}

	want := []*v1.Service{{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "viewer-123-service",
			Namespace: "kubeflow",
			Annotations: map[string]string{
				"getambassador.io/config": "\n---\n" +
					"apiVersion: ambassador/v0\n" +
					"kind: Mapping\n" +
					"name: viewer-mapping-viewer-123\n" +
					"prefix: /tensorboard/viewer-123/\n" +
					"rewrite: /tensorboard/viewer-123/\n" +
					"service: viewer-123-service"},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "kubeflow.org/v1alpha1",
				Kind:               "Viewer",
				Name:               "viewer-123",
				Controller:         boolRef(true),
				BlockOwnerDeletion: boolRef(true)}},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{corev1.ServicePort{
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(80),
				TargetPort: intstr.IntOrString{IntVal: viewerTargetPort},
			}},
			Selector: map[string]string{
				"deployment": "viewer-123-deployment",
			},
		}}}

	got := getServices(t, cli)

	if !cmp.Equal(got, want) {
		t.Errorf("Created viewer CRD %+v\nWant services: %+v\nGot services: %+v\nDiff: %s",
			viewer, got, want, cmp.Diff(want, got))

	}
}

func TestReconcile_UnknownViewerTypesAreIgnored(t *testing.T) {
	viewer := &viewerV1alpha1.Viewer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "viewer-123",
			Namespace: "kubeflow",
		},
		Spec: viewerV1alpha1.ViewerSpec{
			Type: "unknownType",
			TensorboardSpec: viewerV1alpha1.TensorboardSpec{
				LogDir: "gs://tensorboard/logdir",
			},
		},
	}

	cli := fake.NewFakeClient(viewer)
	reconciler := New(cli, scheme.Scheme, &Options{MaxNumViewers: 10})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "viewer-123", Namespace: "kubeflow"},
	}

	got, err := reconciler.Reconcile(req)

	// Want no error and no requeuing.
	want := reconcile.Result{Requeue: false}
	if err != nil || !cmp.Equal(got, want) {
		t.Errorf("Reconcile(%+v) =\nGot %+v, %v\nWant %+v, <nil>\nDiff: %s",
			req, got, err, want, cmp.Diff(want, got))
	}

	dpls := getDeployments(t, cli)
	if len(dpls) > 0 {
		t.Errorf("Reconcile(%+v)\nGot deployments: %+v\nWant none.", req, dpls)
	}

	svcs := getServices(t, cli)
	if len(svcs) > 0 {
		t.Errorf("Reconcile(%+v)\nGot services: %+v\nWant none.", req, svcs)
	}
}

func TestReconcile_UnknownViewerDoesNothing(t *testing.T) {
	// Client with empty store.
	cli := fake.NewFakeClient()

	reconciler := New(cli, scheme.Scheme, &Options{MaxNumViewers: 10})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "viewer-123", Namespace: "kubeflow"},
	}
	got, err := reconciler.Reconcile(req)

	want := reconcile.Result{}
	if err != nil || !cmp.Equal(got, want) {
		t.Errorf("Reconcile(%+v) =\nGot %+v, %v\nWant %+v, <nil>\nDiff: %s",
			req, got, err, want, cmp.Diff(want, got))
	}

	dpls := getDeployments(t, cli)
	if len(dpls) > 0 {
		t.Errorf("Reconcile(%+v)\nGot deployments: %+v\nWant none.", req, dpls)
	}

	svcs := getServices(t, cli)
	if len(svcs) > 0 {
		t.Errorf("Reconcile(%+v)\nGot services: %+v\nWant none.", req, svcs)
	}
}

func makeViewer(id int) (*types.NamespacedName, *viewerV1alpha1.Viewer) {
	v := &viewerV1alpha1.Viewer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("viewer-%d", id),
			Namespace: "kubeflow",
		},
		Spec: viewerV1alpha1.ViewerSpec{
			Type: viewerV1alpha1.ViewerTypeTensorboard,
			TensorboardSpec: viewerV1alpha1.TensorboardSpec{
				LogDir: "gs://tensorboard/logdir",
			},
		},
	}
	n := &types.NamespacedName{
		Name:      fmt.Sprintf("viewer-%d", id),
		Namespace: "kubeflow",
	}

	return n, v
}

func TestReconcile_MaxNumViewersIsEnforced(t *testing.T) {
	cli := fake.NewFakeClient()
	reconciler := New(cli, scheme.Scheme, &Options{MaxNumViewers: 5})

	ctx := context.Background()
	for i := 0; i < 5; i++ {
		n, v := makeViewer(i)
		cli.Create(ctx, v)
		req := reconcile.Request{NamespacedName: *n}
		_, err := reconciler.Reconcile(req)

		if err != nil {
			t.Errorf("Reconcile(%+v) = %v; Want nil error", req, err)
		}
	}

	wantDpls := []string{
		"kubeflow/viewer-0-deployment",
		"kubeflow/viewer-1-deployment",
		"kubeflow/viewer-2-deployment",
		"kubeflow/viewer-3-deployment",
		"kubeflow/viewer-4-deployment",
	}

	gotDpls := deploymentNames(getDeployments(t, cli))
	if !cmp.Equal(wantDpls, gotDpls) {
		t.Errorf("Got deployments %v\n. Want %v", gotDpls, wantDpls)
	}

	gotSvcs := []string{
		"kubeflow/viewer-0-service",
		"kubeflow/viewer-1-service",
		"kubeflow/viewer-2-service",
		"kubeflow/viewer-3-service",
		"kubeflow/viewer-4-service",
	}

	wantSvcs := serviceNames(getServices(t, cli))
	if !cmp.Equal(wantSvcs, gotSvcs) {
		t.Errorf("Got services %v\n. Want %v", gotSvcs, wantSvcs)
	}

	// Now add another viewer. The oldest created service should be deleted, and the new one should

}
