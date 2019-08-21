package server

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cenkalti/backoff"
	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type VisualizationServer struct {
	resourceManager    *resource.ResourceManager
	serviceURL         string
	isServiceAvailable bool
}

func (s *VisualizationServer) CreateVisualization(ctx context.Context, request *go_client.CreateVisualizationRequest) (*go_client.Visualization, error) {
	if err := s.validateCreateVisualizationRequest(request); err != nil {
		return nil, err
	}
	body, err := s.generateVisualizationFromRequest(request)
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
	if len(request.Visualization.Source) == 0 {
		return util.NewInvalidInputError("A visualization requires a Source to be provided. Received %s", request.Visualization.Source)
	}
	// Manually set Arguments to empty JSON if nothing is provided. This is done
	// because visualizations such as TFDV and TFMA only require an InputPath to
	// be provided for a visualization to be generated. If no JSON is provided
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

// generateVisualizationFromRequest communicates with the python visualization
// service to generate HTML visualizations from a request.
// It returns the generated HTML as a string and any error that is encountered.
func (s *VisualizationServer) generateVisualizationFromRequest(request *go_client.CreateVisualizationRequest) ([]byte, error) {
	if !s.isServiceAvailable {
		return nil, util.NewInternalServerError(
			fmt.Errorf("service not available"),
			"Service not available",
		)
	}
	visualizationType := strings.ToLower(go_client.Visualization_Type_name[int32(request.Visualization.Type)])
	arguments := fmt.Sprintf("--type %s --source %s --arguments '%s'", visualizationType, request.Visualization.Source, request.Visualization.Arguments)
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

func isVisualizationServiceAlive(serviceURL string, initConnectionTimeout time.Duration) bool {
	var operation = func() error {
		_, err := http.Get(serviceURL)
		if err != nil {
			glog.Error("Unable to verify visualization service is alive!", err)
			return err
		}
		return nil
	}
	b := backoff.NewExponentialBackOff()
	b.MaxElapsedTime = initConnectionTimeout
	err := backoff.Retry(operation, b)
	return err == nil
}

func NewVisualizationServer(resourceManager *resource.ResourceManager, serviceHost string, servicePort string, initConnectionTimeout time.Duration) *VisualizationServer {
	serviceURL := fmt.Sprintf("http://%s:%s", serviceHost, servicePort)
	isServiceAvailable := isVisualizationServiceAlive(serviceURL, initConnectionTimeout)
	return &VisualizationServer{
		resourceManager:    resourceManager,
		serviceURL:         serviceURL,
		// TODO: isServiceAvailable is used to determine if the new visualization
		//  service is alive. If this is true, then the service is alive and
		//  requests can be made to it. Otherwise, if it is false, the service is
		//  not alive and requests should not be made. This prevents timeouts and
		//  counteracts current instabilities with the service. This should be
		//  removed after the visualization service is deemed stable.
		isServiceAvailable: isServiceAvailable,
	}
}
