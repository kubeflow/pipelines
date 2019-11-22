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

func createK8sClient() (v1.CoreV1Interface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client.")
	}
	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client set.")
	}
	return clientSet.CoreV1(), nil
}

// CreateK8sClientOrFatal creates a new client for the Kubernetes resources.
func CreateK8sClientOrFatal(initConnectionTimeout time.Duration) v1.CoreV1Interface {
	var client v1.CoreV1Interface
	var err error
	var operation = func() error {
		client, err = createK8sClient()
		if err != nil {
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	if err != nil {
		glog.Fatalf("Failed to create k8s client. Error: %v", err)
	}
	return client
}
