// Copyright 2026 The Kubeflow Authors
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
	"testing"

	authzv1 "k8s.io/api/authorization/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFakeSubjectAccessReviewClient_Create(t *testing.T) {
	client := NewFakeSubjectAccessReviewClient()
	review := &authzv1.SubjectAccessReview{}

	result, err := client.Create(context.Background(), review, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if !result.Status.Allowed {
		t.Error("Create() Status.Allowed = false, want true")
	}
	if result.Status.Denied {
		t.Error("Create() Status.Denied = true, want false")
	}
}

func TestFakeSubjectAccessReviewClientUnauthorized_Create(t *testing.T) {
	client := NewFakeSubjectAccessReviewClientUnauthorized()
	review := &authzv1.SubjectAccessReview{}

	result, err := client.Create(context.Background(), review, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if result.Status.Allowed {
		t.Error("Create() Status.Allowed = true, want false")
	}
	if result.Status.Reason != "this is not allowed" {
		t.Errorf("Create() Status.Reason = %q, want %q", result.Status.Reason, "this is not allowed")
	}
}

func TestFakeSubjectAccessReviewClientError_Create(t *testing.T) {
	client := NewFakeSubjectAccessReviewClientError()
	review := &authzv1.SubjectAccessReview{}

	_, err := client.Create(context.Background(), review, v1.CreateOptions{})
	if err == nil {
		t.Error("Create() expected error, got nil")
	}
}
