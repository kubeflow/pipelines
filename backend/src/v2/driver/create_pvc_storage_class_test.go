// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package driver

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/structpb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestContainer_CreatePVCStorageClassSemantics(t *testing.T) {
	tests := []struct {
		name string
		// storageClassInput, when non-nil, is injected into the task spec as
		// the storage_class_name input constant.
		storageClassInput *string
		expectedClassName *string
	}{
		{
			name:              "Omitted input leaves StorageClassName nil for Kubernetes defaulting",
			storageClassInput: nil,
			expectedClassName: nil,
		},
		{
			name:              "Explicit class name is preserved",
			storageClassInput: new("fast-ssd"),
			expectedClassName: new("fast-ssd"),
		},
		{
			name:              "Explicit empty string is preserved as an opt-out of default selection",
			storageClassInput: new(""),
			expectedClassName: new(""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/nested_k8s_pvc_outputs.yaml")

			_, nestedTask := tc.RunDagDriver("nested", tc.RootTask)
			err := tc.Push("createpvc")
			if err != nil {
				t.Fatalf("failed to push scope path: %v", err)
			}
			defer func() {
				_, ok := tc.Pop()
				if !ok {
					t.Fatal("failed to pop scope path")
				}
			}()

			taskSpec := tc.GetLast().GetTaskSpec()
			if tt.storageClassInput != nil {
				taskSpec.GetInputs().GetParameters()["storage_class_name"] =
					&pipelinespec.TaskInputsSpec_InputParameterSpec{
						Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
							RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
								Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
									Constant: structpb.NewStringValue(*tt.storageClassInput),
								},
							},
						},
					}
			}
			kubernetesExecutorConfig, err := util.LoadKubernetesExecutorConfig(tc.GetLast().GetComponentSpec(), tc.PlatformSpec)
			if err != nil {
				t.Fatalf("failed to load kubernetes executor config: %v", err)
			}
			opts := tc.setupContainerOptions(nestedTask, taskSpec, kubernetesExecutorConfig)

			execution, err := Container(context.Background(), opts, tc.ClientManager)
			if err != nil {
				t.Fatalf("expected createpvc execution to succeed, got error: %v", err)
			}
			if execution == nil || execution.TaskID == "" {
				t.Fatal("expected successful createpvc execution to return a task id")
			}

			pvc, err := tc.ClientManager.K8sClient().CoreV1().PersistentVolumeClaims(opts.Namespace).Get(
				context.Background(), "nested-pvc", metav1.GetOptions{},
			)
			if err != nil {
				t.Fatalf("failed to get created PVC: %v", err)
			}
			assert.Equal(t, tt.expectedClassName, pvc.Spec.StorageClassName)
		})
	}
}

// TestContainer_CreatePVCSdkCompiledOmittedStorageClass verifies that a
// pipeline compiled by the SDK, where the CreatePVC component declares
// storage_class_name as optional with no default value, results in a PVC
// with a nil StorageClassName so Kubernetes applies the cluster default.
// This guards the SDK-to-driver integration: the parameter resolver must
// not inject a value for the optional input.
func TestContainer_CreatePVCSdkCompiledOmittedStorageClass(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/sdk_create_pvc_default_storage_class.yaml")

	err := tc.Push("createpvc")
	if err != nil {
		t.Fatalf("failed to push scope path: %v", err)
	}
	defer func() {
		_, ok := tc.Pop()
		if !ok {
			t.Fatal("failed to pop scope path")
		}
	}()

	taskSpec := tc.GetLast().GetTaskSpec()
	kubernetesExecutorConfig, err := util.LoadKubernetesExecutorConfig(tc.GetLast().GetComponentSpec(), tc.PlatformSpec)
	if err != nil {
		t.Fatalf("failed to load kubernetes executor config: %v", err)
	}
	opts := tc.setupContainerOptions(tc.RootTask, taskSpec, kubernetesExecutorConfig)

	execution, err := Container(context.Background(), opts, tc.ClientManager)
	if err != nil {
		t.Fatalf("expected createpvc execution to succeed, got error: %v", err)
	}
	if execution == nil || execution.TaskID == "" {
		t.Fatal("expected successful createpvc execution to return a task id")
	}

	pvc, err := tc.ClientManager.K8sClient().CoreV1().PersistentVolumeClaims(opts.Namespace).Get(
		context.Background(), "sdk-pvc", metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("failed to get created PVC: %v", err)
	}
	assert.Nil(t, pvc.Spec.StorageClassName)
}

