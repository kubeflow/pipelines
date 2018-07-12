package client

import (
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	swfclient "github.com/googleprivate/ml/backend/src/crd/pkg/client/clientset/versioned"
	"github.com/googleprivate/ml/backend/src/crd/pkg/client/clientset/versioned/typed/scheduledworkflow/v1alpha1"
	"k8s.io/client-go/rest"
)

// creates a new client for the Kubernetes ScheduledWorkflow CRD.
func CreateScheduledWorkflowClientOrFatal(namespace string, initConnectionTimeout time.Duration) v1alpha1.ScheduledWorkflowInterface {
	var swfClient v1alpha1.ScheduledWorkflowInterface
	var err error
	var operation = func() error {
		restConfig, err := rest.InClusterConfig()
		if err != nil {
			return err
		}
		swfClientSet := swfclient.NewForConfigOrDie(restConfig)
		swfClient = swfClientSet.ScheduledworkflowV1alpha1().ScheduledWorkflows(namespace)
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	if err != nil {
		glog.Fatalf("Failed to create scheduled workflow client. Error: %v", err)
	}
	return swfClient
}
