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
	"context"
	"errors"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/client"
	util "github.com/kubeflow/pipelines/backend/src/crd/controller/scheduledworkflow/util"
	swfapi "github.com/kubeflow/pipelines/backend/src/crd/pkg/apis/scheduledworkflow/v1beta1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
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
	v1WorkflowSpec := map[string]interface{}{
		"entrypoint": "main",
		"templates": []interface{}{
			map[string]interface{}{
				"name": "main",
				"container": map[string]interface{}{
					"image": "alpine",
				},
			},
		},
	}
	v1Workflow := map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Workflow",
		"spec":       v1WorkflowSpec,
	}
	v2WorkflowSpec := map[string]interface{}{
		"podMetadata": map[string]interface{}{
			"labels": map[string]interface{}{
				util.V2Key: "true",
			},
		},
		"entrypoint": "main",
		"templates": []interface{}{
			map[string]interface{}{
				"name": "main",
				"container": map[string]interface{}{
					"image": "alpine",
				},
			},
		},
	}
	v2Workflow := map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1",
		"kind":       "Workflow",
		"spec":       v2WorkflowSpec,
	}

	tests := []struct {
		name              string
		spec              interface{}
		blockV1           string
		allowedNamespaces string
		namespace         string
		expected          bool
	}{
		{
			name:      "V1 workflow spec, blocking enabled - blocked",
			spec:      v1WorkflowSpec,
			blockV1:   "true",
			namespace: "ns1",
			expected:  true,
		},
		{
			name:      "V1 full workflow, blocking enabled - blocked",
			spec:      v1Workflow,
			blockV1:   "true",
			namespace: "ns1",
			expected:  true,
		},
		{
			name:      "V1 workflow spec string, blocking enabled - blocked",
			spec:      `{"entrypoint":"main","templates":[{"name":"main","container":{"image":"alpine"}}]}`,
			blockV1:   "true",
			namespace: "ns1",
			expected:  true,
		},
		{
			name:      "V1 full workflow string, blocking enabled - blocked",
			spec:      `{"apiVersion":"argoproj.io/v1alpha1","kind":"Workflow","spec":{"entrypoint":"main","templates":[{"name":"main","container":{"image":"alpine"}}]}}`,
			blockV1:   "true",
			namespace: "ns1",
			expected:  true,
		},
		{
			name:      "V2 full workflow on v1beta1 ScheduledWorkflow, blocking enabled - not blocked",
			spec:      v2Workflow,
			blockV1:   "true",
			namespace: "ns1",
			expected:  false,
		},
		{
			name:      "V2 workflow spec on v1beta1 ScheduledWorkflow, blocking enabled - not blocked",
			spec:      v2WorkflowSpec,
			blockV1:   "true",
			namespace: "ns1",
			expected:  false,
		},
		{
			name:      "V2 marker in annotation, blocking enabled - not blocked",
			spec:      `{"spec":{"podMetadata":{"annotations":{"pipelines.kubeflow.org/v2_component":"true"}},"entrypoint":"main"}}`,
			blockV1:   "true",
			namespace: "ns1",
			expected:  false,
		},
		{
			name:      "V2 compatible workflow marker, blocking enabled - not blocked",
			spec:      `{"metadata":{"annotations":{"pipelines.kubeflow.org/v2_pipeline":"true"}},"spec":{"entrypoint":"main"}}`,
			blockV1:   "true",
			namespace: "ns1",
			expected:  false,
		},
		{
			name:      "V2 marker false, blocking enabled - blocked",
			spec:      `{"spec":{"podMetadata":{"labels":{"pipelines.kubeflow.org/v2_component":"false"}},"entrypoint":"main"}}`,
			blockV1:   "true",
			namespace: "ns1",
			expected:  true,
		},
		{
			name:      "V1 workflow spec, blocking disabled - not blocked",
			spec:      v1WorkflowSpec,
			blockV1:   "false",
			namespace: "ns1",
			expected:  false,
		},
		{
			name:              "V1 workflow spec, namespace allowed - not blocked",
			spec:              v1WorkflowSpec,
			blockV1:           "true",
			allowedNamespaces: "ns1",
			namespace:         "ns1",
			expected:          false,
		},
		{
			name:      "Missing workflow spec, blocking enabled - not blocked",
			blockV1:   "true",
			namespace: "ns1",
			expected:  false,
		},
		{
			name:      "Malformed embedded workflow, blocking enabled - blocked",
			spec:      `not valid json`,
			blockV1:   "true",
			namespace: "ns1",
			expected:  true,
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

			swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubeflow.org/v1beta1",
					Kind:       "ScheduledWorkflow",
				},
				ObjectMeta: metav1.ObjectMeta{Namespace: tt.namespace},
				Spec: swfapi.ScheduledWorkflowSpec{
					Workflow: &swfapi.WorkflowResource{
						Spec: tt.spec,
					},
				},
			})
			assert.Equal(t, tt.expected, shouldEnforceV1Block(swf))
		})
	}

	t.Run("Nil ScheduledWorkflow - not blocked", func(t *testing.T) {
		viper.Set(common.BlockV1Pipelines, "true")
		defer viper.Set(common.BlockV1Pipelines, "")

		assert.False(t, shouldEnforceV1Block(nil))
	})
}

