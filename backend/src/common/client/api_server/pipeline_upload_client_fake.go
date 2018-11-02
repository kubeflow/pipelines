package api_server

import (
	"fmt"

	"github.com/go-openapi/strfmt"
	params "github.com/googleprivate/ml/backend/api/go_http_client/pipeline_upload_client/pipeline_upload_service"
	model "github.com/googleprivate/ml/backend/api/go_http_client/pipeline_upload_model"
)

const (
	FileForDefaultTest     = "./samples/parameters.yaml"
	FileForClientErrorTest = "./samples/hello-world.yaml"

	ClientErrorString  = "Error with client"
	InvalidFakeRequest = "Invalid fake request"
)

func getDefaultUploadedPipeline() *model.APIPipeline {
	return &model.APIPipeline{
		ID:          "500",
		CreatedAt:   strfmt.NewDateTime(),
		Name:        "PIPELINE_NAME",
		Description: "PIPELINE_DESCRIPTION",
		Parameters: []*model.APIParameter{&model.APIParameter{
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
	parameters *params.UploadPipelineParams) (*model.APIPipeline, error) {
	switch filePath {
	case FileForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	case FileForDefaultTest:
		return getDefaultUploadedPipeline(), nil
	default:
		return nil, fmt.Errorf(InvalidFakeRequest)
	}
}
