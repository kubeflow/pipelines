// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"slices"
	"testing"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestUnreferencedResourceClaims(t *testing.T) {
	testCases := []struct {
		name string
		pod  *v1.Pod
		want []string
	}{
		{
			name: "claim referenced by annotated default container",
			pod: &v1.Pod{Spec: v1.PodSpec{
				ResourceClaims: []v1.PodResourceClaim{{Name: "gpu"}},
				Containers: []v1.Container{
					{Name: "wait"},
					{Name: "main", Resources: v1.ResourceRequirements{Claims: []v1.ResourceClaim{{Name: "gpu"}}}},
				},
			}, ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{defaultContainerAnnotation: "main"}}},
			want: nil,
		},
		{
			name: "claim not referenced by any container",
			pod: &v1.Pod{Spec: v1.PodSpec{
				ResourceClaims: []v1.PodResourceClaim{{Name: "gpu"}},
				Containers:     []v1.Container{{Name: "main"}},
			}, ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{defaultContainerAnnotation: "main"}}},
			want: []string{"gpu"},
		},
		{
			name: "claim referenced only by non-default container",
			pod: &v1.Pod{Spec: v1.PodSpec{
				ResourceClaims: []v1.PodResourceClaim{{Name: "gpu"}},
				Containers: []v1.Container{
					{Name: "wait", Resources: v1.ResourceRequirements{Claims: []v1.ResourceClaim{{Name: "gpu"}}}},
					{Name: "main"},
				},
			}, ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{defaultContainerAnnotation: "main"}}},
			want: []string{"gpu"},
		},
		{
			name: "claim has no annotated default container",
			pod: &v1.Pod{Spec: v1.PodSpec{
				ResourceClaims: []v1.PodResourceClaim{{Name: "gpu"}},
				Containers: []v1.Container{
					{Name: "main", Resources: v1.ResourceRequirements{Claims: []v1.ResourceClaim{{Name: "gpu"}}}},
				},
			}},
			want: []string{"gpu"},
		},
		{
			name: "one of multiple claims is unreferenced",
			pod: &v1.Pod{Spec: v1.PodSpec{
				ResourceClaims: []v1.PodResourceClaim{{Name: "gpu"}, {Name: "fpga"}},
				Containers: []v1.Container{
					{Name: "main", Resources: v1.ResourceRequirements{Claims: []v1.ResourceClaim{{Name: "gpu"}}}},
				},
			}, ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{defaultContainerAnnotation: "main"}}},
			want: []string{"fpga"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := unreferencedResourceClaims(testCase.pod)
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("unreferencedResourceClaims() = %v, want %v", got, testCase.want)
			}
		})
	}
}

func TestUnallocatedResourceClaims(t *testing.T) {
	boundClaimName := "generated-claim"
	testCases := []struct {
		name string
		pod  *v1.Pod
		want []string
	}{
		{
			name: "claim has matching bound status",
			pod: &v1.Pod{
				Spec: v1.PodSpec{ResourceClaims: []v1.PodResourceClaim{{Name: "gpu"}}},
				Status: v1.PodStatus{ResourceClaimStatuses: []v1.PodResourceClaimStatus{{
					Name: "gpu", ResourceClaimName: &boundClaimName,
				}}},
			},
			want: nil,
		},
		{
			name: "claim has no status",
			pod: &v1.Pod{
				Spec: v1.PodSpec{ResourceClaims: []v1.PodResourceClaim{{Name: "gpu"}}},
			},
			want: []string{"gpu"},
		},
		{
			name: "claim status is not bound",
			pod: &v1.Pod{
				Spec: v1.PodSpec{ResourceClaims: []v1.PodResourceClaim{{Name: "gpu"}}},
				Status: v1.PodStatus{ResourceClaimStatuses: []v1.PodResourceClaimStatus{{
					Name: "gpu",
				}}},
			},
			want: []string{"gpu"},
		},
		{
			name: "one of multiple claims has no matching status",
			pod: &v1.Pod{
				Spec: v1.PodSpec{ResourceClaims: []v1.PodResourceClaim{{Name: "gpu"}, {Name: "fpga"}}},
				Status: v1.PodStatus{ResourceClaimStatuses: []v1.PodResourceClaimStatus{{
					Name: "gpu", ResourceClaimName: &boundClaimName,
				}}},
			},
			want: []string{"fpga"},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := unallocatedResourceClaims(testCase.pod)
			if !slices.Equal(got, testCase.want) {
				t.Fatalf("unallocatedResourceClaims() = %v, want %v", got, testCase.want)
			}
		})
	}
}
