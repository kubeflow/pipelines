// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"fmt"
	"strings"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func WaitForGrpcClientAvailable(
	namespace string,
	initializeTimeout time.Duration,
	basePath string,
	masterurl string,
	kubeconfig string) error {

	formattedBasePath := fmt.Sprintf(basePath, namespace, "healthz")

	clientSet, _, err := GetKubernetesClient(masterurl, kubeconfig)
	if err != nil {
		return errors.Wrapf(err,
			"Failed to get the clientSet when waiting for service (%v) to be ready in namespace (%v).",
			formattedBasePath, namespace)
	}

	var operation = func() error {
		response := clientSet.RESTClient().Get().
			AbsPath(formattedBasePath).Do()
		if response.Error() == nil {
			return nil
		}
		var code int
		response.StatusCode(&code)
		// we wait only on 503 service unavailable. Stop retry otherwise.
		if code != 503 {
			return backoff.Permanent(errors.Wrapf(response.Error(),
				"Could not reach service (%v) in namespace (%v), retrying....",
				formattedBasePath, namespace))
		}
		return response.Error()
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initializeTimeout
	err = backoff.Retry(operation, b)
	return errors.Wrapf(err,
		"Waiting for service (%v) in namespace (%v) failed after all attempts.",
		formattedBasePath, namespace)
}

func GetKubernetesClient(masterUrl string, kubeconfig string) (*kubernetes.Clientset, *rest.Config,
	error) {
	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags(masterUrl, kubeconfig)
	if err != nil {
		return nil, nil, errors.Wrapf(err,
			"Failed to get cluster config during K8s client initialization")
	}
	// create the clientset
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, errors.Wrapf(err,
			"Failed to create client set during K8s client initialization")
	}

	return clientSet, config, nil
}

func GetKubernetesClientFromClientConfig(clientConfig clientcmd.ClientConfig) (
	*kubernetes.Clientset, *rest.Config, string, error) {
	// Get the clientConfig
	config, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, nil, "", errors.Wrapf(err,
			"Failed to get cluster config during K8s client initialization")
	}
	// Get namespace
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		return nil, nil, "", errors.Wrapf(err,
			"Failed to get the namespace during K8s client initialization")
	}
	// create the clientset
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, "", errors.Wrapf(err,
			"Failed to create client set during K8s client initialization")
	}
	return clientSet, config, namespace, nil
}

func GetRpcConnection(namespace string, serviceName string, servicePort string,
	masterurl string, kubeconfig string) (*grpc.ClientConn, error) {
	clientSet, _, err := GetKubernetesClient(masterurl, kubeconfig)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get K8s client set when getting RPC connection")
	}
	svc, err := clientSet.CoreV1().Services(namespace).Get(serviceName, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get ml-pipeline service")
	}
	rpcAddress := svc.Spec.ClusterIP + ":" + servicePort
	conn, err := grpc.Dial(rpcAddress, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create gRPC connection")
	}
	return conn, nil
}

func ExtractMasterIPAndPort(config *rest.Config) string {
	host := config.Host
	host = strings.TrimPrefix(host, "http://")
	host = strings.TrimPrefix(host, "https://")
	return host
}
