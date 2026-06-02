// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/v2/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestResolveArtifactBucketConfig_UsesPathLookupForDefaultMinioRoot(t *testing.T) {
	ctx := context.Background()
	clientSet := fake.NewSimpleClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kfp-launcher",
			Namespace: "kubeflow",
		},
		Data: map[string]string{
			"defaultPipelineRoot": "minio://mlpipeline/v2/artifacts",
		},
	})

	launcherConfig, err := config.FetchLauncherConfigMap(ctx, clientSet, "kubeflow")
	require.NoError(t, err)

	bucketConfig, sessionLookupPath, err := resolveArtifactBucketConfig(
		launcherConfig,
		"minio://mlpipeline/v2/artifacts/run-1/system-container/executor-logs",
	)
	require.NoError(t, err)
	require.Equal(
		t,
		"minio://mlpipeline/v2/artifacts/run-1/system-container/executor-logs",
		sessionLookupPath,
	)
	assert.Equal(
		t,
		"minio://mlpipeline?prefix=v2/artifacts/run-1/system-container/executor-logs/",
		bucketConfig.BucketURL(),
	)

	sessionInfo, err := launcherConfig.GetStoreSessionInfo(sessionLookupPath)
	require.NoError(t, err)
	assert.Equal(t, "minio", sessionInfo.Provider)
	assert.Equal(t, "false", sessionInfo.Params["fromEnv"])
	assert.Equal(t, "minio", sessionInfo.Params["region"])
}

func TestResolveArtifactBucketConfig_PreservesExplicitProviderQueryString(t *testing.T) {
	ctx := context.Background()
	clientSet := fake.NewSimpleClientset(&corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "kfp-launcher",
			Namespace: "kubeflow",
		},
		Data: map[string]string{
			"defaultPipelineRoot": "minio://mlpipeline/v2/artifacts?endpoint=seaweedfs.kubeflow:9000&region=minio&disableSSL=true",
		},
	})

	launcherConfig, err := config.FetchLauncherConfigMap(ctx, clientSet, "kubeflow")
	require.NoError(t, err)

	_, sessionLookupPath, err := resolveArtifactBucketConfig(
		launcherConfig,
		"minio://mlpipeline/v2/artifacts/run-1/system-container/executor-logs",
	)
	require.NoError(t, err)
	assert.Equal(
		t,
		"minio://mlpipeline/v2/artifacts/run-1/system-container/executor-logs?endpoint=seaweedfs.kubeflow:9000&region=minio&disableSSL=true",
		sessionLookupPath,
	)
	assert.NotContains(t, sessionLookupPath, "prefix=")

	sessionInfo, err := launcherConfig.GetStoreSessionInfo(sessionLookupPath)
	require.NoError(t, err)
	assert.Equal(t, "minio", sessionInfo.Provider)
	assert.Equal(t, "true", sessionInfo.Params["fromEnv"])
}
