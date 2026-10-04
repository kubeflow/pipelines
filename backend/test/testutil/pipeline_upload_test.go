// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutil

import (
	"os"
	"path/filepath"
	"testing"

	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
)

type recordingPipelineUploadClient struct {
	api_server.PipelineUploadInterface
	params *upload_params.UploadPipelineParams
}

func (c *recordingPipelineUploadClient) UploadFile(_ string, params *upload_params.UploadPipelineParams) (*model.V2beta1Pipeline, error) {
	c.params = params
	return &model.V2beta1Pipeline{PipelineID: "test-pipeline"}, nil
}

func TestUploadPipelineUsesTenantNamespace(t *testing.T) {
	gomega.RegisterTestingT(t)
	for _, tc := range []struct {
		name          string
		multiUser     bool
		kubeflow      bool
		token         string
		wantNamespace bool
		kubernetes    bool
	}{
		{name: "single user"},
		{name: "multi user", multiUser: true, wantNamespace: true},
		{name: "kubeflow", kubeflow: true, wantNamespace: true},
		{name: "explicit token", token: "test-token", wantNamespace: true},
		{name: "kubernetes single user", kubernetes: true},
		{name: "kubernetes multi user", kubernetes: true, multiUser: true},
		{name: "kubernetes kubeflow", kubernetes: true, kubeflow: true},
		{name: "kubernetes explicit token", kubernetes: true, token: "test-token"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			previousMultiUser, previousKubeflow := *config.MultiUserMode, *config.KubeflowMode
			previousKubernetes := *config.UploadPipelinesWithKubernetes
			previousToken, previousNamespace, previousImage := *config.AuthToken, *config.UserNamespace, *config.BaseImage
			t.Cleanup(func() {
				*config.UploadPipelinesWithKubernetes = previousKubernetes
				*config.MultiUserMode, *config.KubeflowMode = previousMultiUser, previousKubeflow
				*config.AuthToken, *config.UserNamespace, *config.BaseImage = previousToken, previousNamespace, previousImage
			})
			*config.MultiUserMode, *config.KubeflowMode = tc.multiUser, tc.kubeflow
			*config.UploadPipelinesWithKubernetes = tc.kubernetes
			*config.AuthToken, *config.UserNamespace, *config.BaseImage = tc.token, "test-tenant", ""
			path := filepath.Join(t.TempDir(), "pipeline.yaml")
			require.NoError(t, os.WriteFile(path, []byte("pipelineInfo:\n  name: test-pipeline\n"), 0600))
			client := &recordingPipelineUploadClient{}
			name := "test-pipeline"
			_, err := UploadPipeline(client, path, &name, nil)
			require.NoError(t, err)
			require.NotNil(t, client.params)
			if tc.wantNamespace {
				require.NotNil(t, client.params.Namespace)
				require.Equal(t, "test-tenant", *client.params.Namespace)
			} else {
				require.Nil(t, client.params.Namespace)
			}
		})
	}
}
