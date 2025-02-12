// Copyright 2018 The Kubeflow Authors
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

package client

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	minio "github.com/minio/minio-go/v6"
	credentials "github.com/minio/minio-go/v6/pkg/credentials"
	"github.com/pkg/errors"
)

// createCredentialProvidersChain creates a chained providers credential for a minio client.
func createCredentialProvidersChain(endpoint, accessKey, secretKey string) *credentials.Credentials {
	// first try with static api key
	if accessKey != "" && secretKey != "" {
		return credentials.NewStaticV4(accessKey, secretKey, "")
	}
	// otherwise use a chained provider: minioEnv -> awsEnv -> IAM
	providers := []credentials.Provider{
		&credentials.EnvMinio{},
		&credentials.EnvAWS{},
		&credentials.IAM{
			Client: &http.Client{
				Transport: http.DefaultTransport,
			},
		},
	}
	return credentials.New(&credentials.Chain{Providers: providers})
}

func CreateMinioClient(minioServiceHost string, minioServicePort string,
	accessKey string, secretKey string, secure bool, region string,
) (*minio.Client, error) {
	endpoint := joinHostPort(minioServiceHost, minioServicePort)
	cred := createCredentialProvidersChain(endpoint, accessKey, secretKey)
	minioClient, err := minio.NewWithCredentials(endpoint, cred, secure, region)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while creating object store client: %+v", err)
	}
	return minioClient, nil
}

func CreateMinioClientOrFatal(minioServiceHost string, minioServicePort string,
	accessKey string, secretKey string, secure bool, region string, initConnectionTimeout time.Duration,
) *minio.Client {
	var minioClient *minio.Client
	var err error
	operation := func() error {
		minioClient, err = CreateMinioClient(minioServiceHost, minioServicePort,
			accessKey, secretKey, secure, region)
		if err != nil {
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)
	if err != nil {
		glog.Fatalf("Failed to create object store client. Error: %v", err)
	}
	return minioClient
}

// joinHostPort combines host and port into a network address of the form "host:port".
//
// An empty port value results in "host" instead of "host:" (which net.JoinHostPort would return).
func joinHostPort(host, port string) string {
	if port == "" {
		return host
	}
	return fmt.Sprintf("%s:%s", host, port)
}
