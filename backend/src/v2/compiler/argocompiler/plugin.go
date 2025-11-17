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

func driverPlugin(params map[string]interface{}) (*wfapi.Plugin, error) {
	pluginConfig := map[string]interface{}{
		"driver-plugin": map[string]interface{}{
			"args": params,
		},
	}
	jsonConfig, err := json.Marshal(pluginConfig)
	if err != nil {
		return nil, fmt.Errorf("driver plugin creation error: marshaling plugin config to JSON failed: %w", err)
	}
	return &wfapi.Plugin{Object: wfapi.Object{
		Value: jsonConfig,
	}}, nil
}
