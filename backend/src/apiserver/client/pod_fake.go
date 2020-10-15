package client

import (
	"errors"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
)

type FakePodClient struct {
}

func (FakePodClient) GetEphemeralContainers(string, v1.GetOptions) (*corev1.EphemeralContainers, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) UpdateEphemeralContainers(string, *corev1.EphemeralContainers) (*corev1.EphemeralContainers, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Create(*corev1.Pod) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Update(*corev1.Pod) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) UpdateStatus(*corev1.Pod) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Delete(name string, options *v1.DeleteOptions) error {
	return nil
}

func (FakePodClient) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (FakePodClient) Get(name string, options v1.GetOptions) (*corev1.Pod, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) List(opts v1.ListOptions) (*corev1.PodList, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Watch(opts v1.ListOptions) (watch.Interface, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *corev1.Pod, err error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakePodClient) Bind(binding *corev1.Binding) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (FakePodClient) Evict(eviction *v1beta1.Eviction) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (FakePodClient) GetLogs(name string, opts *corev1.PodLogOptions) *rest.Request {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

type FakeBadPodClient struct {
	FakePodClient
}

func (FakeBadPodClient) Delete(name string, options *v1.DeleteOptions) error {
	return errors.New("failed to delete pod")
}
