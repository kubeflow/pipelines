// Copyright 2025 The Kubeflow Authors
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

package api_server_v2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
	params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	apimodel "github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/apiserver/server"
	"github.com/kubeflow/pipelines/backend/src/common/client/api_server"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	k8sapi "github.com/kubeflow/pipelines/backend/src/crd/kubernetes/v2beta1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/tools/clientcmd"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme *k8sruntime.Scheme

func init() {
	scheme = k8sruntime.NewScheme()
	err := k8sapi.AddToScheme(scheme)
	if err != nil {
		// Panic is okay here because it means there's a code issue and so the package shouldn't initialize.
		panic(fmt.Sprintf("Failed to initialize the Kubernetes API scheme: %v", err))
	}
}

type PipelineUploadClientKubernetes struct {
	ctrlClient ctrlclient.Client
	namespace  string
}

func deriveNameDisplayAndDescription(providedName, providedDisplayName, providedDescription *string, defaultName string) (string, string, string) {
	name := defaultName
	if providedName != nil {
		name = *providedName
	} else if providedDisplayName != nil {
		name = *providedDisplayName
	}

	displayName := name
	if providedDisplayName != nil {
		displayName = *providedDisplayName
	}

	var description string
	if providedDescription != nil {
		description = *providedDescription
	}

	return name, displayName, description
}

func NewPipelineUploadClientKubernetes(clientConfig clientcmd.ClientConfig, namespace string) (
	PipelineUploadInterface, error,
) {
	restConfig, err := clientConfig.ClientConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get the rest config: %w", err)
	}

	ctrlClient, err := ctrlclient.New(restConfig, ctrlclient.Options{Scheme: scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the controller-runtime client: %w", err)
	}

	return &PipelineUploadClientKubernetes{
		ctrlClient: ctrlClient,
		namespace:  namespace,
	}, nil
}

func (c *PipelineUploadClientKubernetes) UploadFile(filePath string, parameters *params.UploadPipelineParams) (
	*model.V2beta1Pipeline, error,
) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, util.NewUserErrorWithSingleMessage(err,
			fmt.Sprintf("Failed to open file '%s'", filePath))
	}
	defer file.Close()

	processedFile, err := server.ReadPipelineFile(path.Base(filePath), file, common.MaxFileLength)
	if err != nil {
		return nil, util.NewUserErrorWithSingleMessage(err, "Failed to read pipeline spec file")
	}

	parameters.Uploadfile = runtime.NamedReader(path.Base(filePath), io.NopCloser(bytes.NewReader(processedFile)))
	return c.Upload(parameters)
}

func (c *PipelineUploadClientKubernetes) Upload(parameters *params.UploadPipelineParams) (*model.V2beta1Pipeline,
	error,
) {
	if parameters.Namespace != nil && *parameters.Namespace != c.namespace {
		return nil, util.NewUserError(errors.New("namespace cannot be set as an upload parameter"),
			"Namespace cannot be set as an upload parameter in the Kubernetes client",
			"Namespace cannot be set as an upload parameter")
	}

	piplineSpec, err := io.ReadAll(parameters.Uploadfile)
	if err != nil {
		return nil, fmt.Errorf("failed to read pipeline spec: %w", err)
	}

	defer parameters.Uploadfile.Close()

	name, displayName, description := deriveNameDisplayAndDescription(
		parameters.Name,
		parameters.DisplayName,
		parameters.Description,
		path.Base(parameters.Uploadfile.Name()),
	)

	pipelineModel := apimodel.Pipeline{
		Name:        name,
		Namespace:   c.namespace,
		DisplayName: displayName,
		Description: apimodel.LargeText(description),
	}
	pipeline := k8sapi.FromPipelineModel(pipelineModel)

	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	err = c.ctrlClient.Create(ctx, &pipeline)
	if err != nil {
		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to upload pipeline. Params: '%v'", parameters),
			"Failed to upload pipeline")
	}

	pipelineVersionModel := apimodel.PipelineVersion{
		Name:         name,
		DisplayName:  displayName,
		Description:  apimodel.LargeText(description),
		PipelineSpec: apimodel.LargeText(piplineSpec),
	}

	pipelineVersion, err := k8sapi.FromPipelineVersionModel(pipelineModel, pipelineVersionModel)
	if err != nil {
		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to parse pipeline version. Params: '%v'", parameters),
			"Failed to parse pipeline version")
	}

	err = c.ctrlClient.Create(ctx, pipelineVersion)
	if err != nil {
		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to upload pipeline version. Params: '%v'", parameters),
			"Failed to upload pipeline version")
	}

	rv := &model.V2beta1Pipeline{
		CreatedAt:   strfmt.DateTime(pipeline.CreationTimestamp.Time),
		Description: pipeline.Spec.Description,
		DisplayName: pipeline.Spec.DisplayName,
		Name:        pipeline.Name,
		PipelineID:  string(pipeline.UID),
		Namespace:   pipeline.Namespace,
	}

	return rv, nil
}

