package client

import (
	argoclient "github.com/argoproj/argo/pkg/client/clientset/versioned"
	"github.com/argoproj/argo/pkg/client/clientset/versioned/typed/workflow/v1alpha1"
	"github.com/pkg/errors"
	k8sclient "k8s.io/api/core/v1"
	"k8s.io/client-go/rest"
)

func CreateWorkflowClient() (v1alpha1.WorkflowInterface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize workflow client.")
	}
	wfClientSet := argoclient.NewForConfigOrDie(restConfig)
	wfClient := wfClientSet.ArgoprojV1alpha1().Workflows(k8sclient.NamespaceDefault)
	return wfClient, nil
}
