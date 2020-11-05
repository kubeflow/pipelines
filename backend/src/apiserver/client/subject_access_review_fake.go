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
	"github.com/pkg/errors"
	authzv1 "k8s.io/api/authorization/v1"
)

type FakeSubjectAccessReviewClient struct {
}

func (FakeSubjectAccessReviewClient) Create(*authzv1.SubjectAccessReview) (*authzv1.SubjectAccessReview, error) {
	return &authzv1.SubjectAccessReview{Status: authzv1.SubjectAccessReviewStatus{
		Allowed:         true,
		Denied:          false,
		Reason:          "",
		EvaluationError: "",
	}}, nil
}

func NewFakeSubjectAccessReviewClient() FakeSubjectAccessReviewClient {
	return FakeSubjectAccessReviewClient{}
}

type FakeSubjectAccessReviewClientUnauthorized struct {
}

func (FakeSubjectAccessReviewClientUnauthorized) Create(*authzv1.SubjectAccessReview) (*authzv1.SubjectAccessReview, error) {
	return &authzv1.SubjectAccessReview{Status: authzv1.SubjectAccessReviewStatus{
		Allowed:         false,
		Denied:          false,
		Reason:          "this is not allowed",
		EvaluationError: "",
	}}, nil
}

func NewFakeSubjectAccessReviewClientUnauthorized() FakeSubjectAccessReviewClientUnauthorized {
	return FakeSubjectAccessReviewClientUnauthorized{}
}

type FakeSubjectAccessReviewClientError struct {
}

func (FakeSubjectAccessReviewClientError) Create(*authzv1.SubjectAccessReview) (*authzv1.SubjectAccessReview, error) {
	return nil, errors.New("failed to create subject access review")
}

func NewFakeSubjectAccessReviewClientError() FakeSubjectAccessReviewClientError {
	return FakeSubjectAccessReviewClientError{}
}
