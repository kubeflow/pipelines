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

package api

// DriverHTTPResponse is the JSON body returned by the KFP driver HTTP endpoint
// (POST /api/v1/template.execute) when invoked via an Argo HTTP template.
// All fields correspond to Argo workflow output parameters extracted by the
// compiler using jsonpath(response.body, "$.FIELD") expressions.
type DriverHTTPResponse struct {
	ExecutionID    string `json:"execution-id,omitempty"`
	IterationCount string `json:"iteration-count,omitempty"`
	CachedDecision string `json:"cached-decision,omitempty"`
	Condition      string `json:"condition,omitempty"`
	PodSpecPatch   string `json:"pod-spec-patch,omitempty"`
}
