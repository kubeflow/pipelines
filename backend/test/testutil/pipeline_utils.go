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

package testutil

import (
	"fmt"
	"os"

	pipeline_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test/logger"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"k8s.io/client-go/tools/clientcmd"
)

func GetPipelineUploadClient(
	uploadPipelinesWithKubernetes bool,
	isKubeflowMode bool,
	isDebugMode bool,
	namespace string,
	clientConfig clientcmd.ClientConfig,
) (api_server.PipelineUploadInterface, error) {
	if uploadPipelinesWithKubernetes {
		return api_server.NewPipelineUploadClientKubernetes(clientConfig, namespace)
	}

	if isKubeflowMode {
		return api_server.NewKubeflowInClusterPipelineUploadClient(namespace, isDebugMode)
	}

	return api_server.NewPipelineUploadClient(clientConfig, isDebugMode)
}

func ListPipelines(client *api_server.PipelineClient, namespace *string) []*pipeline_model.V2beta1Pipeline {
	parameters := &pipeline_params.PipelineServiceListPipelinesParams{}
	if namespace != nil {
		parameters.Namespace = namespace
	}
	logger.Log("Listing all pipelines")
	pipelines, _, _, err := client.List(parameters)
	gomega.Expect(err).NotTo(gomega.HaveOccurred(), "Error listing pipelines")
	return pipelines
}

// UploadPipeline - Upload a pipeline
func UploadPipeline(pipelineUploadClient api_server.PipelineUploadInterface, pipelineFilePath string, pipelineName *string, pipelineDisplayName *string) (*model.V2beta1Pipeline, error) {
	uploadParams := upload_params.NewUploadPipelineParams()
	uploadParams.SetName(pipelineName)
	if pipelineDisplayName != nil {
		uploadParams.SetDisplayName(pipelineDisplayName)
	}
	logger.Log("Creating temp pipeline file with overridden SDK Version")
	overriddenPipelineFileWithSDKVersion := ReplaceSDKInPipelineSpec(pipelineFilePath, false, nil)
	tempPipelineFile := CreateTempFile(overriddenPipelineFileWithSDKVersion)
	defer func() {
		// Ensure the temporary file is removed when the function exits
		if err := os.Remove(tempPipelineFile.Name()); err != nil {
			logger.Log("Error removing temporary file: %s", err)
		}
	}()
	logger.Log("Uploading pipeline with name=%s, from file %s", *pipelineName, pipelineFilePath)
	return pipelineUploadClient.UploadFile(tempPipelineFile.Name(), uploadParams)
}

/* DeletePipeline deletes a pipeline by id */
func DeletePipeline(client *api_server.PipelineClient, pipelineID string) {
	ginkgo.GinkgoHelper()
	_, err := client.Get(&pipeline_params.PipelineServiceGetPipelineParams{PipelineID: pipelineID})
	if err == nil {
		logger.Log("Deleting all pipeline version of pipeline with id=%s", pipelineID)
		DeleteAllPipelineVersions(client, pipelineID)
		logger.Log("Deleting pipeline with id=%s", pipelineID)
		err = client.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{PipelineID: pipelineID})
		gomega.Expect(err).NotTo(gomega.HaveOccurred(), fmt.Sprintf("Error occurred while deleting pipeline with id=%s", pipelineID))
		logger.Log("Pipeline with id=%s, DELETED", pipelineID)
	} else {
		logger.Log("Pipeline with id=%s does not exist, so skipping deleting it", pipelineID)
	}

}

/* DeleteAllPipelines deletes all pipelines */
func DeleteAllPipelines(client *api_server.PipelineClient, namespace *string) {
	ginkgo.GinkgoHelper()
	pipelines := ListPipelines(client, namespace)
	deletedPipelines := make(map[string]bool)
	for _, p := range pipelines {
		deletedPipelines[p.PipelineID] = false
	}
	for pID, isRemoved := range deletedPipelines {
		if !isRemoved {
			DeleteAllPipelineVersions(client, pID)
			deletedPipelines[pID] = true
		}
		logger.Log("Deleting pipeline with id=%s", pID)
		gomega.Expect(client.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{PipelineID: pID})).NotTo(gomega.HaveOccurred(), fmt.Sprintf("Error occurred while deleting pipeline with id=%s", pID))
	}
	for _, isRemoved := range deletedPipelines {
		gomega.Expect(isRemoved).To(gomega.BeTrue())
	}
}

/* GetPipeline does its job via GET pipeline end point call, so that we retrieve the values from DB */
func GetPipeline(client *api_server.PipelineClient, pipelineID string) model.V2beta1Pipeline {
	ginkgo.GinkgoHelper()
	params := new(pipeline_params.PipelineServiceGetPipelineParams)
	params.PipelineID = pipelineID
	logger.Log("Get pipeline with id=%s", pipelineID)
	pipeline, err := client.Get(params)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	return model.V2beta1Pipeline{
		DisplayName: pipeline.DisplayName,
		Description: pipeline.Description,
		PipelineID:  pipeline.PipelineID,
		CreatedAt:   pipeline.CreatedAt,
		Namespace:   pipeline.Namespace,
		Name:        pipeline.Name,
	}
}

/* FindPipelineByName gets all pipelines (upto 1000) and filter by name, if the pipeline exists, return true otherwise false
 */
func FindPipelineByName(client *api_server.PipelineClient, pipelineName string) bool {
	ginkgo.GinkgoHelper()
	requestedNumberOfPipelinesPerPage := 1000
	params := new(pipeline_params.PipelineServiceListPipelinesParams)
	params.PageSize = util.Int32Pointer(int32(requestedNumberOfPipelinesPerPage))
	logger.Log("Get all pipelines")
	pipelines, size, _, err := client.List(params)
	gomega.Expect(err).NotTo(gomega.HaveOccurred())
	logger.Log("Finding pipeline with name=%s", pipelineName)
	if size < requestedNumberOfPipelinesPerPage {
		for _, pipeline := range pipelines {
			if pipeline.DisplayName == pipelineName {
				return true
			}
		}
	}
	return false
}
