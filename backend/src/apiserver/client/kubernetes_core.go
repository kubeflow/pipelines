package client

import (
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"time"
)

type KubernetesCoreInterface interface {
	PodClient(namespace string) v1.PodInterface
}

type KubernetesCore struct {
	coreV1Client v1.CoreV1Interface
}

func (c *KubernetesCore) PodClient(namespace string) v1.PodInterface {
	return c.coreV1Client.Pods(namespace)
}

func createKubernetesCore() (KubernetesCoreInterface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client.")
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client set.")
	}
	return &KubernetesCore{clientSet.CoreV1()}, nil
}

// CreateKubernetesCoreOrFatal creates a new client for the Kubernetes pod.
func CreateKubernetesCoreOrFatal(initConnectionTimeout time.Duration) KubernetesCoreInterface {
	var client KubernetesCoreInterface
	var err error
	var operation = func() error {
		client, err = createKubernetesCore()
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
