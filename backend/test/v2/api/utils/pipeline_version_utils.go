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
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	. "github.com/onsi/gomega"
	"os"
	"sigs.k8s.io/yaml"
)

/*
Construct expected Pipeline Spec from the uploaded file
*/
func JsonFromYAML(pipelineFilePath string) []byte {
	pipelineSpec, err := os.ReadFile(pipelineFilePath)
	Expect(err).NotTo(HaveOccurred())
	jsonSpecFromFile, errDataConvertion := yaml.YAMLToJSON(pipelineSpec)
	Expect(errDataConvertion).NotTo(HaveOccurred())
	return jsonSpecFromFile
}

func ListPipelineVersions(client *api_server.PipelineClient, pipelineId string) (
	[]*pipeline_model.V2beta1PipelineVersion, int, string, error,
) {
	logger.Log("Listing pipeline versions for pipeline %s", pipelineId)
	parameters := &pipeline_params.PipelineServiceListPipelineVersionsParams{PipelineID: pipelineId}
	return client.ListPipelineVersions(parameters)
}

func DeletePipelineVersion(client *api_server.PipelineClient, pipelineId string, pipelineVersionId string) {
	logger.Log("Deleting pipeline versions for pipeline %s with version id=%s", pipelineId, pipelineVersionId)
	err := client.DeletePipelineVersion(&pipeline_params.PipelineServiceDeletePipelineVersionParams{PipelineID: pipelineId, PipelineVersionID: pipelineVersionId})
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Pipeline version with id=%s of pipelineId=%s failed", pipelineVersionId, pipelineId))
}

func DeleteAllPipelineVersions(client *api_server.PipelineClient, pipelineId string) {
	logger.Log("Deleting all pipeline versions for pipeline %s", pipelineId)
	pipelineVersions, _, _, err := ListPipelineVersions(client, pipelineId)
	Expect(err).NotTo(HaveOccurred(), "Error occurred while listing pipeline versions")
	for _, pv := range pipelineVersions {
		Expect(client.DeletePipelineVersion(&pipeline_params.PipelineServiceDeletePipelineVersionParams{PipelineID: pipelineId, PipelineVersionID: pv.PipelineVersionID})).NotTo(HaveOccurred(), fmt.Sprintf("Pipeline version with id=%s of pipelineId=%s failed", pv.PipelineVersionID, pipelineId))
	}
}
