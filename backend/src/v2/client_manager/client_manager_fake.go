package client_manager

import (
	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"k8s.io/client-go/kubernetes"
)

type FakeClientManager struct {
	k8sClient      kubernetes.Interface
	metadataClient metadata.ClientInterface
	cacheClient    cacheutils.Client
}

// Ensure FakeClientManager implements ClientManagerInterface
var _ ClientManagerInterface = (*FakeClientManager)(nil)

func (f *FakeClientManager) K8sClient() kubernetes.Interface {
	return f.k8sClient
}

func (f *FakeClientManager) MetadataClient() metadata.ClientInterface {
	return f.metadataClient
}

func (f *FakeClientManager) CacheClient() cacheutils.Client {
	return f.cacheClient
}

func NewFakeClientManager(k8sClient kubernetes.Interface, metadataClient metadata.ClientInterface, cacheClient cacheutils.Client) *FakeClientManager {
	return &FakeClientManager{
		k8sClient:      k8sClient,
		metadataClient: metadataClient,
		cacheClient:    cacheClient,
	}
}
