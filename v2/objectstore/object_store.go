// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// This package contains helper methods for using object stores.
// TODO: move other object store related methods here.
package objectstore

import (
	"context"
	"fmt"
	"os"

	"github.com/golang/glog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// The endpoint uses Kubernetes service DNS name with namespace:
// https://kubernetes.io/docs/concepts/services-networking/service/#dns
const defaultMinioEndpointInMultiUserMode = "minio-service.kubeflow:9000"
const minioArtifactSecretName = "mlpipeline-minio-artifact"

func MinioDefaultEndpoint() string {
	// Discover minio-service in the same namespace by env var.
	// https://kubernetes.io/docs/concepts/services-networking/service/#environment-variables
	minioHost := os.Getenv("MINIO_SERVICE_SERVICE_HOST")
	minioPort := os.Getenv("MINIO_SERVICE_SERVICE_PORT")
	if minioHost != "" && minioPort != "" {
		// If there is a minio-service Kubernetes service in the same namespace,
		// MINIO_SERVICE_SERVICE_HOST and MINIO_SERVICE_SERVICE_PORT env vars should
		// exist by default, so we use it as default.
		return minioHost + ":" + minioPort
	}
	// If the env vars do not exist, we guess that we are running in KFP multi user mode, so default minio service should be `minio-service.kubeflow:9000`.
	glog.Infof("Cannot detect minio-service in the same namespace, default to %s as MinIO endpoint.", defaultMinioEndpointInMultiUserMode)
	return defaultMinioEndpointInMultiUserMode
}

type MinioCredential struct {
	AccessKey string
	SecretKey string
}

func GetMinioCredential(clientSet *kubernetes.Clientset, namespace string) (cred MinioCredential, err error) {
	defer func() {
		if err != nil {
			// wrap error before returning
			err = fmt.Errorf("Failed to get MinIO credential from secret name=%q namespace=%q: %w", minioArtifactSecretName, namespace, err)
		}
	}()
	secret, err := clientSet.CoreV1().Secrets(namespace).Get(
		context.Background(),
		minioArtifactSecretName,
		metav1.GetOptions{})
	if err != nil {
		return cred, err
	}
	cred.AccessKey = string(secret.Data["accesskey"])
	cred.SecretKey = string(secret.Data["secretkey"])
	if cred.AccessKey == "" {
		return cred, fmt.Errorf("does not have 'accesskey' key")
	}
	if cred.SecretKey == "" {
		return cred, fmt.Errorf("does not have 'secretkey' key")
	}
	return cred, nil
}
