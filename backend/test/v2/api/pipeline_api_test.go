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

package api

import (
	"fmt"
	"github.com/go-openapi/strfmt"
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	matcher "github.com/kubeflow/pipelines/backend/test/v2/api/matcher"
	utils "github.com/kubeflow/pipelines/backend/test/v2/api/utils"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"log"
	"slices"
	"time"
)

const (
	hello_world_pipeline_name = "hello-world.yaml"
	pipeline_with_args_name   = "arguments-parameters.yaml"
)

var expectedPipeline *model.V2beta1Pipeline
var createdPipelines []*model.V2beta1Pipeline
var testStartTime strfmt.DateTime

var _ = BeforeEach(func() {
	testStartTime, _ = strfmt.ParseDateTime(time.Now().Format(time.DateTime))
	expectedPipeline = new(model.V2beta1Pipeline)
	expectedPipeline.CreatedAt = testStartTime
	//expectedPipeline.Namespace = *namespace
})

var _ = AfterEach(func() {
	// Delete pipelines created during the test
	for i, pipeline := range createdPipelines {
		utils.DeletePipeline(pipelineClient, pipeline.PipelineID)
		createdPipelines = slices.Delete(createdPipelines, i, i+1)
	}
})

var _ = Describe("Verify Pipeline APIs", func() {

	/* Test 1:  */
	Context("Upload a pipeline and verify pipeline metadata after upload", func() {
		It(fmt.Sprintf("Upload %s pipeline", hello_world_pipeline_name), func() {
			createPipelineAndVerify(hello_world_pipeline_name)
		})

		It(fmt.Sprintf("Upload %s pipeline", pipeline_with_args_name), func() {
			createPipelineAndVerify(pipeline_with_args_name)
		})

		It(fmt.Sprintf("Upload duplicate pipeline with name '%s' pipeline", pipeline_with_args_name), func() {
			createPipelineAndVerify(pipeline_with_args_name)
			_, err := createPipeline(pipeline_with_args_name)
			Expect(err.Error()).To(ContainSubstring("Failed to upload pipeline"))
		})
	})
})

func createPipeline(pipelineName string) (*model.V2beta1Pipeline, error) {
	pipelineFile := "../resources/" + pipelineName
	log.Printf("Creating pipeline with name=%s, from file %s", pipelineName, pipelineFile)
	return pipelineUploadClient.UploadFile(pipelineFile, upload_params.NewUploadPipelineParams())
}

func createPipelineAndVerify(pipelineName string) {
	createdPipeline, err := createPipeline(pipelineName)
	Expect(err).NotTo(HaveOccurred())
	expectedPipeline.DisplayName = pipelineName
	createdPipelines = append(createdPipelines, createdPipeline)
	matcher.MatchPipelines(createdPipeline, expectedPipeline)
}
