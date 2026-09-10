// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"

	apiv1beta1 "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	apiv2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"

	"github.com/golang/glog"
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

func buildVisualizationServiceURL(namespace string) string {
	host := common.GetStringConfig(visualizationServiceName)
	if common.IsMultiUserMode() && len(namespace) > 0 {
		host = fmt.Sprintf("%s.%s", host, namespace)
	}
	u := &url.URL{
		Scheme: "http",
		Host:   net.JoinHostPort(host, common.GetStringConfig(visualizationServicePort)),
	}
	return u.String()
}

func isVisualizationServiceAlive(serviceURL string) error {
	resp, err := http.Get(serviceURL)
	if err != nil {
		wrappedErr := util.Wrapf(err, "Unable to verify visualization service aliveness by sending request to %s", serviceURL)
		glog.Error(wrappedErr)
		return wrappedErr
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		wrappedErr := errors.New(fmt.Sprintf("Unable to verify visualization service aliveness by sending request to %s and got response code: %s", serviceURL, resp.Status))
		glog.Error(wrappedErr)
		return wrappedErr
	}
	return nil
}

type VisualizationServerV1 struct {
	resourceManager *resource.ResourceManager
	apiv1beta1.UnimplementedVisualizationServiceServer
}

func NewVisualizationServerV1(resourceManager *resource.ResourceManager) *VisualizationServerV1 {
	return &VisualizationServerV1{resourceManager: resourceManager}
}

func (s *VisualizationServerV1) CreateVisualizationV1(ctx context.Context, request *apiv1beta1.CreateVisualizationRequest) (*apiv1beta1.Visualization, error) {
	if err := s.validateCreateVisualizationRequest(request); err != nil {
		return nil, err
	}

	// In multi-user mode, allow empty namespace falls back to the
	// visualization service running in the system namespace.
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
		if err := s.resourceManager.IsAuthorized(ctx, resourceAttributes); err != nil {
			return nil, util.Wrap(err, "Failed to authorize on namespace")
		}
	}

	body, err := s.generateVisualization(request)
	if err != nil {
		return nil, err
	}
	request.Visualization.Html = string(body)
	return request.Visualization, nil
}

func (s *VisualizationServerV1) validateCreateVisualizationRequest(request *apiv1beta1.CreateVisualizationRequest) error {
	if request.Visualization.Type != apiv1beta1.Visualization_CUSTOM {
		if len(request.Visualization.Source) == 0 {
			return util.NewInvalidInputError("A visualization requires a Source to be provided. Received %s", request.Visualization.Source)
		}
	}
	if len(request.Visualization.Arguments) == 0 {
		request.Visualization.Arguments = "{}"
	}
	if !json.Valid([]byte(request.Visualization.Arguments)) {
		return util.NewInvalidInputError("A visualization requires valid JSON to be provided as Arguments. Received %s", request.Visualization.Arguments)
	}
	return nil
}

func (s *VisualizationServerV1) generateVisualization(request *apiv1beta1.CreateVisualizationRequest) ([]byte, error) {
	serviceURL := buildVisualizationServiceURL(request.Namespace)
	if err := isVisualizationServiceAlive(serviceURL); err != nil {
		return nil, util.Wrap(err, "Cannot generate visualization")
	}
	visualizationType := strings.ToLower(apiv1beta1.Visualization_Type_name[int32(request.Visualization.Type)])
	urlValues := url.Values{
		"arguments": {request.Visualization.Arguments},
		"source":    {request.Visualization.Source},
		"type":      {visualizationType},
	}
	resp, err := http.PostForm(serviceURL, urlValues)
	if err != nil {
		return nil, util.Wrap(err, "Unable to initialize visualization request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("visualization service returned non-OK status: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, util.Wrap(err, "Unable to parse visualization response")
	}
	return body, nil
}

type VisualizationServer struct {
	resourceManager *resource.ResourceManager
	apiv2beta1.UnimplementedVisualizationServiceServer
}

func NewVisualizationServer(resourceManager *resource.ResourceManager) *VisualizationServer {
	return &VisualizationServer{resourceManager: resourceManager}
}

func (s *VisualizationServer) CreateVisualization(ctx context.Context, request *apiv2beta1.CreateVisualizationRequest) (*apiv2beta1.Visualization, error) {
	if err := s.validateCreateVisualizationRequest(request); err != nil {
		return nil, err
	}

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
		if err := s.resourceManager.IsAuthorized(ctx, resourceAttributes); err != nil {
			return nil, util.Wrap(err, "Failed to authorize on namespace")
		}
	}

	body, err := s.generateVisualization(request)
	if err != nil {
		return nil, err
	}
	request.Visualization.Html = string(body)
	return request.Visualization, nil
}

func (s *VisualizationServer) validateCreateVisualizationRequest(request *apiv2beta1.CreateVisualizationRequest) error {
	if request.Visualization.Type != apiv2beta1.Visualization_CUSTOM {
		if len(request.Visualization.Source) == 0 {
			return util.NewInvalidInputError("A visualization requires a Source to be provided. Received %s", request.Visualization.Source)
		}
	}
	if len(request.Visualization.Arguments) == 0 {
		request.Visualization.Arguments = "{}"
	}
	if !json.Valid([]byte(request.Visualization.Arguments)) {
		return util.NewInvalidInputError("A visualization requires valid JSON to be provided as Arguments. Received %s", request.Visualization.Arguments)
	}
	return nil
}

func (s *VisualizationServer) generateVisualization(request *apiv2beta1.CreateVisualizationRequest) ([]byte, error) {
	serviceURL := buildVisualizationServiceURL(request.Namespace)
	if err := isVisualizationServiceAlive(serviceURL); err != nil {
		return nil, util.Wrap(err, "Cannot generate visualization")
	}
	visualizationType := strings.ToLower(apiv2beta1.Visualization_Type_name[int32(request.Visualization.Type)])
	urlValues := url.Values{
		"arguments": {request.Visualization.Arguments},
		"source":    {request.Visualization.Source},
		"type":      {visualizationType},
	}
	resp, err := http.PostForm(serviceURL, urlValues)
	if err != nil {
		return nil, util.Wrap(err, "Unable to initialize visualization request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("visualization service returned non-OK status: %s", resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, util.Wrap(err, "Unable to parse visualization response")
	}
	return body, nil
}
