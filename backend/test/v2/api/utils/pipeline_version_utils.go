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
	"os"
	"sort"
	"time"

	pipeline_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_client/pipeline_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_model"
	api_server "github.com/kubeflow/pipelines/backend/src/common/client/api_server/v2"
	"github.com/kubeflow/pipelines/backend/test/v2/api/logger"
	. "github.com/onsi/gomega"
	"sigs.k8s.io/yaml"
)

// JsonFromYAML - Construct expected Pipeline Spec from the uploaded file
func JsonFromYAML(pipelineFilePath string) []byte {
	pipelineSpec, err := os.ReadFile(pipelineFilePath)
	Expect(err).NotTo(HaveOccurred())
	jsonSpecFromFile, errDataConvertion := yaml.YAMLToJSON(pipelineSpec)
	Expect(errDataConvertion).NotTo(HaveOccurred())
	return jsonSpecFromFile
}

func ListPipelineVersions(client *api_server.PipelineClient, pipelineID string) (
	[]*pipeline_model.V2beta1PipelineVersion, int, string, error,
) {
	logger.Log("Listing pipeline versions for pipeline %s", pipelineID)
	parameters := &pipeline_params.PipelineServiceListPipelineVersionsParams{PipelineID: pipelineID}
	return client.ListPipelineVersions(parameters)
}

func DeletePipelineVersion(client *api_server.PipelineClient, pipelineID string, pipelineVersionID string) {
	logger.Log("Deleting pipeline versions for pipeline %s with version id=%s", pipelineID, pipelineVersionID)
	err := client.DeletePipelineVersion(&pipeline_params.PipelineServiceDeletePipelineVersionParams{PipelineID: pipelineID, PipelineVersionID: pipelineVersionID})
	Expect(err).NotTo(HaveOccurred(), fmt.Sprintf("Pipeline version with id=%s of pipelineID=%s failed", pipelineVersionID, pipelineID))
}

func DeleteAllPipelineVersions(client *api_server.PipelineClient, pipelineID string) {
	logger.Log("Deleting all pipeline versions for pipeline %s", pipelineID)
	pipelineVersions, _, _, err := ListPipelineVersions(client, pipelineID)
	Expect(err).NotTo(HaveOccurred(), "Error occurred while listing pipeline versions")
	for _, pv := range pipelineVersions {
		Expect(client.DeletePipelineVersion(&pipeline_params.PipelineServiceDeletePipelineVersionParams{PipelineID: pipelineID, PipelineVersionID: pv.PipelineVersionID})).NotTo(HaveOccurred(), fmt.Sprintf("Pipeline version with id=%s of pipelineID=%s failed", pv.PipelineVersionID, pipelineID))
	}
}

// GetSortedPipelineVersionsByCreatedAt - Get a list of pipeline upload version for a specific pipeline, and sort the list by CreatedAt before returning it
//
// sortBy - ASC or DESC, If nil, then the default will be DESC
func GetSortedPipelineVersionsByCreatedAt(client *api_server.PipelineClient, pipelineID string, sortBy *string) []*pipeline_model.V2beta1PipelineVersion {
	versions, _, _, err := ListPipelineVersions(client, pipelineID)
	if err != nil {
		return nil
	}
	sort.Slice(versions, func(i, j int) bool {
		versionTime1 := time.Time(versions[i].CreatedAt).UTC()
		versionTime2 := time.Time(versions[j].CreatedAt).UTC()
		if sortBy != nil && *sortBy == "ASC" {
			return versionTime1.Before(versionTime2)
		} else {
			return versionTime1.After(versionTime2)
		}
	})
	return versions
}