// UploadPipelineVersion uploads pipeline version from local file.
func (c *PipelineUploadClientKubernetes) UploadPipelineVersion(filePath string, parameters *params.UploadPipelineVersionParams) (*model.V2beta1PipelineVersion,
	error,
) {
	if parameters.Pipelineid == nil {
		return nil, util.NewUserError(errors.New("pipelineid is required"),
			"pipelineid is required",
			"pipelineid is required")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, util.NewUserErrorWithSingleMessage(err,
			fmt.Sprintf("Failed to open file '%s'", filePath))
	}
	defer file.Close()

	processedFile, err := server.ReadPipelineFile(path.Base(filePath), file, common.MaxFileLength)
	if err != nil {
		return nil, util.NewUserErrorWithSingleMessage(err, "Failed to read pipeline spec file")
	}

	ctx, cancel := context.WithTimeout(context.Background(), api_server.APIServerDefaultTimeout)
	defer cancel()

	pipelineList := &k8sapi.PipelineList{}
	err = c.ctrlClient.List(ctx, pipelineList, &ctrlclient.ListOptions{
		Namespace: c.namespace,
	})
	if err != nil {
		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to list pipelines. Params: '%v'", parameters),
			"Failed to list pipelines")
	}

	var pipeline *k8sapi.Pipeline

	for _, listedPipeline := range pipelineList.Items {
		if string(listedPipeline.UID) == *parameters.Pipelineid {
			pipeline = &listedPipeline
			break
		}
	}

	if pipeline == nil {
		return nil, util.NewUserError(errors.New("pipeline not found"),
			fmt.Sprintf("Pipeline with id %s not found", *parameters.Pipelineid),
			"Pipeline not found")
	}

	modelPipeline := pipeline.ToModel()

	name, displayName, description := deriveNameDisplayAndDescription(
		parameters.Name,
		parameters.DisplayName,
		parameters.Description,
		path.Base(filePath),
	)

	modelPipelineVersion := apimodel.PipelineVersion{
		Name:         name,
		PipelineSpec: apimodel.LargeText(processedFile),
		DisplayName:  displayName,
		Description:  apimodel.LargeText(description),
		PipelineId:   modelPipeline.UUID,
	}

	pipelineVersion, err := k8sapi.FromPipelineVersionModel(*modelPipeline, modelPipelineVersion)
	if err != nil {
		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to parse pipeline version. Params: '%v'", parameters),
			"Failed to parse pipeline version")
	}

	err = c.ctrlClient.Create(ctx, pipelineVersion)
	if err != nil {
		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to upload pipeline version. Params: '%v'", parameters),
			"Failed to upload pipeline version")
	}

	pipelineVersionModel, err := pipelineVersion.ToModel()
	if err != nil {
		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to convert pipeline version to model. Params: '%v'", parameters),
			"Encountered an invalid pipeline version")
	}

	rv := &model.V2beta1PipelineVersion{
		CreatedAt:         strfmt.DateTime(pipelineVersion.CreationTimestamp.Time),
		Description:       string(pipelineVersionModel.Description),
		DisplayName:       pipelineVersionModel.DisplayName,
		Name:              pipelineVersionModel.Name,
		PipelineID:        modelPipeline.UUID,
		PipelineVersionID: pipelineVersionModel.UUID,
		PipelineSpec:      nil,
		CodeSourceURL:     pipelineVersionModel.CodeSourceUrl,
	}

	// Handles the case where there is a platform spec in the pipeline spec.
	if spec, err := server.YamlStringToPipelineSpecStruct(string(pipelineVersionModel.PipelineSpec)); err == nil && spec != nil {
		rv.PipelineSpec = spec.AsMap()
	} else if err != nil {
		return nil, util.NewUserError(err,
			fmt.Sprintf("Failed to parse pipeline version. Params: '%v'", parameters),
			"Failed to parse pipeline version")
	}

	return rv, nil
}
