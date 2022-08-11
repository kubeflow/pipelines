package client

import (
	"context"
	"errors"

	"github.com/golang/glog"
	applyv1 "k8s.io/client-go/applyconfigurations/core/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type FakePodClient struct {
}

func (FakePodClient) GetEphemeralContainers(context.Context, string, v1.GetOptions) (*corev1.EphemeralContainers, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) UpdateEphemeralContainers(context.Context, string, *corev1.EphemeralContainers, v1.UpdateOptions) (*corev1.EphemeralContainers, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Create(context.Context, *corev1.Pod, v1.CreateOptions) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Apply(ctx context.Context, pod *applyv1.PodApplyConfiguration, opts v1.ApplyOptions) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) ApplyStatus(ctx context.Context, pod *applyv1.PodApplyConfiguration, opts v1.ApplyOptions) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Update(context.Context, *corev1.Pod, v1.UpdateOptions) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) UpdateStatus(context.Context, *corev1.Pod, v1.UpdateOptions) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Delete(ctx context.Context, name string, options v1.DeleteOptions) error {
	return nil
}

func (FakePodClient) DeleteCollection(ctx context.Context, options v1.DeleteOptions, listOptions v1.ListOptions) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (FakePodClient) Get(ctx context.Context, name string, options v1.GetOptions) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) List(ctx context.Context, opts v1.ListOptions) (*corev1.PodList, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Watch(ctx context.Context, opts v1.ListOptions) (watch.Interface, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts v1.PatchOptions, subresources ...string) (result *corev1.Pod, err error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Bind(ctx context.Context, binding *corev1.Binding, opts v1.CreateOptions) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (FakePodClient) Evict(ctx context.Context, eviction *v1beta1.Eviction) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (FakePodClient) GetLogs(name string, opts *corev1.PodLogOptions) *rest.Request {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (FakePodClient) ProxyGet(scheme, name, port, path string, params map[string]string) rest.ResponseWrapper {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

type FakeBadPodClient struct {
	FakePodClient
}

func (FakeBadPodClient) Delete(ctx context.Context, name string, options v1.DeleteOptions) error {
	return errors.New("failed to delete pod")
}
