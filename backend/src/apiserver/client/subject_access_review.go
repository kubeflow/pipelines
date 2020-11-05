// Copyright 2020 Arrikto Inc.
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
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	authzv1 "k8s.io/api/authorization/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type SubjectAccessReviewInterface interface {
	Create(sar *authzv1.SubjectAccessReview) (result *authzv1.SubjectAccessReview, err error)
}

func createSubjectAccessReviewClient() (SubjectAccessReviewInterface, error) {
	restConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client.")
	}

	clientSet, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to initialize kubernetes client set.")
	}
	return clientSet.AuthorizationV1().SubjectAccessReviews(), nil
}

// CreateSubjectAccessReviewClientOrFatal creates a new SubjectAccessReview client.
func CreateSubjectAccessReviewClientOrFatal(initConnectionTimeout time.Duration) SubjectAccessReviewInterface {
	var client SubjectAccessReviewInterface
	var err error
	var operation = func() error {
		client, err = createSubjectAccessReviewClient()
		return err
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	if err != nil {
		glog.Fatalf("Failed to create SubjectAccessReview client. Error: %v", err)
	}
	return client
}
