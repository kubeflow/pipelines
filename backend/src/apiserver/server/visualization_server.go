package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/resource"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/pkg/errors"
	authorizationv1 "k8s.io/api/authorization/v1"
)

const (
	visualizationServiceName = "VisualizationService.Name"
	visualizationServicePort = "VisualizationService.Port"
)

type VisualizationServer struct {
	resourceManager *resource.ResourceManager
	serviceURL      string
}

func (s *VisualizationServer) CreateVisualization(ctx context.Context, request *go_client.CreateVisualizationRequest) (*go_client.Visualization, error) {
	if err := s.validateCreateVisualizationRequest(request); err != nil {
		return nil, err
	}

	// In multi-user mode, we allow empty namespace in which case we fall back to use the visualization service in system namespace.
	// See getVisualizationServiceURL() for details.
	if common.IsMultiUserMode() && len(request.Namespace) > 0 {
		resourceAttributes := &authorizationv1.ResourceAttributes{
			Namespace:   request.Namespace,
			Verb:        common.RbacResourceVerbCreate,
			Group:       common.RbacPipelinesGroup,
			Version:     common.RbacPipelinesVersion,
			Resource:    common.RbacResourceTypeVisualizations,
			Subresource: "",
			Name:        "",
		}
		err := isAuthorized(s.resourceManager, ctx, resourceAttributes)
		if err != nil {
			return nil, util.Wrap(err, "Failed to authorize on namespace.")
		}
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
	// Only validate that a source is provided for non-custom visualizations.
	if request.Visualization.Type != go_client.Visualization_CUSTOM {
		if len(request.Visualization.Source) == 0 {
			return util.NewInvalidInputError("A visualization requires a Source to be provided. Received %s", request.Visualization.Source)
		}
	}
	// Manually set Arguments to empty JSON if nothing is provided. This is done
	// because visualizations such as TFDV and TFMA only require a Source to
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
	serviceURL := s.getVisualizationServiceURL(request)
	if err := isVisualizationServiceAlive(serviceURL); err != nil {
		return nil, util.Wrap(err, "Cannot generate visualization.")
	}
	visualizationType := strings.ToLower(go_client.Visualization_Type_name[int32(request.Visualization.Type)])
	urlValues := url.Values{
		"arguments": {request.Visualization.Arguments},
		"source":    {request.Visualization.Source},
		"type":      {visualizationType},
	}
	resp, err := http.PostForm(serviceURL, urlValues)
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

func (s *VisualizationServer) getVisualizationServiceURL(request *go_client.CreateVisualizationRequest) string {
	if common.IsMultiUserMode() && len(request.Namespace) > 0 {
		return fmt.Sprintf("http://%s.%s:%s",
			common.GetStringConfig(visualizationServiceName),
			request.Namespace,
			common.GetStringConfig(visualizationServicePort))
	}
	return s.serviceURL
}

func isVisualizationServiceAlive(serviceURL string) error {
	resp, err := http.Get(serviceURL)

	if err != nil {
		wrappedErr := util.Wrap(err, fmt.Sprintf("Unable to verify visualization service aliveness by sending request to %s", serviceURL))
		glog.Error(wrappedErr)
		return wrappedErr
	} else if resp.StatusCode != http.StatusOK {
		wrappedErr := errors.New(fmt.Sprintf("Unable to verify visualization service aliveness by sending request to %s and get response code: %s !", serviceURL, resp.Status))
		glog.Error(wrappedErr)
		return wrappedErr
	}
	return nil
}

func NewVisualizationServer(resourceManager *resource.ResourceManager, serviceHost string, servicePort string) *VisualizationServer {
	serviceURL := fmt.Sprintf("http://%s:%s", serviceHost, servicePort)
	return &VisualizationServer{
		resourceManager: resourceManager,
		serviceURL:      serviceURL,
	}
}
