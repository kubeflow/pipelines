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
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/pkg/errors"
	authv1 "k8s.io/api/authentication/v1"
)

type FakeTokenReviewClient struct {
}

func (FakeTokenReviewClient) Create(*authv1.TokenReview) (*authv1.TokenReview, error) {
	return &authv1.TokenReview{Status: authv1.TokenReviewStatus{
		Authenticated: true,
		User:          authv1.UserInfo{Username: "test"},
		Audiences:     []string{common.GetTokenReviewAudience()},
		Error:         "",
	}}, nil
}

func NewFakeTokenReviewClient() FakeTokenReviewClient {
	return FakeTokenReviewClient{}
}

type FakeTokenReviewClientUnauthenticated struct {
}

func (FakeTokenReviewClientUnauthenticated) Create(*authv1.TokenReview) (*authv1.TokenReview, error) {
	return &authv1.TokenReview{Status: authv1.TokenReviewStatus{
		Authenticated: false,
		User:          authv1.UserInfo{},
		Audiences:     nil,
		Error:         "Unauthenticated",
	}}, nil
}

func NewFakeTokenReviewClientUnauthenticated() FakeTokenReviewClientUnauthenticated {
	return FakeTokenReviewClientUnauthenticated{}
}

type FakeTokenReviewClientError struct {
}

func (FakeTokenReviewClientError) Create(*authv1.TokenReview) (*authv1.TokenReview, error) {
	return nil, errors.New("failed to create token review")
}

func NewFakeTokenReviewClientError() FakeTokenReviewClientError {
	return FakeTokenReviewClientError{}
}
