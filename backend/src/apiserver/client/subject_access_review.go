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
	"context"
	"time"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	authzv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SubjectAccessReviewInterface interface {
	Create(ctx context.Context, sar *authzv1.SubjectAccessReview, opts v1.CreateOptions) (result *authzv1.SubjectAccessReview, err error)
}

func createSubjectAccessReviewClient(clientParams util.ClientParameters) (SubjectAccessReviewInterface, error) {
	clientSet, err := getKubernetesClientset(clientParams)
	if err != nil {
		return nil, err
	}
	return clientSet.AuthorizationV1().SubjectAccessReviews(), nil
}

// CreateSubjectAccessReviewClientOrFatal creates a new SubjectAccessReview client.
func CreateSubjectAccessReviewClientOrFatal(initConnectionTimeout time.Duration, clientParams util.ClientParameters) SubjectAccessReviewInterface {
	var client SubjectAccessReviewInterface
	var err error
	var operation = func() error {
		client, err = createSubjectAccessReviewClient(clientParams)
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
