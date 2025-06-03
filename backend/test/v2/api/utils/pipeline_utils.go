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
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	. "github.com/onsi/gomega"
	"log"
)

func ListPipelines(client *api_server.PipelineClient) (
	[]*pipeline_model.V2beta1Pipeline, int, string, error,
) {
	parameters := &pipeline_params.PipelineServiceListPipelinesParams{}
	return client.List(parameters)
}

func DeletePipeline(client *api_server.PipelineClient, pipelineId string) {
	log.Printf("Deleting all pipeline version of pipeline with id=%s", pipelineId)
	DeleteAllPipelineVersions(client, pipelineId)
	log.Printf("Deleting pipeline with id=%s", pipelineId)
	err := client.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{PipelineID: pipelineId})
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Error occurred while deleting pipeline with id=%s", pipelineId))
	log.Printf("Pipeline with id=%s, DELETED", pipelineId)
}

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
		Expect(client.Delete(&pipeline_params.PipelineServiceDeletePipelineParams{PipelineID: pId})).NotTo(HaveOccurred(), fmt.Sprintf("Error occurred while deleting pipeline with id=%s", pId))
	}
	for _, isRemoved := range deletedPipelines {
		Expect(isRemoved).To(BeTrue())
	}
}
