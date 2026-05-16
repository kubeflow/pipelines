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

package main

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	util "github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsV1PipelineBlocked(t *testing.T) {
	tests := []struct {
		name              string
		blockV1           string
		allowedNamespaces string
		namespace         string
		expected          bool
	}{
		{
			name:      "Blocking disabled - not blocked",
			blockV1:   "false",
			namespace: "ns1",
			expected:  false,
		},
		{
			name:      "Blocking not set - not blocked",
			blockV1:   "",
			namespace: "ns1",
			expected:  false,
		},
		{
			name:      "Blocking enabled, no allowed namespaces - blocked",
			blockV1:   "true",
			namespace: "ns1",
			expected:  true,
		},
		{
			name:              "Blocking enabled, namespace allowed - not blocked",
			blockV1:           "true",
			allowedNamespaces: "ns1",
			namespace:         "ns1",
			expected:          false,
		},
		{
			name:              "Blocking enabled, namespace not in allowed list - blocked",
			blockV1:           "true",
			allowedNamespaces: "ns2,ns3",
			namespace:         "ns1",
			expected:          true,
		},
		{
			name:              "Blocking enabled, namespace in allowed list - not blocked",
			blockV1:           "true",
			allowedNamespaces: "ns1,ns2,ns3",
			namespace:         "ns2",
			expected:          false,
		},
		{
			name:              "Blocking enabled, case insensitive namespace match - not blocked",
			blockV1:           "true",
			allowedNamespaces: "NS1",
			namespace:         "ns1",
			expected:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set(common.BlockV1Pipelines, tt.blockV1)
			viper.Set(common.V1NamespaceWhitelist, tt.allowedNamespaces)
			defer func() {
				viper.Set(common.BlockV1Pipelines, "")
				viper.Set(common.V1NamespaceWhitelist, "")
			}()

			result := isV1PipelineBlocked(tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestShouldEnforceV1Block(t *testing.T) {
	tests := []struct {
		name       string
		apiVersion string
		blockV1    string
		namespace  string
		expected   bool
	}{
		{
			name:       "V1 SWF, blocking enabled - blocked",
			apiVersion: commonutil.ApiVersionV1,
			blockV1:    "true",
			namespace:  "ns1",
			expected:   true,
		},
		{
			name:       "V1 SWF, blocking disabled - not blocked",
			apiVersion: commonutil.ApiVersionV1,
			blockV1:    "false",
			namespace:  "ns1",
			expected:   false,
		},
		{
			name:       "V2 SWF, blocking enabled - not blocked",
			apiVersion: commonutil.ApiVersionV2,
			blockV1:    "true",
			namespace:  "ns1",
			expected:   false,
		},
		{
			name:       "V2 SWF, blocking disabled - not blocked",
			apiVersion: commonutil.ApiVersionV2,
			blockV1:    "false",
			namespace:  "ns1",
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set(common.BlockV1Pipelines, tt.blockV1)
			defer viper.Set(common.BlockV1Pipelines, "")

			swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
				TypeMeta:   metav1.TypeMeta{APIVersion: tt.apiVersion},
				ObjectMeta: metav1.ObjectMeta{Namespace: tt.namespace},
			})
			assert.Equal(t, tt.expected, shouldEnforceV1Block(swf))
		})
	}
}
