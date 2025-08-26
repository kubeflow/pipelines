package api

import (
	upload_params "github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	"github.com/kubeflow/pipelines/backend/api/v2beta1/go_http_client/pipeline_upload_model"
	"time"
)

type TestContext struct {
	// Common Test Data
	TestStartTimeUTC time.Time

	// Pipeline Context
	PipelineGeneratedName string
	UploadParams          *upload_params.UploadPipelineParams
	ExpectedPipeline      *pipeline_upload_model.V2beta1Pipeline
	CreatedPipelines      []*pipeline_upload_model.V2beta1Pipeline
}

type Pipeline struct {
	pipelineName string
}
