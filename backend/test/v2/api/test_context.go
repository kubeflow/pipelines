// Copyright 2021-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"time"

	uploadparams "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
)

type TestContext struct {
	// Common Test Data
	TestStartTimeUTC time.Time

	// Pipeline Context
	Pipeline Pipeline

	PipelineRun PipelineRun
	Experiment  Experiment
}

type Pipeline struct {
	PipelineGeneratedName string
	UploadParams          *uploadparams.UploadPipelineParams
	ExpectedPipeline      *pipeline_upload_model.V2beta1Pipeline
	CreatedPipelines      []*pipeline_upload_model.V2beta1Pipeline
}

type PipelineRun struct {
	CreatedRunIds []string
}

type Experiment struct {
	CreatedExperimentIds []string
}