func TestSubmitNewWorkflowIfNotAlreadySubmitted_BlockV1AllowsV2(t *testing.T) {
	tests := []struct {
		name          string
		workflowSpec  interface{}
		expectCreated bool
		expectError   string
	}{
		{
			name: "blocks v1 embedded workflow",
			workflowSpec: `{
				"apiVersion": "argoproj.io/v1alpha1",
				"kind": "Workflow",
				"spec": {
					"entrypoint": "main",
					"templates": [
						{"name": "main", "container": {"image": "alpine"}}
					]
				}
			}`,
			expectCreated: false,
			expectError:   "not allowed to run v1 pipelines",
		},
		{
			name: "allows v2 embedded workflow",
			workflowSpec: `{
				"apiVersion": "argoproj.io/v1alpha1",
				"kind": "Workflow",
				"spec": {
					"podMetadata": {
						"labels": {
							"pipelines.kubeflow.org/v2_component": "true"
						}
					},
					"entrypoint": "main",
					"templates": [
						{"name": "main", "container": {"image": "alpine"}}
					]
				}
			}`,
			expectCreated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			viper.Set(common.BlockV1Pipelines, "true")
			viper.Set(common.V1NamespaceWhitelist, "")
			defer func() {
				viper.Set(common.BlockV1Pipelines, "")
				viper.Set(common.V1NamespaceWhitelist, "")
			}()

			executionClient := &fakeExecutionClient{}
			controller := &Controller{
				workflowClient: client.NewWorkflowClient(executionClient, &fakeExecutionInformer{}),
			}
			swf := util.NewScheduledWorkflow(&swfapi.ScheduledWorkflow{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "kubeflow.org/v1beta1",
					Kind:       "ScheduledWorkflow",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "scheduled-workflow",
					Namespace: "blocked-namespace",
					UID:       "scheduled-workflow-uid",
				},
				Spec: swfapi.ScheduledWorkflowSpec{
					Workflow: &swfapi.WorkflowResource{
						Spec: tt.workflowSpec,
					},
				},
			})

			submitted, workflowName, err := controller.submitNewWorkflowIfNotAlreadySubmitted(
				context.Background(), swf, 100, 200)

			if tt.expectError != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectError)
				assert.False(t, submitted)
				assert.Empty(t, workflowName)
			} else {
				require.NoError(t, err)
				assert.True(t, submitted)
				assert.NotEmpty(t, workflowName)
			}
			assert.Equal(t, tt.expectCreated, executionClient.createdWorkflow != nil)
		})
	}
}

type fakeExecutionClient struct {
	createdWorkflow commonutil.ExecutionSpec
}

func (f *fakeExecutionClient) Execution(namespace string) commonutil.ExecutionInterface {
	return &fakeExecutionInterface{client: f}
}

func (f *fakeExecutionClient) Compare(old, new interface{}) bool {
	return false
}

type fakeExecutionInterface struct {
	client *fakeExecutionClient
}

func (f *fakeExecutionInterface) Create(ctx context.Context, execution commonutil.ExecutionSpec,
	opts metav1.CreateOptions) (commonutil.ExecutionSpec, error) {
	f.client.createdWorkflow = execution
	return execution, nil
}

func (f *fakeExecutionInterface) Update(ctx context.Context, execution commonutil.ExecutionSpec,
	opts metav1.UpdateOptions) (commonutil.ExecutionSpec, error) {
	return execution, nil
}

func (f *fakeExecutionInterface) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return nil
}

func (f *fakeExecutionInterface) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions,
	listOpts metav1.ListOptions) error {
	return nil
}

