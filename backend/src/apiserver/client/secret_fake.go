package client

import (
	"context"
	"errors"
	"k8s.io/apimachinery/pkg/watch"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

type FakeSecretClient struct{}

func (FakeSecretClient) Create(context.Context, *v1.Secret, metav1.CreateOptions) (*v1.Secret, error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}
func (FakeSecretClient) Update(context.Context, *v1.Secret, metav1.UpdateOptions) (*v1.Secret, error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}
func (FakeSecretClient) Delete(context.Context, string, metav1.DeleteOptions) error {
	glog.Error("This fake method is not yet implemented")
	return nil
}
func (FakeSecretClient) DeleteCollection(context.Context, metav1.DeleteOptions, metav1.ListOptions) error {
	return nil
}
func (FakeSecretClient) Get(context.Context, string, metav1.GetOptions) (*v1.Secret, error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}
func (FakeSecretClient) List(context.Context, metav1.ListOptions) (*v1.SecretList, error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}
func (FakeSecretClient) Watch(context.Context, metav1.ListOptions) (watch.Interface, error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}
func (FakeSecretClient) Patch(context.Context, string, types.PatchType, []byte, metav1.PatchOptions, ...string) (result *v1.Secret, err error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}
func (FakeSecretClient) Apply(context.Context, *corev1.SecretApplyConfiguration, metav1.ApplyOptions) (result *v1.Secret, err error) {
	glog.Error("This fake method is not yet implemented")
	return nil, nil
}

type FakeBadSecretClient struct {
	FakeSecretClient
}

func (FakeBadSecretClient) Delete(context.Context, string, metav1.DeleteOptions) error {
	return errors.New("failed to delete pod")
}
