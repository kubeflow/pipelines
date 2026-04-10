// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argocompiler

import (
	"encoding/json"
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// driverEndpointURL is the in-cluster URL for the standalone KFP driver
// service. Argo HTTP templates POST to this URL for every driver node,
// eliminating per-workflow sidecar or agent pods.
const driverEndpointURL = "http://ml-pipeline-driver.kubeflow.svc.cluster.local:8080/api/v1/template.execute"

// driverHTTPTemplate builds an Argo HTTP template that invokes the centralized
// KFP driver endpoint. The params map is marshaled as the JSON request body;
// Argo performs template-variable substitution on the body at runtime.
//
// The successCondition ensures Argo marks the node as failed when the driver
// returns a non-200 status (e.g. on driver logic errors).
func driverHTTPTemplate(params map[string]interface{}) (*wfapi.HTTP, error) {
	bodyJSON, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("driver HTTP template: marshaling args to JSON failed: %w", err)
	}
	timeout := int64(600)
	return &wfapi.HTTP{
		Method:           "POST",
		URL:              driverEndpointURL,
		Headers:          wfapi.HTTPHeaders{{Name: "Content-Type", Value: "application/json"}},
		Body:             string(bodyJSON),
		SuccessCondition: "response.statusCode == 200",
		TimeoutSeconds:   &timeout,
	}, nil
}
