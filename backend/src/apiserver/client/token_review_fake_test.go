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

	authv1 "k8s.io/api/authentication/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFakeTokenReviewClient_Create(t *testing.T) {
	client := NewFakeTokenReviewClient()
	review := &authv1.TokenReview{}

	result, err := client.Create(context.Background(), review, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if !result.Status.Authenticated {
		t.Error("Create() Status.Authenticated = false, want true")
	}
	if result.Status.User.Username != "test" {
		t.Errorf("Create() Status.User.Username = %q, want %q", result.Status.User.Username, "test")
	}
}

func TestFakeTokenReviewClientUnauthenticated_Create(t *testing.T) {
	client := NewFakeTokenReviewClientUnauthenticated()
	review := &authv1.TokenReview{}

	result, err := client.Create(context.Background(), review, v1.CreateOptions{})
	if err != nil {
		t.Fatalf("Create() unexpected error: %v", err)
	}
	if result.Status.Authenticated {
		t.Error("Create() Status.Authenticated = true, want false")
	}
	if result.Status.Error != "Unauthenticated" {
		t.Errorf("Create() Status.Error = %q, want %q", result.Status.Error, "Unauthenticated")
	}
}

func TestFakeTokenReviewClientError_Create(t *testing.T) {
	client := NewFakeTokenReviewClientError()
	review := &authv1.TokenReview{}

	_, err := client.Create(context.Background(), review, v1.CreateOptions{})
	if err == nil {
		t.Error("Create() expected error, got nil")
	}
}
