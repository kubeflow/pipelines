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

package test

import (
	"fmt"
	pipeline_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	. "github.com/onsi/gomega"
)

func ListPipelines(client *api_server.PipelineClient) (
	[]*pipeline_model.V2beta1Pipeline, int, string, error,
) {
	parameters := &pipeline_params.PipelineServiceListPipelinesParams{}
	logger.Log("Listing all pipelines")
	return client.List(parameters)
}

/*
Delete a pipeline by id
*/
func DeletePipeline(client *api_server.PipelineClient, pipelineId string) {
	logger.Log("Deleting all pipeline version of pipeline with id=%s", pipelineId)
	DeleteAllPipelineVersions(client, pipelineId)
	logger.Log("Deleting pipeline with id=%s", pipelineId)
	err := client.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{PipelineID: pipelineId})
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error occurred while deleting pipeline with id=%s", pipelineId))
	logger.Log("Pipeline with id=%s, DELETED", pipelineId)
}

/*
Delete all pipelines
*/
func DeleteAllPipelines(client *api_server.PipelineClient) {
	pipelines, _, _, err := ListPipelines(client)
	Expect(err).NotTo(HaveOccurred(), "Error occurred while listing pipelines")
	deletedPipelines := make(map[string]bool)
	for _, p := range pipelines {
		deletedPipelines[p.PipelineID] = false
	}
	for pId, isRemoved := range deletedPipelines {
		if !isRemoved {
			DeleteAllPipelineVersions(client, pId)
			deletedPipelines[pId] = true
		}
		logger.Log("Deleting pipeline with id=%s", pId)
		Expect(client.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{PipelineID: pId})).NotTo(HaveOccurred(), fmt.Sprintf("Error occurred while deleting pipeline with id=%s", pId))
	}
	for _, isRemoved := range deletedPipelines {
		Expect(isRemoved).To(BeTrue())
	}
}

/*
Get pipeline via GET pipeline end point call, so that we retreive the values from DB
*/
func GetPipeline(client *api_server.PipelineClient, pipelineId string) model.V2beta1Pipeline {
	params := new(pipeline_params.PipelineServiceGetPipelineParams)
	params.PipelineID = pipelineId
	logger.Log("Get pipeline with id=%s", pipelineId)
	pipeline, err := client.Get(params)
	Expect(err).NotTo(HaveOccurred())
	return model.V2beta1Pipeline{
		DisplayName: pipeline.DisplayName,
		Description: pipeline.Description,
		PipelineID:  pipeline.PipelineID,
		CreatedAt:   pipeline.CreatedAt,
		Namespace:   pipeline.Namespace,
	}
}

/*
Get all pipelines (upto 1000) and filter by name, if the pipeline exists, return true otherwise false
*/
func FindPipelineByName(client *api_server.PipelineClient, pipelineName string) bool {
	requestedNumberOfPipelinesPerPage := 1000
	params := new(pipeline_params.PipelineServiceListPipelinesParams)
	params.PageSize = util.Int32Pointer(int32(requestedNumberOfPipelinesPerPage))
	logger.Log("Get all pipelines")
	pipelines, size, _, err := client.List(params)
	Expect(err).NotTo(HaveOccurred())
	logger.Log("Finding pipeline with name=%s", pipelineName)
	if size < requestedNumberOfPipelinesPerPage {
		for _, pipeline := range pipelines {
			if pipeline.DisplayName == pipelineName {
				return true
			} else {
				return false
			}
		}
	}
	return false
}
