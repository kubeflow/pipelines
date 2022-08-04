package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"

	"github.com/kubeflow/pipelines/backend/src/common/util"
)

type KubernetesCoreInterface interface {
	NamespaceClient() v1.NamespaceInterface
	GetNamespaceOwner(namespace string) (string, error)
}

type KubernetesCore struct {
	coreV1Client v1.CoreV1Interface
}

func (c *KubernetesCore) NamespaceClient() v1.NamespaceInterface {
	return c.coreV1Client.Namespaces()
}

func (c *KubernetesCore) GetNamespaceOwner(namespace string) (string, error) {
	if os.Getenv("MULTIUSER") == "" || os.Getenv("MULTIUSER") == "false" {
		return "", nil
	}
	ns, err := c.NamespaceClient().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, fmt.Sprintf("failed to get namespace '%v'", namespace))
	}
	owner, ok := ns.Annotations["owner"]
	if !ok {
		return "", errors.New(fmt.Sprintf("namespace '%v' has no owner in the annotations", namespace))
	}
	return owner, nil
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
	var operation = func() error {
		client, err = createKubernetesCore(clientParams)
		return err
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	if err != nil {
		glog.Fatalf("Failed to create namespace client. Error: %v", err)
	}
	return client
}

func getKubernetesClientset(clientParams util.ClientParameters) (*kubernetes.Clientset, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client.")
	}
	restConfig.QPS = float32(clientParams.QPS)
	restConfig.Burst = clientParams.Burst

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client set.")
	}
	return clientSet, nil
}
