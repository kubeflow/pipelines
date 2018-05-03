package client

import (
	argoclient "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
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
func CreateWorkflowClientOrFatal(namespace string) v1alpha1.WorkflowInterface {
	wfClient, err := CreateWorkflowClient(namespace)
	if err != nil {
		glog.Fatalf("Failed to create workflow client. Error: %v", err)
	}
	return wfClient
}
