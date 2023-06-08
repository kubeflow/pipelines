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
	"context"
	"fmt"
	"os"

	"github.com/go-openapi/runtime"
	rt "github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	apiclient "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_client"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_model"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	pipelineUploadFieldName      = "uploadfile"
	pipelineUploadPath           = "pipelines/upload"
	pipelineUploadServerBasePath = "/api/v1/namespaces/%s/services/ml-pipeline:8888/proxy/apis/v1beta1/%s"
	pipelineUploadContentTypeKey = "Content-Type"
	pipelineVersionUploadPath    = "pipelines/upload_version"
)

type PipelineUploadInterface interface {
	UploadFile(filePath string, parameters *params.UploadPipelineParams) (*model.APIPipeline, error)
}

type PipelineUploadClient struct {
	apiClient      *apiclient.PipelineUpload
	authInfoWriter rt.ClientAuthInfoWriter
}

func NewPipelineUploadClient(clientConfig clientcmd.ClientConfig, debug bool) (
	*PipelineUploadClient, error) {

	runtime, err := api_server.NewHTTPRuntime(clientConfig, debug)
	if err != nil {
		return nil, fmt.Errorf("Error occurred when creating pipeline upload client: %w", err)
	}

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating upload client
	return &PipelineUploadClient{
		apiClient: apiClient,
	}, nil
}

func NewKubeflowInClusterPipelineUploadClient(namespace string, debug bool) (
	*PipelineUploadClient, error) {

	runtime := api_server.NewKubeflowInClusterHTTPRuntime(namespace, debug)

	apiClient := apiclient.New(runtime, strfmt.Default)

	// Creating upload client
	return &PipelineUploadClient{
		apiClient:      apiClient,
		authInfoWriter: api_server.SATokenVolumeProjectionAuth,
	}, nil
}

func (c *PipelineUploadClient) UploadFile(filePath string, parameters *params.UploadPipelineParams) (
	*model.APIPipeline, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, util.NewUserErrorWithSingleMessage(err,
			fmt.Sprintf("Failed to open file '%s'", filePath))
	}
	defer file.Close()

	parameters.Uploadfile = runtime.NamedReader(filePath, file)
	return c.Upload(parameters)
}

func (c *PipelineUploadClient) Upload(parameters *params.UploadPipelineParams) (*model.APIPipeline,
	error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineUploadService.UploadPipeline(parameters, c.authInfoWriter)

	if err != nil {
		if defaultError, ok := err.(*params.UploadPipelineDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to upload pipeline. Params: '%v'", parameters),
			fmt.Sprintf("Failed to upload pipeline"))
	}

	return response.Payload, nil
}

// UploadPipelineVersion uploads pipeline version from local file.
func (c *PipelineUploadClient) UploadPipelineVersion(filePath string, parameters *params.UploadPipelineVersionParams) (*model.APIPipelineVersion,
	error) {
	// Get file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, util.NewUserErrorWithSingleMessage(err,
			fmt.Sprintf("Failed to open file '%s'", filePath))
	}
	defer file.Close()
	parameters.Uploadfile = runtime.NamedReader(filePath, file)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	// Make service call
	parameters.Context = ctx
	response, err := c.apiClient.PipelineUploadService.UploadPipelineVersion(parameters, c.authInfoWriter)

	if err != nil {
		if defaultError, ok := err.(*params.UploadPipelineVersionDefault); ok {
			err = api_server.CreateErrorFromAPIStatus(defaultError.Payload.Error, defaultError.Payload.Code)
		} else {
			err = api_server.CreateErrorCouldNotRecoverAPIStatus(err)
		}

		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to upload pipeline version. Params: '%v'", parameters),
			fmt.Sprintf("Failed to upload pipeline version"))
	}

	return response.Payload, nil
}
