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

package client

import (
	"fmt"
	"net/http"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	minio "github.com/minio/minio-go"
	credentials "github.com/minio/minio-go/pkg/credentials"
	"github.com/pkg/errors"
)

func getEndpoint(host, port string) string {
	if port == "" {
		return host
	}
	return fmt.Sprintf("%s:%s", host, port)
}

// createCredentialProvidersChain creates a chained providers credential for a minio client
func createCredentialProvidersChain(endpoint, accessKey, secretKey, region string) *credentials.Credentials {
	var providers []credentials.Provider = []credentials.Provider{}
	// first try with static api key
	if accessKey != "" && secretKey != "" {
		staticCred := &credentials.Static{
			Value: credentials.Value{
				AccessKeyID:     accessKey,
				SecretAccessKey: secretKey,
				SessionToken:    "",
				SignerType:      credentials.SignatureV4,
			},
		}
		providers = append(providers, staticCred)
	}

	minioEnv := &credentials.EnvMinio{}
	awsEnv := &credentials.EnvAWS{}
	awsIAM := &credentials.IAM{
		Client: &http.Client{
			Transport: http.DefaultTransport,
		},
	}
	providers = append(providers, minioEnv, awsEnv, awsIAM)
	return credentials.New(&credentials.Chain{Providers: providers})
}

func CreateMinioClient(minioServiceHost string, minioServicePort string,
	accessKey string, secretKey string, secure bool, region string) (*minio.Client, error) {

	endpoint := getEndpoint(minioServiceHost, minioServicePort)
	cred := createCredentialProvidersChain(endpoint, accessKey, secretKey, region)

	minioClient, err := minio.NewWithCredentials(endpoint, cred, secure, region)
	if err != nil {
		return nil, errors.Wrapf(err, "Error while creating minio client: %+v", err)
	}
	return minioClient, nil
}

func CreateMinioClientOrFatal(minioServiceHost string, minioServicePort string,
	accessKey string, secretKey string, secure bool, region string, initConnectionTimeout time.Duration) *minio.Client {
	var minioClient *minio.Client
	var err error
	var operation = func() error {
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
		glog.Fatalf("Failed to create Minio client. Error: %v", err)
	}
	return minioClient
}

// joinHostPort combines host and port into a network address of the form "host:port".
//
// An empty port value results in "host" instead of "host:" (which net.JoinHostPort would return)
func joinHostPort(host, port string) string {
	if port == "" {
		return host
	}
	return fmt.Sprintf("%s:%s", host, port)
}
