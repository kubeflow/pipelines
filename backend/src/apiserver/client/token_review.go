// Copyright 2021 Arrikto Inc.
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

	"github.com/kubeflow/pipelines/backend/src/common/util"

	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type TokenReviewInterface interface {
	Create(ctx context.Context, tokenReview *authv1.TokenReview, opts v1.CreateOptions) (result *authv1.TokenReview, err error)
}

func createTokenReviewClient(clientParams util.ClientParameters) (TokenReviewInterface, error) {
	clientSet, err := getKubernetesClientset(clientParams)
	if err != nil {
		return nil, err
	}
	return clientSet.AuthenticationV1().TokenReviews(), nil
}

// CreateTokenReviewClientOrFatal creates a new TokenReview client.
func CreateTokenReviewClientOrFatal(initConnectionTimeout time.Duration, clientParams util.ClientParameters) TokenReviewInterface {
	var client TokenReviewInterface
	var err error
	var operation = func() error {
		client, err = createTokenReviewClient(clientParams)
		return err
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err = backoff.Retry(operation, b)

	if err != nil {
		glog.Fatalf("Failed to create TokenReview client. Error: %v", err)
	}
	return client
}
