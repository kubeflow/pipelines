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
	if err := s.validateCreateVisualizationRequest(request); err != nil {
		return nil, err
	}
	body, err := s.GenerateVisualizationFromRequest(request)
	if err != nil {
		return nil, err
	}
	request.Visualization.Html = string(body)
	return request.Visualization, nil
}

// validateCreateVisualizationRequest ensures that a go_client.Visualization
// object has valid values.
// It returns an error if a go_client.Visualization object does not have valid
// values.
func (s *VisualizationServer) validateCreateVisualizationRequest(request *go_client.CreateVisualizationRequest) error {
	if len(request.Visualization.InputPath) == 0 {
		return util.NewInvalidInputError("A visualization requires an InputPath to be provided. Received %s", request.Visualization.InputPath)
	}
	// Manually set Arguments to empty JSON if nothing is provided. This is done
	// because visualizations such as TFDV and TFMA only require an InputPath to
	// provided for a visualization to be generated. If no JSON is provided
	// json.Valid will fail without this check as an empty string is provided for
	// those visualizations.
	if len(request.Visualization.Arguments) == 0 {
		request.Visualization.Arguments = "{}"
	}
	if !json.Valid([]byte(request.Visualization.Arguments)) {
		return util.NewInvalidInputError("A visualization requires valid JSON to be provided as Arguments. Received %s", request.Visualization.Arguments)
	}
	return nil
}

// getArgumentsAsJSONFromRequest will convert the values within a
// go_client.CreateVisualizationRequest object to valid JSON that can be used to
// pass arguments to the python visualization service.
// It returns the generated JSON as an array of bytes and any error that is
// encountered.
func (s *VisualizationServer) getArgumentsAsJSONFromRequest(request *go_client.CreateVisualizationRequest) ([]byte, error) {
	var arguments map[string]interface{}
	if err := json.Unmarshal([]byte(request.Visualization.Arguments), &arguments); err != nil {
		return nil, util.Wrap(err, "Unable to parse provided JSON.")
	}
	args, err := json.Marshal(arguments)
	if err != nil {
		return nil, util.Wrap(err, "Unable to compose provided JSON as string.")
	}
	return args, nil
}

// createPythonArgumentsFromRequest converts the values within a
// go_client.CreateVisualizationRequest object to those expected by the python
// visualization service.
// It returns the converted values as a string and any error that is
// encountered.
func (s *VisualizationServer) createPythonArgumentsFromRequest(request *go_client.CreateVisualizationRequest) (string, error) {
	visualizationType := strings.ToLower(go_client.Visualization_Type_name[int32(request.Visualization.Type)])
	arguments, err := s.getArgumentsAsJSONFromRequest(request)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("--type %s --input_path %s --arguments '%s'", visualizationType, request.Visualization.InputPath, arguments), nil
}

// GenerateVisualizationFromRequest communicates with the python visualization
// service to generate HTML visualizations from a request.
// It returns the generated HTML as a string and any error that is encountered.
func (s *VisualizationServer) GenerateVisualizationFromRequest(request *go_client.CreateVisualizationRequest) ([]byte, error) {
	arguments, err := s.createPythonArgumentsFromRequest(request)
	if err != nil {
		return nil, err
	}
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
