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

func createPodClient(namespace string) (v1.PodInterface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client.")
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client set.")
	}
	return clientSet.CoreV1().Pods(namespace), nil
}

// CreatePodClientOrFatal creates a new client for the Kubernetes pod.
func CreatePodClientOrFatal(namespace string, initConnectionTimeout time.Duration) v1.PodInterface{
	var client v1.PodInterface
	var err error
	var operation = func() error {
		client, err = createPodClient(namespace)
		if err != nil {
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	if err != nil {
		glog.Fatalf("Failed to create pod client. Error: %v", err)
	}
	return client
}
