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
