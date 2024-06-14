package client

import (
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type KubernetesCoreInterface interface {
	PodClient(namespace string) v1.PodInterface
	SecretClient(namespace string) v1.SecretInterface
}

type KubernetesCore struct {
	coreV1Client v1.CoreV1Interface
}

func (c *KubernetesCore) PodClient(namespace string) v1.PodInterface {
	return c.coreV1Client.Pods(namespace)
}

func (c *KubernetesCore) SecretClient(namespace string) v1.SecretInterface {
	return c.coreV1Client.Secrets(namespace)
}

func createKubernetesCore(clientParams util.ClientParameters) (KubernetesCoreInterface, error) {
	clientSet, err := getKubernetesClientset(clientParams)
	if err != nil {
		return nil, err
	}
	return &KubernetesCore{clientSet.CoreV1()}, nil
}

// CreateKubernetesCoreOrFatal creates a new client for the Kubernetes pod.
func CreateKubernetesCoreOrFatal(initConnectionTimeout time.Duration, clientParams util.ClientParameters) KubernetesCoreInterface {
	var client KubernetesCoreInterface
	var err error
	operation := func() error {
		client, err = createKubernetesCore(clientParams)
		return err
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	if err != nil {
		glog.Fatalf("Failed to create pod client. Error: %v", err)
	}
	return client
}
