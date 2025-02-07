// Copyright 2024 The Kubeflow Authors
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

package server

import (
	"context"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strings"
	"testing"
	"time"
)

func TestGetArtifacts(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewArtifactServer(manager, &ArtifactServerOptions{CollectMetrics: false})
	ctx := context.Background()

	tests := []struct {
		name             string
		artifactRequest  *apiv2beta1.GetArtifactRequest
		expectedArtifact *apiv2beta1.Artifact
		wantErr          bool
		errMsg           string
	}{
		{
			name: "Fetch artifact 0",
			artifactRequest: &apiv2beta1.GetArtifactRequest{
				ArtifactId: "0",
				View:       apiv2beta1.GetArtifactRequest_DOWNLOAD,
			},
			expectedArtifact: &apiv2beta1.Artifact{
				ArtifactId:      "0",
				StorageProvider: "s3",
				StoragePath:     "pipeline/some-pipeline-id/task/key0",
				Uri:             "s3://test-bucket/pipeline/some-pipeline-id/task/key0",
				DownloadUrl:     "dummy-signed-url", // defined in object_store_fake
				Namespace:       "test-namespace",
				ArtifactType:    "1",
				ArtifactSize:    123,

				CreatedAt:     timestamppb.New(time.Unix(1, 0)),
				LastUpdatedAt: timestamppb.New(time.Unix(1, 0)),
			},
		},
		{
			name: "Fetch artifact 1",
			artifactRequest: &apiv2beta1.GetArtifactRequest{
				ArtifactId: "1",
				View:       apiv2beta1.GetArtifactRequest_BASIC,
			},
			expectedArtifact: &apiv2beta1.Artifact{
				ArtifactId:      "1",
				StorageProvider: "s3",
				StoragePath:     "pipeline/some-pipeline-id/task/key1",
				Uri:             "s3://test-bucket/pipeline/some-pipeline-id/task/key1",
				DownloadUrl:     "",
				Namespace:       "test-namespace",
				ArtifactType:    "1",
				ArtifactSize:    123,
				CreatedAt:       timestamppb.New(time.Unix(2, 0)),
				LastUpdatedAt:   timestamppb.New(time.Unix(2, 0)),
			},
		},
		{
			name: "Fetch artifact 1 - View unspecified",
			artifactRequest: &apiv2beta1.GetArtifactRequest{
				ArtifactId: "1",
				View:       apiv2beta1.GetArtifactRequest_ARTIFACT_VIEW_UNSPECIFIED,
			},
			expectedArtifact: &apiv2beta1.Artifact{
				ArtifactId:      "1",
				StorageProvider: "s3",
				StoragePath:     "pipeline/some-pipeline-id/task/key1",
				Uri:             "s3://test-bucket/pipeline/some-pipeline-id/task/key1",
				DownloadUrl:     "",
				Namespace:       "test-namespace",
				ArtifactType:    "1",
				ArtifactSize:    123,
				CreatedAt:       timestamppb.New(time.Unix(2, 0)),
				LastUpdatedAt:   timestamppb.New(time.Unix(2, 0)),
			},
		},
		{
			name: "Error on non existent artifact",
			artifactRequest: &apiv2beta1.GetArtifactRequest{
				ArtifactId: "999999",
				View:       apiv2beta1.GetArtifactRequest_BASIC,
			},
			expectedArtifact: &apiv2beta1.Artifact{},
			wantErr:          true,
			errMsg:           "ResourceNotFoundError: failed to find artifact",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact, err1 := server.GetArtifact(ctx, tt.artifactRequest)
			if !tt.wantErr {
				require.Nil(t, err1)
				require.Equal(t, tt.expectedArtifact, artifact)
				require.NotNil(t, artifact)
			} else {
				require.NotNil(t, err1)
				if tt.errMsg != "" {
					require.True(t, strings.HasPrefix(err1.Error(), tt.errMsg))
				}
			}
		})
	}
}

func TestListArtifacts(t *testing.T) {
	clients, manager, _ := initWithExperiment(t)
	defer clients.Close()
	server := NewArtifactServer(manager, &ArtifactServerOptions{CollectMetrics: false})
	ctx := context.Background()

	artifact0 := &apiv2beta1.Artifact{
		ArtifactId:      "0",
		StorageProvider: "s3",
		StoragePath:     "pipeline/some-pipeline-id/task/key0",
		Uri:             "s3://test-bucket/pipeline/some-pipeline-id/task/key0",
		DownloadUrl:     "", // defined in object_store_fake
		Namespace:       "test-namespace",
		ArtifactType:    "1",
		ArtifactSize:    123,
		CreatedAt:       timestamppb.New(time.Unix(1, 0)),
		LastUpdatedAt:   timestamppb.New(time.Unix(1, 0)),
	}

	artifact1 := &apiv2beta1.Artifact{
		ArtifactId:      "1",
		StorageProvider: "s3",
		StoragePath:     "pipeline/some-pipeline-id/task/key1",
		Uri:             "s3://test-bucket/pipeline/some-pipeline-id/task/key1",
		DownloadUrl:     "",
		Namespace:       "test-namespace",
		ArtifactType:    "1",
		ArtifactSize:    123,

		CreatedAt:     timestamppb.New(time.Unix(2, 0)),
		LastUpdatedAt: timestamppb.New(time.Unix(2, 0)),
	}

	tests := []struct {
		name              string
		artifactRequest   *apiv2beta1.ListArtifactRequest
		expectedArtifacts []*apiv2beta1.Artifact
		wantErr           bool
		errMsg            string
	}{
		{
			name: "List artifacts",
			artifactRequest: &apiv2beta1.ListArtifactRequest{
				MaxResultSize: 2,
				// note fake mlmd client does not implement context filtering by namespace
				Namespace: "test-namespace",
				OrderBy:   "asc",
			},
			expectedArtifacts: []*apiv2beta1.Artifact{artifact0, artifact1},
			wantErr:           false,
		},
		{
			name: "List artifacts with different result size",
			artifactRequest: &apiv2beta1.ListArtifactRequest{
				MaxResultSize: 1,
				Namespace:     "test-namespace",
				OrderBy:       "asc",
			},
			expectedArtifacts: []*apiv2beta1.Artifact{artifact0},
			wantErr:           false,
		},
		{
			name: "Sort artifacts by desc",
			artifactRequest: &apiv2beta1.ListArtifactRequest{
				MaxResultSize: 2,
				Namespace:     "test-namespace",
				OrderBy:       "desc",
			},
			expectedArtifacts: []*apiv2beta1.Artifact{artifact1, artifact0},
			wantErr:           false,
		},
		{
			name: "Namespace required",
			artifactRequest: &apiv2beta1.ListArtifactRequest{
				MaxResultSize: 2,
				Namespace:     "",
				OrderBy:       "desc",
			},
			expectedArtifacts: []*apiv2beta1.Artifact{artifact1, artifact0},
			wantErr:           true,
			errMsg:            "Missing required namespace parameter.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err1 := server.ListArtifacts(ctx, tt.artifactRequest)
			if !tt.wantErr {
				require.NotNil(t, resp)
				require.Equal(t, len(tt.expectedArtifacts), len(resp.Artifacts))
				for i := 0; i < len(tt.expectedArtifacts); i += 1 {
					require.Equal(t, tt.expectedArtifacts[i], resp.Artifacts[i])
				}
				require.NotNil(t, resp.Artifacts)
				require.Nil(t, err1)
			} else {
				require.NotNil(t, err1)
				if tt.errMsg != "" {
					require.True(t, strings.HasPrefix(err1.Error(), tt.errMsg))
				}
			}
		})
	}
}
