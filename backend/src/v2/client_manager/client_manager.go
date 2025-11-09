package client_manager

import (
	"crypto/tls"
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient"
	"github.com/kubeflow/pipelines/backend/src/v2/apiclient/kfpapi"
	"k8s.io/client-go/kubernetes"
)

type ClientManagerInterface interface {
	K8sClient() kubernetes.Interface
	KFPAPIClient() kfpapi.API
}

// Ensure ClientManager implements ClientManagerInterface
var _ ClientManagerInterface = (*ClientManager)(nil)

// ClientManager is a container for various service clients.
type ClientManager struct {
	k8sClient    kubernetes.Interface
	kfpAPIClient kfpapi.API
}

type Options struct {
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

func (cm *ClientManager) KFPAPIClient() kfpapi.API {
	return cm.kfpAPIClient
}

func (cm *ClientManager) init(opts *Options) error {
	var tlsCfg *tls.Config
	var err error
	if opts.MLPipelineTLSEnabled {
		tlsCfg, err = util.GetTLSConfig(opts.CaCertPath)
		if err != nil {
			return err
		}
	}
	k8sClient, err := initK8sClient()
	if err != nil {
		return err
	}
	cm.k8sClient = k8sClient

	// Initialize connection to new KFP v2beta1 API server
	apiCfg := apiclient.FromEnv()
	kfpAPIClient, apiErr := apiclient.New(apiCfg, tlsCfg)
	if apiErr != nil {
		return fmt.Errorf("failed to init KFP API client: %w", apiErr)
	}
	var kfpAPI = kfpapi.New(kfpAPIClient)
	cm.kfpAPIClient = kfpAPI
	return nil
}

func initK8sClient() (kubernetes.Interface, error) {
	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return nil, err
	}
	k8sClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize kubernetes client set: %w", err)
	}
	return k8sClient, nil
}
