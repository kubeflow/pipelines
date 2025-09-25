package client_manager

import (
	"crypto/tls"
	"fmt"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"

	"github.com/kubeflow/pipelines/backend/src/v2/cacheutils"
	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ClientManagerInterface interface {
	K8sClient() kubernetes.Interface
	MetadataClient() metadata.ClientInterface
	CacheClient() cacheutils.Client
}

// Ensure ClientManager implements ClientManagerInterface
var _ ClientManagerInterface = (*ClientManager)(nil)

// ClientManager is a container for various service clients.
type ClientManager struct {
	k8sClient      kubernetes.Interface
	metadataClient metadata.ClientInterface
	cacheClient    cacheutils.Client
}

type Options struct {
	MLMDServerAddress    string
	MLMDServerPort       string
	CacheDisabled        bool
	MLPipelineTLSEnabled bool
	CaCertPath           string
}

// NewClientManager creates and Init a new instance of ClientManager.
func NewClientManager(options *Options) (*ClientManager, error) {
	clientManager := &ClientManager{}
	err := clientManager.init(options)
	if err != nil {
		return nil, err
	}

	return clientManager, nil
}

func (cm *ClientManager) K8sClient() kubernetes.Interface {
	return cm.k8sClient
}

func (cm *ClientManager) MetadataClient() metadata.ClientInterface {
	return cm.metadataClient
}

func (cm *ClientManager) CacheClient() cacheutils.Client {
	return cm.cacheClient
}

func (cm *ClientManager) init(opts *Options) error {
	tlsCfg := util.GetTLSConfig(opts.CaCertPath)
	k8sClient, err := initK8sClient()
	if err != nil {
		return err
	}
	metadataClient, err := initMetadataClient(opts.MLMDServerAddress, opts.MLMDServerPort, opts.MLPipelineTLSEnabled, tlsCfg)
	if err != nil {
		return err
	}
	cacheClient, err := initCacheClient(opts.CacheDisabled, opts.MLPipelineTLSEnabled, tlsCfg)
	if err != nil {
		return err
	}
	cm.k8sClient = k8sClient
	cm.metadataClient = metadataClient
	cm.cacheClient = cacheClient
	return nil
}

func initK8sClient() (kubernetes.Interface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client: %w", err)
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
	}
	return k8sClient, nil
}

func initMetadataClient(address string, port string, mlPipelineTLSEnabled bool, tlsCfg *tls.Config) (metadata.ClientInterface, error) {
	glog.Info("Calling metadata.NewClient() from client_manager.initMetadataClient()")
	return metadata.NewClient(address, port, mlPipelineTLSEnabled, tlsCfg)
}

func initCacheClient(cacheDisabled bool, mlPipelineServiceTLSEnabled bool, tlsCfg *tls.Config) (cacheutils.Client, error) {
	return cacheutils.NewClient(cacheDisabled, mlPipelineServiceTLSEnabled, tlsCfg)
}
