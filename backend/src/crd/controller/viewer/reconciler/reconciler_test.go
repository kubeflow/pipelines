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
	"time"

	"github.com/google/go-cmp/cmp"
	_ "github.com/google/go-cmp/cmp/cmpopts"
	viewerV1beta1 "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/viewer/v1beta1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var viewer *Reconciler

const tensorflowImage = "potentially_custom_tensorflow:dummy"

func TestMain(m *testing.M) {
	viewerV1beta1.AddToScheme(scheme.Scheme)
	os.Exit(m.Run())
}

func getDeployments(t *testing.T, c client.Client) []*appsv1.Deployment {
	dplList := &appsv1.DeploymentList{}

	if err := c.List(context.Background(), dplList, &client.ListOptions{}); err != nil {
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

	if err := c.List(context.Background(), svcList, &client.ListOptions{}); err != nil {
		t.Fatalf("Failed to list services with fake client: %v", err)
	}

	var svcs []*corev1.Service
	for _, svc := range svcList.Items {
		svc := svc
		svcs = append(svcs, &svc)
	}
	return svcs
}

func getViewers(t *testing.T, c client.Client) []*viewerV1beta1.Viewer {
	list := &viewerV1beta1.ViewerList{}

	if err := c.List(context.Background(), list, &client.ListOptions{}); err != nil {
		t.Fatalf("Failed to list viewers with fake client: %v", err)
	}

	var items []*viewerV1beta1.Viewer
	for _, i := range list.Items {
		i := i
		items = append(items, &i)
	}
	return items
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

func viewerNames(items []*viewerV1beta1.Viewer) []string {
	var ns []string

	for _, s := range items {
		ns = append(ns, s.ObjectMeta.Namespace+"/"+s.ObjectMeta.Name)
	}
	return ns
}

func boolRef(val bool) *bool {
	return &val
}

func TestReconcile_EachViewerCreatesADeployment(t *testing.T) {
	viewer := &viewerV1beta1.Viewer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "viewer-123",
			Namespace: "kubeflow",
		},
		Spec: viewerV1beta1.ViewerSpec{
			Type: viewerV1beta1.ViewerTypeTensorboard,
			TensorboardSpec: viewerV1beta1.TensorboardSpec{
				LogDir:          "gs://tensorboard/logdir",
				TensorflowImage: tensorflowImage,
			},
		},
	}

	cli := fake.NewFakeClient(viewer)
	reconciler, _ := New(cli, scheme.Scheme, &Options{MaxNumViewers: 10})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "viewer-123", Namespace: "kubeflow"},
	}
	_, err := reconciler.Reconcile(req)

	if err != nil {
		t.Fatalf("Reconcile(%+v) = %v; Want nil error", req, err)
	}

	wantDpls := []*appsv1.Deployment{{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "viewer-123-deployment",
			Namespace:       "kubeflow",
			ResourceVersion: "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "kubeflow.org/v1beta1",
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
					"app":        "viewer",
					"viewer":     "viewer-123",
				}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deployment": "viewer-123-deployment",
						"app":        "viewer",
						"viewer":     "viewer-123",
					}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "viewer-123-pod",
						Image: tensorflowImage,
						Args: []string{
							"tensorboard",
							"--logdir=gs://tensorboard/logdir",
							"--path_prefix=/tensorboard/viewer-123/",
							"--bind_all"},
						Ports: []corev1.ContainerPort{{ContainerPort: 6006}},
					}}}}}}}

	gotDpls := getDeployments(t, cli)

	if !cmp.Equal(gotDpls, wantDpls) {
		t.Errorf("Created viewer CRD %+v\nWant deployment: %+v\nGot deployment: %+v\nDiff: %s",
			viewer, gotDpls, wantDpls, cmp.Diff(wantDpls, gotDpls))

	}
}