// TestContainer_CreatePVCLegacyDefaultExplicitEmptyStorageClass verifies the
// discriminator between provenance and value: when the component carries an
// empty-string defaultValue but the task supplies an explicit empty
// storage_class_name, the value is a task input, not an injected default, and must
// remain an opt-out of default storage-class selection.
func TestContainer_CreatePVCLegacyDefaultExplicitEmptyStorageClass(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/sdk_create_pvc_legacy_default_storage_class.yaml")

	err := tc.Push("createpvc")
	if err != nil {
		t.Fatalf("failed to push scope path: %v", err)
	}
	defer func() {
		_, ok := tc.Pop()
		if !ok {
			t.Fatal("failed to pop scope path")
		}
	}()

	taskSpec := tc.GetLast().GetTaskSpec()
	taskSpec.GetInputs().GetParameters()["storage_class_name"] =
		&pipelinespec.TaskInputsSpec_InputParameterSpec{
			Kind: &pipelinespec.TaskInputsSpec_InputParameterSpec_RuntimeValue{
				RuntimeValue: &pipelinespec.ValueOrRuntimeParameter{
					Value: &pipelinespec.ValueOrRuntimeParameter_Constant{
						Constant: structpb.NewStringValue(""),
					},
				},
			},
		}
	kubernetesExecutorConfig, err := util.LoadKubernetesExecutorConfig(tc.GetLast().GetComponentSpec(), tc.PlatformSpec)
	if err != nil {
		t.Fatalf("failed to load kubernetes executor config: %v", err)
	}
	opts := tc.setupContainerOptions(tc.RootTask, taskSpec, kubernetesExecutorConfig)

	execution, err := Container(context.Background(), opts, tc.ClientManager)
	if err != nil {
		t.Fatalf("expected createpvc execution to succeed, got error: %v", err)
	}
	if execution == nil || execution.TaskID == "" {
		t.Fatal("expected successful createpvc execution to return a task id")
	}

	pvc, err := tc.ClientManager.K8sClient().CoreV1().PersistentVolumeClaims(opts.Namespace).Get(
		context.Background(), "sdk-pvc", metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("failed to get created PVC: %v", err)
	}
	if pvc.Spec.StorageClassName == nil {
		t.Fatal("expected task-level empty string to remain an opt-out, got nil StorageClassName")
	}
	assert.Equal(t, "", *pvc.Spec.StorageClassName)
}

// TestContainer_CreatePVCLegacyDefaultStorageClass verifies that pipelines
// compiled with an older SDK, where the CreatePVC component carries an
// empty-string defaultValue for storage_class_name and the parameter
// resolver injects it for omitted arguments, still get a PVC with a nil
// StorageClassName so Kubernetes applies the cluster default. A task-level
// explicit empty string remains an opt-out; only the injected default is
// treated as omitted.
func TestContainer_CreatePVCLegacyDefaultStorageClass(t *testing.T) {
	tc := NewTestContextWithRootExecuted(t, &pipelinespec.PipelineJob_RuntimeConfig{}, "test_data/sdk_create_pvc_legacy_default_storage_class.yaml")

	err := tc.Push("createpvc")
	if err != nil {
		t.Fatalf("failed to push scope path: %v", err)
	}
	defer func() {
		_, ok := tc.Pop()
		if !ok {
			t.Fatal("failed to pop scope path")
		}
	}()

	taskSpec := tc.GetLast().GetTaskSpec()
	kubernetesExecutorConfig, err := util.LoadKubernetesExecutorConfig(tc.GetLast().GetComponentSpec(), tc.PlatformSpec)
	if err != nil {
		t.Fatalf("failed to load kubernetes executor config: %v", err)
	}
	opts := tc.setupContainerOptions(tc.RootTask, taskSpec, kubernetesExecutorConfig)

	execution, err := Container(context.Background(), opts, tc.ClientManager)
	if err != nil {
		t.Fatalf("expected createpvc execution to succeed, got error: %v", err)
	}
	if execution == nil || execution.TaskID == "" {
		t.Fatal("expected successful createpvc execution to return a task id")
	}

	pvc, err := tc.ClientManager.K8sClient().CoreV1().PersistentVolumeClaims(opts.Namespace).Get(
		context.Background(), "sdk-pvc", metav1.GetOptions{},
	)
	if err != nil {
		t.Fatalf("failed to get created PVC: %v", err)
	}
	assert.Nil(t, pvc.Spec.StorageClassName)
}
