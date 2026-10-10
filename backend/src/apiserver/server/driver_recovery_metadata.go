// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package server

import (
	"strings"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
)

// Legacy draft checkpoints are never exposed or promoted into trusted columns.
func publicTaskStatusMetadata(stored model.JSONData) model.JSONData {
	if stored == nil {
		return nil
	}
	result := make(model.JSONData, len(stored))
	for key, value := range stored {
		result[key] = value
	}
	properties, ok := stored["customProperties"].(map[string]interface{})
	if !ok {
		return result
	}
	filtered := make(map[string]interface{}, len(properties))
	for key, value := range properties {
		if !strings.HasPrefix(key, "_kfp_driver_") {
			filtered[key] = value
		}
	}
	result["customProperties"] = filtered
	return result
}
