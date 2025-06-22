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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/pkg/errors"
)

// createCredentialsProvider creates AWS credentials for S3 client
func createCredentialsProvider(accessKey, secretKey string) *credentials.Credentials {
	// If static credentials are provided, use them
	if accessKey != "" && secretKey != "" {
		return credentials.NewStaticCredentials(accessKey, secretKey, "")
	}
	// Otherwise use environment variables
	return credentials.NewEnvCredentials()
}

func CreateS3Client(serviceHost string, servicePort string,
	accessKey string, secretKey string, secure bool, region string,
) (*s3.S3, error) {
	endpoint := joinHostPort(serviceHost, servicePort)

	// Build endpoint URL
	scheme := "http"
	if secure {
		scheme = "https"
	}
	endpointURL := fmt.Sprintf("%s://%s", scheme, endpoint)

	creds := createCredentialsProvider(accessKey, secretKey)

	// Create AWS session with custom endpoint for SeaweedFS
	sess, err := session.NewSession(&aws.Config{
		Endpoint:         aws.String(endpointURL),
		Region:           aws.String(region),
		Credentials:      creds,
		S3ForcePathStyle: aws.Bool(true), // Required for SeaweedFS compatibility
		DisableSSL:       aws.Bool(!secure),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "Error while creating S3 session: %+v", err)
	}

	return s3.New(sess), nil
}

func CreateS3ClientOrFatal(serviceHost string, servicePort string,
	accessKey string, secretKey string, secure bool, region string, initConnectionTimeout time.Duration,
) *s3.S3 {
	var s3Client *s3.S3
	var err error
	operation := func() error {
		s3Client, err = CreateS3Client(serviceHost, servicePort,
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
		glog.Fatalf("Failed to create S3 client. Error: %v", err)
	}
	return s3Client
}

// Legacy function name for backward compatibility - now creates S3 client
func CreateMinioClient(minioServiceHost string, minioServicePort string,
	accessKey string, secretKey string, secure bool, region string,
) (*s3.S3, error) {
	return CreateS3Client(minioServiceHost, minioServicePort, accessKey, secretKey, secure, region)
}

// Legacy function name for backward compatibility - now creates S3 client
func CreateMinioClientOrFatal(minioServiceHost string, minioServicePort string,
	accessKey string, secretKey string, secure bool, region string, initConnectionTimeout time.Duration,
) *s3.S3 {
	return CreateS3ClientOrFatal(minioServiceHost, minioServicePort, accessKey, secretKey, secure, region, initConnectionTimeout)
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
