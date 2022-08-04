package client

import (
	"context"
	"errors"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	corev1 "k8s.io/client-go/applyconfigurations/core/v1"
)

type FakeNamespaceClient struct {
	namespace string
	user      string
}

func (f *FakeNamespaceClient) SetReturnValues(namespace string, user string) {
	f.namespace = namespace
	f.user = user
}

func (f FakeNamespaceClient) Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.Namespace, error) {
	if f.namespace == name && len(f.user) != 0 {
		ns := v1.Namespace{ObjectMeta: metav1.ObjectMeta{
			Namespace: f.namespace,
			Annotations: map[string]string{
				"owner": f.user,
			},
		}}
		return &ns, nil
	}
	return nil, errors.New("failed to get namespace")
}

func (f FakeNamespaceClient) Create(ctx context.Context, namespace *v1.Namespace, opts metav1.CreateOptions) (*v1.Namespace, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (f FakeNamespaceClient) Update(ctx context.Context, namespace *v1.Namespace, opts metav1.UpdateOptions) (*v1.Namespace, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (f FakeNamespaceClient) UpdateStatus(ctx context.Context, namespace *v1.Namespace, opts metav1.UpdateOptions) (*v1.Namespace, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (f FakeNamespaceClient) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	glog.Error("This fake method is not yet implemented.")
	return nil
}

func (f FakeNamespaceClient) List(ctx context.Context, opts metav1.ListOptions) (*v1.NamespaceList, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (f FakeNamespaceClient) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (f FakeNamespaceClient) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.Namespace, err error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (f FakeNamespaceClient) Apply(ctx context.Context, namespace *corev1.NamespaceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Namespace, err error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (f FakeNamespaceClient) ApplyStatus(ctx context.Context, namespace *corev1.NamespaceApplyConfiguration, opts metav1.ApplyOptions) (result *v1.Namespace, err error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}

func (f FakeNamespaceClient) Finalize(ctx context.Context, item *v1.Namespace, opts metav1.UpdateOptions) (*v1.Namespace, error) {
	glog.Error("This fake method is not yet implemented.")
	return nil, nil
}
