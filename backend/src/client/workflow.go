package client

import (
	"time"

	argoclient "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/client-go/rest"
)

func CreateWorkflowClient(namespace string) (v1alpha1.WorkflowInterface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize workflow client.")
	}
	wfClientSet := argoclient.NewForConfigOrDie(restConfig)
	wfClient := wfClientSet.ArgoprojV1alpha1().Workflows(namespace)
	return wfClient, nil
}

// creates a new client for the Kubernetes Workflow CRD.
func CreateWorkflowClientOrFatal(namespace string, initConnectionTimeout time.Duration) v1alpha1.WorkflowInterface {
	var wfClient v1alpha1.WorkflowInterface
	var err error
	var operation = func() error {
		wfClient, err = CreateWorkflowClient(namespace)
		if err != nil {
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	if err != nil {
		glog.Fatalf("Failed to create workflow client. Error: %v", err)
	}
	return wfClient
}
