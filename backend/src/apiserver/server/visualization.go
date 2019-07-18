package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type VisualizationServer struct {
	resourceManager *resource.ResourceManager
}

func (s *VisualizationServer) CreateVisualization(ctx context.Context, request *go_client.CreateVisualizationRequest) (*go_client.Visualization, error) {
	if err := s.validateCreateVisualizationRequest(request.Visualization); err != nil {
		return nil, err
	}
	args, err := s.getArgumentsAsJSONFromVisualization(request.Visualization)
	if err != nil {
		return nil, err
	}
	arguments := s.createPythonArgumentsFromTypeAndJSON(request.Visualization.Type, args)
	body, err := s.generateVisualization(arguments)
	if err != nil {
		return nil, err
	}
	request.Visualization.Html = string(body)
	return request.Visualization, nil
}

func (s *VisualizationServer) validateCreateVisualizationRequest(visualization *go_client.Visualization) error {
	if len(visualization.InputPath) == 0 {
		return util.NewInvalidInputError("A visualization requires an InputPath to be provided. Received %s", visualization.InputPath)
	}
	// Manually set Arguments to empty JSON if nothing is provided. This is done
	// because visualizations such as TFDV and TFMA only require an InputPath to
	// provided for a visualization tobe generated. If no JSON is provided
	// json.Valid will fail without this check as an empty string is provided for
	// those visualizations.
	if len(visualization.Arguments) == 0 {
		visualization.Arguments = "{}"
	}
	if !json.Valid([]byte(visualization.Arguments)) {
		return util.NewInvalidInputError("A visualization requires valid JSON to be provided as Arguments. Received %s", visualization.Arguments)
	}
	return nil
}

func (s *VisualizationServer) getArgumentsAsJSONFromVisualization(visualization *go_client.Visualization) ([]byte, error) {
	var arguments map[string]interface{}
	if err := json.Unmarshal([]byte(visualization.Arguments), &arguments); err != nil {
		return nil, util.Wrap(err, "Unable to parse provided JSON.")
	}
	arguments["input_path"] = visualization.InputPath
	args, err := json.Marshal(arguments)
	if err != nil {
		return nil, util.Wrap(err, "Unable to compose provided JSON as string.")
	}
	return args, nil
}

func (s *VisualizationServer) createPythonArgumentsFromTypeAndJSON(visualizationType go_client.Visualization_Type, arguments []byte) string {
	var _visualizationType = strings.ToLower(go_client.Visualization_Type_name[int32(visualizationType)])
	return fmt.Sprintf("--type %s --arguments '%s'", _visualizationType, arguments)
}

func (s *VisualizationServer) generateVisualization(arguments string) ([]byte, error) {
	resp, err := http.PostForm("http://visualization-service.kubeflow", url.Values{"arguments": {arguments}})
	if err != nil {
		return nil, util.Wrap(err, "Unable to initialize visualization request.")
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, util.Wrap(err, "Unable to parse visualization response.")
	}
	return body, nil
}

func NewVisualizationServer(resourceManager *resource.ResourceManager) *VisualizationServer {
	return &VisualizationServer{resourceManager: resourceManager}
}
