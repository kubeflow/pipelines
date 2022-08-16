package client

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
)

type KubernetesCoreFake struct {
	coreV1ClientFake *FakeNamespaceClient
}

func (c *KubernetesCoreFake) NamespaceClient() v1.NamespaceInterface {
	return c.coreV1ClientFake
}

func (c *KubernetesCoreFake) GetNamespaceOwner(namespace string) (string, error) {
	ns, err := c.NamespaceClient().Get(context.Background(), namespace, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	owner, ok := ns.Annotations["owner"]
	if !ok {
		return "", errors.New(fmt.Sprintf("namespace '%v' has no owner in the annotations", namespace))
	}
	return owner, nil
}

func NewKubernetesCoreFake() *KubernetesCoreFake {
	return &KubernetesCoreFake{&FakeNamespaceClient{}}
}
func (c *KubernetesCoreFake) Set(namespaceToReturn string, userToReturn string) {
	c.coreV1ClientFake.SetReturnValues(namespaceToReturn, userToReturn)
}
