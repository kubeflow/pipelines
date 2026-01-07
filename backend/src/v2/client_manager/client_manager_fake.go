package client_manager

import (
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"k8s.io/client-go/kubernetes"
)

type FakeClientManager struct {
	k8sClient kubernetes.Interface
	kfpAPI    kfpapi.API
}

// Ensure FakeClientManager implements ClientManagerInterface
var _ ClientManagerInterface = (*FakeClientManager)(nil)

func (f *FakeClientManager) K8sClient() kubernetes.Interface {
	return f.k8sClient
}

func (f *FakeClientManager) KFPAPIClient() kfpapi.API {
	return f.kfpAPI
}

func NewFakeClientManager(k8sClient kubernetes.Interface, kfpAPI kfpapi.API) *FakeClientManager {
	return &FakeClientManager{
		k8sClient: k8sClient,
		kfpAPI:    kfpAPI,
	}
}
