package resource

import (
	"errors"
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

type FakeSecretClient struct {
}

func (FakeSecretClient) Create(*corev1.Secret) (*corev1.Secret, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakeSecretClient) Update(*corev1.Secret) (*corev1.Secret, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakeSecretClient) UpdateStatus(*corev1.Secret) (*corev1.Secret, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakeSecretClient) Delete(name string, options *v1.DeleteOptions) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (FakeSecretClient) DeleteCollection(options *v1.DeleteOptions, listOptions v1.ListOptions) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (FakeSecretClient) Get(name string, options v1.GetOptions) (*corev1.Secret, error) {
	return &corev1.Secret{}, nil
}

func (FakeSecretClient) List(opts v1.ListOptions) (*corev1.SecretList, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakeSecretClient) Watch(opts v1.ListOptions) (watch.Interface, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (FakeSecretClient) Patch(name string, pt types.PatchType, data []byte, subresources ...string) (result *corev1.Secret, err error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

type FakeSecretNotFoundClient struct {
	FakeSecretClient
}

func (FakeSecretNotFoundClient) Get(name string, options v1.GetOptions) (*corev1.Secret, error) {
	return nil, errors.New("Secret not found")
}
