// Copyright 2018-2023 The Kubeflow Authors
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

package api_server

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_model"
)

const (
	FileForDefaultTest     = "./samples/parameters.yaml"
	FileForClientErrorTest = "./samples/hello-world.yaml"

	ClientErrorString  = "Error with client"
	InvalidFakeRequest = "Invalid fake request, don't know how to handle '%s' in the fake client."
)

func getDefaultUploadedPipeline() *model.APIPipeline {
	return &model.APIPipeline{
		ID:          "500",
		CreatedAt:   strfmt.NewDateTime(),
		Name:        "PIPELINE_NAME",
		Description: "PIPELINE_DESCRIPTION",
		Parameters: []*model.APIParameter{&model.APIParameter{
			Name:  "PARAM_NAME",
			Value: "PARAM_VALUE",
		}},
	}
}

type PipelineUploadClientFake struct{}

func NewPipelineUploadClientFake() *PipelineUploadClientFake {
	return &PipelineUploadClientFake{}
}

func (c *PipelineUploadClientFake) UploadFile(filePath string,
	parameters *params.UploadPipelineParams) (*model.APIPipeline, error) {
	switch filePath {
	case FileForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultUploadedPipeline(), nil
	}
}

// TODO(jingzhang36): add UploadPipelineVersion fake to be used in integration test
// after go_http_client and go_client are auto-generated from UploadPipelineVersion in PipelineUploadServer
