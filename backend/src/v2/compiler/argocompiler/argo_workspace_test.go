// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocompiler_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/v2/compiler/argocompiler"
	"google.golang.org/protobuf/types/known/structpb"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	k8smeta "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetWorkspacePVC(t *testing.T) {
	tests := []struct {
		name        string
		workspace   *pipelinespec.WorkspaceConfig
		opts        *argocompiler.Options
		expectedPVC k8score.PersistentVolumeClaim
		expectError bool
	}{
		{
			name: "workspace with size specified",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "5Gi",
			},
			opts: nil,
			expectedPVC: k8score.PersistentVolumeClaim{
				ObjectMeta: k8smeta.ObjectMeta{
					Name: "kfp-workspace",
				},
				Spec: k8score.PersistentVolumeClaimSpec{
					Resources: k8score.VolumeResourceRequirements{
						Requests: map[k8score.ResourceName]resource.Quantity{
							k8score.ResourceStorage: resource.MustParse("5Gi"),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "workspace with default size from options",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "", // empty size
			},
			opts: &argocompiler.Options{
				DefaultWorkspaceSize: "20Gi",
			},
			expectedPVC: k8score.PersistentVolumeClaim{
				ObjectMeta: k8smeta.ObjectMeta{
					Name: "kfp-workspace",
				},
				Spec: k8score.PersistentVolumeClaimSpec{
					Resources: k8score.VolumeResourceRequirements{
						Requests: map[k8score.ResourceName]resource.Quantity{
							k8score.ResourceStorage: resource.MustParse("20Gi"),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "workspace with fallback size",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "", // empty size
			},
			opts: nil, // no options
			expectedPVC: k8score.PersistentVolumeClaim{
				ObjectMeta: k8smeta.ObjectMeta{
					Name: "kfp-workspace",
				},
				Spec: k8score.PersistentVolumeClaimSpec{
					Resources: k8score.VolumeResourceRequirements{
						Requests: map[k8score.ResourceName]resource.Quantity{
							k8score.ResourceStorage: resource.MustParse("10Gi"),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "workspace with default PVC spec from options",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "15Gi",
			},
			opts: &argocompiler.Options{
				DefaultWorkspace: &k8score.PersistentVolumeClaimSpec{
					AccessModes: []k8score.PersistentVolumeAccessMode{
						k8score.ReadWriteOnce,
					},
					StorageClassName: stringPtr("fast-ssd"),
					Resources: k8score.VolumeResourceRequirements{
						Requests: map[k8score.ResourceName]resource.Quantity{
							k8score.ResourceStorage: resource.MustParse("1Gi"), // will be overridden
						},
					},
				},
			},
			expectedPVC: k8score.PersistentVolumeClaim{
				ObjectMeta: k8smeta.ObjectMeta{
					Name: "kfp-workspace",
				},
				Spec: k8score.PersistentVolumeClaimSpec{
					AccessModes: []k8score.PersistentVolumeAccessMode{
						k8score.ReadWriteOnce,
					},
					StorageClassName: stringPtr("fast-ssd"),
					Resources: k8score.VolumeResourceRequirements{
						Requests: map[k8score.ResourceName]resource.Quantity{
							k8score.ResourceStorage: resource.MustParse("15Gi"),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "workspace with Kubernetes PVC spec patch",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "25Gi",
				Kubernetes: &pipelinespec.KubernetesWorkspaceConfig{
					PvcSpecPatch: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"accessModes": structpb.NewListValue(&structpb.ListValue{
								Values: []*structpb.Value{
									structpb.NewStringValue("ReadWriteMany"),
								},
							}),
							"storageClassName": structpb.NewStringValue("nfs-storage"),
						},
					},
				},
			},
			opts: &argocompiler.Options{
				DefaultWorkspace: &k8score.PersistentVolumeClaimSpec{
					AccessModes: []k8score.PersistentVolumeAccessMode{
						k8score.ReadWriteOnce,
					},
					StorageClassName: stringPtr("default-storage"),
				},
			},
			expectedPVC: k8score.PersistentVolumeClaim{
				ObjectMeta: k8smeta.ObjectMeta{
					Name: "kfp-workspace",
				},
				Spec: k8score.PersistentVolumeClaimSpec{
					AccessModes: []k8score.PersistentVolumeAccessMode{
						k8score.ReadWriteMany,
					},
					StorageClassName: stringPtr("nfs-storage"),
					Resources: k8score.VolumeResourceRequirements{
						Requests: map[k8score.ResourceName]resource.Quantity{
							k8score.ResourceStorage: resource.MustParse("25Gi"),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "workspace with invalid size",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "invalid-size",
			},
			opts:        nil,
			expectedPVC: k8score.PersistentVolumeClaim{},
			expectError: true,
		},
		{
			name: "workspace with invalid PVC spec patch",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "10Gi",
				Kubernetes: &pipelinespec.KubernetesWorkspaceConfig{
					PvcSpecPatch: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"invalidField": structpb.NewStringValue("invalid"),
						},
					},
				},
			},
			opts:        nil,
			expectedPVC: k8score.PersistentVolumeClaim{},
			expectError: true,
		},
		{
			name: "workspace with complex PVC spec patch",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "50Gi",
				Kubernetes: &pipelinespec.KubernetesWorkspaceConfig{
					PvcSpecPatch: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"accessModes": structpb.NewListValue(&structpb.ListValue{
								Values: []*structpb.Value{
									structpb.NewStringValue("ReadWriteOnce"),
									structpb.NewStringValue("ReadOnlyMany"),
								},
							}),
							"storageClassName": structpb.NewStringValue("premium-ssd"),
						},
					},
				},
			},
			opts: &argocompiler.Options{
				DefaultWorkspace: &k8score.PersistentVolumeClaimSpec{
					AccessModes: []k8score.PersistentVolumeAccessMode{
						k8score.ReadWriteOnce,
					},
					StorageClassName: stringPtr("default"),
				},
			},
			expectedPVC: k8score.PersistentVolumeClaim{
				ObjectMeta: k8smeta.ObjectMeta{
					Name: "kfp-workspace",
				},
				Spec: k8score.PersistentVolumeClaimSpec{
					AccessModes: []k8score.PersistentVolumeAccessMode{
						k8score.ReadWriteOnce,
						k8score.ReadOnlyMany,
					},
					StorageClassName: stringPtr("premium-ssd"),
					Resources: k8score.VolumeResourceRequirements{
						Requests: map[k8score.ResourceName]resource.Quantity{
							k8score.ResourceStorage: resource.MustParse("50Gi"),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "workspace with nil Kubernetes config",
			workspace: &pipelinespec.WorkspaceConfig{
				Size:       "30Gi",
				Kubernetes: nil,
			},
			opts: nil,
			expectedPVC: k8score.PersistentVolumeClaim{
				ObjectMeta: k8smeta.ObjectMeta{
					Name: "kfp-workspace",
				},
				Spec: k8score.PersistentVolumeClaimSpec{
					Resources: k8score.VolumeResourceRequirements{
						Requests: map[k8score.ResourceName]resource.Quantity{
							k8score.ResourceStorage: resource.MustParse("30Gi"),
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "workspace with empty Kubernetes config",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "40Gi",
				Kubernetes: &pipelinespec.KubernetesWorkspaceConfig{
					PvcSpecPatch: nil,
				},
			},
			opts: nil,
			expectedPVC: k8score.PersistentVolumeClaim{
				ObjectMeta: k8smeta.ObjectMeta{
					Name: "kfp-workspace",
				},
				Spec: k8score.PersistentVolumeClaimSpec{
					Resources: k8score.VolumeResourceRequirements{
						Requests: map[k8score.ResourceName]resource.Quantity{
							k8score.ResourceStorage: resource.MustParse("40Gi"),
						},
					},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := argocompiler.GetWorkspacePVC(tt.workspace, tt.opts)
			if (err != nil) != tt.expectError {
				t.Errorf("GetWorkspacePVC() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if !tt.expectError {
				if diff := cmp.Diff(tt.expectedPVC, got); diff != "" {
					t.Errorf("GetWorkspacePVC() mismatch (-expected +got):\n%s", diff)
				}
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}

func TestGetWorkspacePVC_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		workspace   *pipelinespec.WorkspaceConfig
		opts        *argocompiler.Options
		expectError bool
		errMsg      string
	}{
		{
			name: "workspace with very large size",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "1000Ti",
			},
			opts:        nil,
			expectError: false, // should be valid
		},
		{
			name: "workspace with decimal size",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "1.5Gi",
			},
			opts:        nil,
			expectError: false, // should be valid
		},
		{
			name: "workspace with invalid size format",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "10GB", // should be "10Gi"
			},
			opts:        nil,
			expectError: true,
		},
		{
			name: "workspace with negative size",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "-10Gi",
			},
			opts:        nil,
			expectError: true,
		},
		{
			name: "workspace with malformed PVC spec patch",
			workspace: &pipelinespec.WorkspaceConfig{
				Size: "10Gi",
				Kubernetes: &pipelinespec.KubernetesWorkspaceConfig{
					PvcSpecPatch: &structpb.Struct{
						Fields: map[string]*structpb.Value{
							"resources": structpb.NewStructValue(&structpb.Struct{
								Fields: map[string]*structpb.Value{
									"requests": structpb.NewStringValue("invalid"), // should be a map
								},
							}),
						},
					},
				},
			},
			opts:        nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := argocompiler.GetWorkspacePVC(tt.workspace, tt.opts)
			if (err != nil) != tt.expectError {
				t.Errorf("getWorkspacePVC() error = %v, expectError %v", err, tt.expectError)
				return
			}
			if tt.expectError && tt.errMsg != "" && err != nil {
				if err.Error() != tt.errMsg {
					t.Errorf("getWorkspacePVC() error message = %v, expect %v", err.Error(), tt.errMsg)
				}
			}
		})
	}
}

// TestGetWorkspacePVC_Integration tests integration scenarios
func TestGetWorkspacePVC_Integration(t *testing.T) {
	t.Run("complete workspace configuration", func(t *testing.T) {
		workspace := &pipelinespec.WorkspaceConfig{
			Size: "100Gi",
			Kubernetes: &pipelinespec.KubernetesWorkspaceConfig{
				PvcSpecPatch: &structpb.Struct{
					Fields: map[string]*structpb.Value{
						"accessModes": structpb.NewListValue(&structpb.ListValue{
							Values: []*structpb.Value{
								structpb.NewStringValue("ReadWriteOnce"),
							},
						}),
						"storageClassName": structpb.NewStringValue("gp2"),
					},
				},
			},
		}

		opts := &argocompiler.Options{
			DefaultWorkspaceSize: "50Gi",
			DefaultWorkspace: &k8score.PersistentVolumeClaimSpec{
				AccessModes: []k8score.PersistentVolumeAccessMode{
					k8score.ReadWriteOnce,
				},
				StorageClassName: stringPtr("default"),
				Resources: k8score.VolumeResourceRequirements{
					Requests: map[k8score.ResourceName]resource.Quantity{
						k8score.ResourceStorage: resource.MustParse("10Gi"),
					},
				},
			},
		}

		pvc, err := argocompiler.GetWorkspacePVC(workspace, opts)
		if err != nil {
			t.Fatalf("getWorkspacePVC() unexpected error: %v", err)
		}

		// Verify the result
		expected := k8score.PersistentVolumeClaim{
			ObjectMeta: k8smeta.ObjectMeta{
				Name: "kfp-workspace",
			},
			Spec: k8score.PersistentVolumeClaimSpec{
				AccessModes: []k8score.PersistentVolumeAccessMode{
					k8score.ReadWriteOnce,
				},
				StorageClassName: stringPtr("gp2"),
				Resources: k8score.VolumeResourceRequirements{
					Requests: map[k8score.ResourceName]resource.Quantity{
						k8score.ResourceStorage: resource.MustParse("100Gi"),
					},
				},
			},
		}

		if diff := cmp.Diff(expected, pvc); diff != "" {
			t.Errorf("getWorkspacePVC() mismatch (-want +got):\n%s", diff)
		}
	})
}