func (f *fakeExecutionInterface) Get(ctx context.Context, name string,
	opts metav1.GetOptions) (commonutil.ExecutionSpec, error) {
	return nil, nil
}

func (f *fakeExecutionInterface) List(ctx context.Context,
	opts metav1.ListOptions) (*commonutil.ExecutionSpecList, error) {
	return &commonutil.ExecutionSpecList{}, nil
}

func (f *fakeExecutionInterface) Patch(ctx context.Context, name string, pt types.PatchType, data []byte,
	opts metav1.PatchOptions, subresources ...string) (commonutil.ExecutionSpec, error) {
	return nil, nil
}

type fakeExecutionInformer struct{}

func (f *fakeExecutionInformer) AddEventHandler(funcs cache.ResourceEventHandler) (
	cache.ResourceEventHandlerRegistration, error) {
	return nil, nil
}

func (f *fakeExecutionInformer) HasSynced() func() bool {
	return func() bool { return true }
}

func (f *fakeExecutionInformer) Get(namespace string, name string) (commonutil.ExecutionSpec, bool, error) {
	return nil, true, errors.New("not found")
}

func (f *fakeExecutionInformer) List(labels *labels.Selector) (commonutil.ExecutionSpecList, error) {
	return commonutil.ExecutionSpecList{}, nil
}

func (f *fakeExecutionInformer) InformerFactoryStart(stopCh <-chan struct{}) {}

var _ commonutil.ExecutionClient = &fakeExecutionClient{}
var _ commonutil.ExecutionInterface = &fakeExecutionInterface{}
var _ commonutil.ExecutionInformer = &fakeExecutionInformer{}

func TestCrdPluginsInputToProto(t *testing.T) {
	t.Run("valid single plugin", func(t *testing.T) {
		input := map[string]apiextensionsv1.JSON{
			"mlflow": {Raw: []byte(`{"experiment_name":"my-exp"}`)},
		}
		result, err := crdPluginsInputToProto(input)
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Contains(t, result, "mlflow")
		assert.Equal(t, "my-exp", result["mlflow"].Fields["experiment_name"].GetStringValue())
	})

	t.Run("valid multiple plugins", func(t *testing.T) {
		input := map[string]apiextensionsv1.JSON{
			"mlflow": {Raw: []byte(`{"experiment_name":"exp-1"}`)},
			"other":  {Raw: []byte(`{"key":"val"}`)},
		}
		result, err := crdPluginsInputToProto(input)
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Contains(t, result, "mlflow")
		assert.Contains(t, result, "other")
	})

	t.Run("empty map", func(t *testing.T) {
		result, err := crdPluginsInputToProto(map[string]apiextensionsv1.JSON{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("invalid inner value", func(t *testing.T) {
		input := map[string]apiextensionsv1.JSON{
			"mlflow": {Raw: []byte(`"not-an-object"`)},
		}
		_, err := crdPluginsInputToProto(input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plugins_input entry")
	})
}

func TestCrdPluginsInputToProto(t *testing.T) {
	t.Run("valid single plugin", func(t *testing.T) {
		input := map[string]apiextensionsv1.JSON{
			"mlflow": {Raw: []byte(`{"experiment_name":"my-exp"}`)},
		}
		result, err := crdPluginsInputToProto(input)
		require.NoError(t, err)
		require.Len(t, result, 1)
		require.Contains(t, result, "mlflow")
		assert.Equal(t, "my-exp", result["mlflow"].Fields["experiment_name"].GetStringValue())
	})

	t.Run("valid multiple plugins", func(t *testing.T) {
		input := map[string]apiextensionsv1.JSON{
			"mlflow": {Raw: []byte(`{"experiment_name":"exp-1"}`)},
			"other":  {Raw: []byte(`{"key":"val"}`)},
		}
		result, err := crdPluginsInputToProto(input)
		require.NoError(t, err)
		require.Len(t, result, 2)
		assert.Contains(t, result, "mlflow")
		assert.Contains(t, result, "other")
	})

	t.Run("empty map", func(t *testing.T) {
		result, err := crdPluginsInputToProto(map[string]apiextensionsv1.JSON{})
		require.NoError(t, err)
		assert.Empty(t, result)
	})

	t.Run("invalid inner value", func(t *testing.T) {
		input := map[string]apiextensionsv1.JSON{
			"mlflow": {Raw: []byte(`"not-an-object"`)},
		}
		_, err := crdPluginsInputToProto(input)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid plugins_input entry")
	})
}