func TestReconcile_ViewerUsesSpecifiedVolumeMountsForDeployment(t *testing.T) {
	viewer := &viewerV1beta1.Viewer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "viewer-123",
			Namespace: "kubeflow",
		},
		Spec: viewerV1beta1.ViewerSpec{
			Type: viewerV1beta1.ViewerTypeTensorboard,
			TensorboardSpec: viewerV1beta1.TensorboardSpec{
				LogDir:          "gs://tensorboard/logdir",
				TensorflowImage: tensorflowImage,
			},
			PodTemplateSpec: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						corev1.Container{
							VolumeMounts: []corev1.VolumeMount{
								corev1.VolumeMount{
									Name:      "/volume-mount-name",
									MountPath: "/mount/path",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						corev1.Volume{
							Name: "/volume-mount-name",
							VolumeSource: corev1.VolumeSource{
								GCEPersistentDisk: &corev1.GCEPersistentDiskVolumeSource{
									PDName: "my-persistent-volume",
									FSType: "ext4",
								},
							},
						},
					},
				},
			},
		},
	}

	cli := fake.NewFakeClient(viewer)
	reconciler, _ := New(cli, scheme.Scheme, &Options{MaxNumViewers: 10})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "viewer-123", Namespace: "kubeflow"},
	}
	_, err := reconciler.Reconcile(req)

	if err != nil {
		t.Fatalf("Reconcile(%+v) = %v; Want nil error", req, err)
	}

	wantDpls := []*appsv1.Deployment{{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "viewer-123-deployment",
			Namespace:       "kubeflow",
			ResourceVersion: "1",
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "kubeflow.org/v1beta1",
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
					"app":        "viewer",
					"viewer":     "viewer-123",
				}},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"deployment": "viewer-123-deployment",
						"app":        "viewer",
						"viewer":     "viewer-123",
					}},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "viewer-123-pod",
						Image: tensorflowImage,
						Args: []string{
							"tensorboard",
							"--logdir=gs://tensorboard/logdir",
							"--path_prefix=/tensorboard/viewer-123/",
							"--bind_all"},
						Ports: []corev1.ContainerPort{{ContainerPort: 6006}},
						VolumeMounts: []v1.VolumeMount{
							{Name: "/volume-mount-name", MountPath: "/mount/path"},
						},
					}},
					Volumes: []v1.Volume{{
						Name: "/volume-mount-name",
						VolumeSource: v1.VolumeSource{
							GCEPersistentDisk: &corev1.GCEPersistentDiskVolumeSource{
								PDName: "my-persistent-volume",
								FSType: "ext4",
							},
						},
					}},
				}}},
	}}

	gotDpls := getDeployments(t, cli)

	if !cmp.Equal(gotDpls, wantDpls) {
		t.Errorf("Created viewer CRD %+v\nWant deployment: %+v\nGot deployment: %+v\nDiff: %s",
			viewer, gotDpls, wantDpls, cmp.Diff(wantDpls, gotDpls))

	}
}

func TestReconcile_EachViewerCreatesAService(t *testing.T) {
	viewer := &viewerV1beta1.Viewer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "viewer-123",
			Namespace: "kubeflow",
		},
		Spec: viewerV1beta1.ViewerSpec{
			Type: viewerV1beta1.ViewerTypeTensorboard,
			TensorboardSpec: viewerV1beta1.TensorboardSpec{
				LogDir:          "gs://tensorboard/logdir",
				TensorflowImage: tensorflowImage,
			},
		},
	}

	cli := fake.NewFakeClient(viewer)
	reconciler, _ := New(cli, scheme.Scheme, &Options{MaxNumViewers: 10})

	req := reconcile.Request{
		NamespacedName: types.NamespacedName{Name: "viewer-123", Namespace: "kubeflow"},
	}
	_, err := reconciler.Reconcile(req)

	if err != nil {
		t.Fatalf("Reconcile(%+v) = %v; Want nil error", req, err)
	}

	want := []*v1.Service{{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "viewer-123-service",
			Namespace:       "kubeflow",
			ResourceVersion: "1",
			Annotations: map[string]string{
				"getambassador.io/config": "\n---\n" +
					"apiVersion: ambassador/v0\n" +
					"kind: Mapping\n" +
					"name: viewer-mapping-viewer-123\n" +
					"prefix: /tensorboard/viewer-123/\n" +
					"rewrite: /tensorboard/viewer-123/\n" +
					"service: viewer-123-service"},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "kubeflow.org/v1beta1",
				Kind:               "Viewer",
				Name:               "viewer-123",
				Controller:         boolRef(true),
				BlockOwnerDeletion: boolRef(true)}},
			Labels: map[string]string{
				"app":    "viewer",
				"viewer": "viewer-123",
			}},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{corev1.ServicePort{
				Name:       "http",
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(80),
				TargetPort: intstr.IntOrString{IntVal: viewerTargetPort},
			}},
			Selector: map[string]string{
				"deployment": "viewer-123-deployment",
				"app":        "viewer",
				"viewer":     "viewer-123",
			},
		}}}

	got := getServices(t, cli)

	if !cmp.Equal(got, want) {
		t.Errorf("Created viewer CRD %+v\nWant services: %+v\nGot services: %+v\nDiff: %s",
			viewer, got, want, cmp.Diff(want, got))

	}
}

