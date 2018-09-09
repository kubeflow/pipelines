package api_server

import (
	"fmt"

	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/googleprivate/ml/backend/api"
)

const (
	FileForDefaultTest     = "./samples/parameters.yaml"
	FileForClientErrorTest = "./samples/hello-world.yaml"

	ClientErrorString  = "Error with client"
	InvalidFakeRequest = "Invalid fake request"
)

func getDefaultUploadedPipeline() *api.Pipeline {
	return &api.Pipeline{
		Id:          "500",
		CreatedAt:   &timestamp.Timestamp{Seconds: 10},
		Name:        "PIPELINE_NAME",
		Description: "PIPELINE_DESCRIPTION",
		Parameters: []*api.Parameter{&api.Parameter{
			Name:  "PARAM_NAME",
			Value: "PARAM_VALUE",
		}},
	}
}

type PipelineUploadClientFake struct{}

func NewPipelineUploadClientFake() *PipelineUploadClientFake {
	return &PipelineUploadClientFake{}
}

func (c *PipelineUploadClientFake) Upload(filePath string) (*api.Pipeline, error) {
	switch filePath {
	case FileForClientErrorTest:
		return nil, fmt.Errorf(ClientErrorString)
	case FileForDefaultTest:
		return getDefaultUploadedPipeline(), nil
	default:
		return nil, fmt.Errorf(InvalidFakeRequest)
	}
}
