package api_server

import (
	"encoding/json"
	"fmt"

	params "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/visualization_client/visualization_service"
	model "github.com/kubeflow/pipelines/backend/api/v1beta1/go_http_client/visualization_model"
)

type VisualizationArguments struct {
	fail bool
}

type VisualizationClientFake struct{}

func NewVisualizationClientFake() *VisualizationClientFake {
	return &VisualizationClientFake{}
}

func (c *VisualizationClientFake) Create(params *params.VisualizationServiceCreateVisualizationV1Params) (
	*model.APIVisualization, error) {
	var arguments VisualizationArguments
	err := json.Unmarshal([]byte(params.Body.Arguments), &arguments)
	if err != nil {
		return nil, err
	}
	if arguments.fail {
		return nil, fmt.Errorf(ClientErrorString)
	}
	return params.Body, nil
}