func TestReconcile_UnknownViewerTypesAreIgnored(t *testing.T) {
	viewer := &viewerV1beta1.Viewer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "viewer-123",
			Namespace: "kubeflow",
		},
		Spec: viewerV1beta1.ViewerSpec{
			Type: "unknownType",
			TensorboardSpec: viewerV1beta1.TensorboardSpec{
				LogDir:          "gs://tensorboard/logdir",
				TensorflowImage: tensorflowImage,
			},
		},
	}

	cli := fake.NewFakeClient(viewer)
	reconciler, _ := New(cli, scheme.Scheme, &Options{MaxNumViewers: 10})

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

	reconciler, _ := New(cli, scheme.Scheme, &Options{MaxNumViewers: 10})

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

func makeViewer(id int) (*types.NamespacedName, *viewerV1beta1.Viewer) {
	v := &viewerV1beta1.Viewer{
		ObjectMeta: metav1.ObjectMeta{
			Name:              fmt.Sprintf("viewer-%d", id),
			Namespace:         "kubeflow",
			CreationTimestamp: metav1.Time{Time: time.Unix(int64(id), 0)},
		},
		Spec: viewerV1beta1.ViewerSpec{
			Type: viewerV1beta1.ViewerTypeTensorboard,
			TensorboardSpec: viewerV1beta1.TensorboardSpec{
				LogDir:          "gs://tensorboard/logdir",
				TensorflowImage: tensorflowImage,
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
	reconciler, _ := New(cli, scheme.Scheme, &Options{MaxNumViewers: 5})

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

	// Check viewers.
	wantViewers := []string{
		"kubeflow/viewer-0",
		"kubeflow/viewer-1",
		"kubeflow/viewer-2",
		"kubeflow/viewer-3",
		"kubeflow/viewer-4",
	}
	gotViewers := viewerNames(getViewers(t, cli))
	if !cmp.Equal(wantViewers, gotViewers) {
		t.Errorf("Got viewers %v\n. Want %v", gotViewers, wantViewers)
	}

	// Check deployments.
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

	// Check services.
	wantSvcs := []string{
		"kubeflow/viewer-0-service",
		"kubeflow/viewer-1-service",
		"kubeflow/viewer-2-service",
		"kubeflow/viewer-3-service",
		"kubeflow/viewer-4-service",
	}

	gotSvcs := serviceNames(getServices(t, cli))
	if !cmp.Equal(wantSvcs, gotSvcs) {
		t.Errorf("Got services %v\n. Want %v", gotSvcs, wantSvcs)
	}

	// Now add a 6th viewer. The oldest created service should be deleted, and
	// the new one should launch a corresponding deployment and service.

	n, v := makeViewer(5)
	cli.Create(ctx, v)

	req := reconcile.Request{NamespacedName: *n}
	_, err := reconciler.Reconcile(req)

	if err != nil {
		t.Errorf("Reconcile(%+v) = %v; Want nil error", req, err)
	}

	// Check viewers. Viewer 0, which is the oldest, should be deleted, and we
	// should see the newly created viewer 5.
	wantViewers = []string{
		"kubeflow/viewer-1",
		"kubeflow/viewer-2",
		"kubeflow/viewer-3",
		"kubeflow/viewer-4",
		"kubeflow/viewer-5",
	}
	gotViewers = viewerNames(getViewers(t, cli))
	if !cmp.Equal(wantViewers, gotViewers) {
		t.Errorf("Got viewers %v\n. Want %v", gotViewers, wantViewers)
	}

	// Check deployments.
	wantDpls = []string{
		// The fake client does not propagate deletion requests based on owner
		// references, so the original deployment will still be present. In
		// production, this is not the case as Kubernetes will ensure that the child
		// resources are deleted as well.
		"kubeflow/viewer-0-deployment",
		"kubeflow/viewer-1-deployment",
		"kubeflow/viewer-2-deployment",
		"kubeflow/viewer-3-deployment",
		"kubeflow/viewer-4-deployment",
		"kubeflow/viewer-5-deployment",
	}

	gotDpls = deploymentNames(getDeployments(t, cli))
	if !cmp.Equal(wantDpls, gotDpls) {
		t.Errorf("Got deployments %v\n. Want %v", gotDpls, wantDpls)
	}

	// Check services.
	wantSvcs = []string{
		// The fake client does not propagate deletion requests based on owner
		// references, so the original service will still be present. In
		// production, this is not the case as Kubernetes will ensure that the child
		// resources are deleted as well.
		"kubeflow/viewer-0-service",
		"kubeflow/viewer-1-service",
		"kubeflow/viewer-2-service",
		"kubeflow/viewer-3-service",
		"kubeflow/viewer-4-service",
		"kubeflow/viewer-5-service",
	}

	gotSvcs = serviceNames(getServices(t, cli))
	if !cmp.Equal(wantSvcs, gotSvcs) {
		t.Errorf("Got services %v\n. Want %v", gotSvcs, wantSvcs)
	}
}
