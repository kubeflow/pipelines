// Copyright 2018 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package test

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"flag"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	// ML pipeline API server root URL
	mlPipelineAPIServerBase = "/api/v1/namespaces/%s/services/ml-pipeline:8888/proxy/apis/v1beta1/%s"
)

var namespace = flag.String("namespace", "kubeflow", "The namespace ml pipeline deployed to")
var initializeTimeout = flag.Duration("initializeTimeout", 2*time.Minute, "Duration to wait for test initialization")

func getKubernetesClient() (*kubernetes.Clientset, error) {
	// use the current context in kubeconfig
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get cluster config during K8s client initialization")
	}
	// create the clientset
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create client set during K8s client initialization")
	}

	return clientSet, nil
}

func waitForReady(namespace string, initializeTimeout time.Duration) error {
	clientSet, err := getKubernetesClient()
	if err != nil {
		return errors.Wrapf(err, "Failed to get K8s client set when waiting for ML pipeline to be ready")
	}

	var operation = func() error {
		response := clientSet.RESTClient().Get().
			AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, namespace, "healthz")).Do()
		if response.Error() == nil {
			return nil
		}
		var code int
		response.StatusCode(&code)
		// we wait only on 503 service unavailable. Stop retry otherwise.
		if code != 503 {
			return backoff.Permanent(errors.Wrapf(response.Error(), "Waiting for ml pipeline failed with non retriable error."))
		}
		return response.Error()
	}

	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initializeTimeout
	err = backoff.Retry(operation, b)
	return errors.Wrapf(err, "Waiting for ml pipeline failed after all attempts.")
}

func uploadPipelineFileOrFail(path string) (*bytes.Buffer, *multipart.Writer) {
	file, err := os.Open(path)
	if err != nil {
		glog.Fatalf("Failed to open the pipeline file: %v", err.Error())
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("uploadfile", filepath.Base(path))
	if err != nil {
		glog.Fatalf("Failed to create form file: %v", err.Error())
	}
	_, err = io.Copy(part, file)
	if err != nil {
		glog.Fatalf("Failed to copy file to multipart writer: %v", err.Error())
	}

	err = writer.Close()
	if err != nil {
		glog.Fatalf("Failed to close multipart writer: %v", err.Error())
	}

	return body, writer
}

func getRpcConnection(namespace string) (*grpc.ClientConn, error) {
	clientSet, err := getKubernetesClient()
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get K8s client set when getting RPC connection")
	}
	svc, err := clientSet.CoreV1().Services(namespace).Get("ml-pipeline", metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to get ml-pipeline service")
	}
	rpcAddress := svc.Spec.ClusterIP + ":8887"
	conn, err := grpc.Dial(rpcAddress, grpc.WithInsecure())
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to create gRPC connection")
	}
	return conn, nil
}
