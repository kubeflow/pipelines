package api_server

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/pipeline_upload_model"
)

const (
	FileForDefaultTest     = "./samples/parameters.yaml"
	FileForClientErrorTest = "./samples/hello-world.yaml"

	ClientErrorString  = "Error with client"
	InvalidFakeRequest = "Invalid fake request, don't know how to handle '%s' in the fake client."
)

func getDefaultUploadedPipeline() *model.V1beta1Pipeline {
	return &model.V1beta1Pipeline{
		ID:          "500",
		CreatedAt:   strfmt.NewDateTime(),
		Name:        "PIPELINE_NAME",
		Description: "PIPELINE_DESCRIPTION",
		Parameters: []*model.V1beta1Parameter{&model.V1beta1Parameter{
			Name:  "PARAM_NAME",
			Value: "PARAM_VALUE",
		}},
	}
}

type PipelineUploadClientFake struct{}

func NewPipelineUploadClientFake() *PipelineUploadClientFake {
	return &PipelineUploadClientFake{}
}

func (c *PipelineUploadClientFake) UploadFile(filePath string,
	parameters *params.UploadPipelineParams) (*model.V1beta1Pipeline, error) {
	switch filePath {
	case FileForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	default:
		return getDefaultUploadedPipeline(), nil
	}
}

// TODO(jingzhang36): add UploadPipelineVersion fake to be used in integration test
// after go_http_client and go_client are auto-generated from UploadPipelineVersion in PipelineUploadServer
