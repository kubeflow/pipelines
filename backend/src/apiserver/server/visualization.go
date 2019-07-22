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
	serviceURL      string
}

func (s *VisualizationServer) CreateVisualization(ctx context.Context, request *go_client.CreateVisualizationRequest) (*go_client.Visualization, error) {
	if err := s.ValidateCreateVisualizationRequest(request.Visualization); err != nil {
		return nil, err
	}
	arguments, err := s.GetArgumentsAsJSONFromVisualization(request.Visualization)
	if err != nil {
		return nil, err
	}
	pythonArguments := s.CreatePythonArgumentsFromTypeAndJSON(request.Visualization.Type, arguments)
	body, err := s.GenerateVisualization(pythonArguments)
	if err != nil {
		return nil, err
	}
	request.Visualization.Html = string(body)
	return request.Visualization, nil
}

// ValidateCreateVisualizationRequest ensures that a go_client.Visualization
// object has valid values.
// It returns an error if a go_client.Visualization object does not have valid
// values.
func (s *VisualizationServer) ValidateCreateVisualizationRequest(visualization *go_client.Visualization) error {
	if len(visualization.InputPath) == 0 {
		return util.NewInvalidInputError("A visualization requires an InputPath to be provided. Received %s", visualization.InputPath)
	}
	// Manually set Arguments to empty JSON if nothing is provided. This is done
	// because visualizations such as TFDV and TFMA only require an InputPath to
	// provided for a visualization to be generated. If no JSON is provided
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

// GetArgumentsAsJSONFromVisualization will convert the values within a
// go_client.Visualization object to valid JSON that can be used to pass
// arguments to the python visualization service.
// It returns the generated JSON as an array of bytes and any error that is
// encountered.
func (s *VisualizationServer) GetArgumentsAsJSONFromVisualization(visualization *go_client.Visualization) ([]byte, error) {
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

// CreatePythonArgumentsFromTypeAndJSON converts the values within a
// go_client.Visualization object to those expected by the python visualization
// service.
// It returns the converted values as a string.
func (s *VisualizationServer) CreatePythonArgumentsFromTypeAndJSON(visualizationType go_client.Visualization_Type, arguments []byte) string {
	var _visualizationType = strings.ToLower(go_client.Visualization_Type_name[int32(visualizationType)])
	return fmt.Sprintf("--type %s --arguments '%s'", _visualizationType, arguments)
}

// GenerateVisualization communicates with the python visualization service to
// generate HTML visualizations from specified arguments.
// It returns the generated HTML as a string and any error that is encountered.
func (s *VisualizationServer) GenerateVisualization(arguments string) ([]byte, error) {
	resp, err := http.PostForm(s.serviceURL, url.Values{"arguments": {arguments}})
	if err != nil {
		return nil, util.Wrap(err, "Unable to initialize visualization request.")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(resp.Status)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, util.Wrap(err, "Unable to parse visualization response.")
	}
	return body, nil
}

func NewVisualizationServer(resourceManager *resource.ResourceManager) *VisualizationServer {
	return &VisualizationServer{resourceManager: resourceManager, serviceURL: "http://visualization-service.kubeflow"}
}
