// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package resolver

import (
	"fmt"

	apiV2beta1 "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
)

func findParameterByIOKey(
	key string,
	pms []ParameterMetadata,
) (*apiV2beta1.PipelineTaskDetail_InputOutputs_IOParameter, error) {
	for _, pm := range pms {
		if pm.ParameterIO.GetParameterKey() == key {
			return pm.ParameterIO, nil
		}
	}
	return nil, fmt.Errorf("parameter not found")
}

func findArtifactByIOKey(
	key string,
	ams []ArtifactMetadata,
) (*apiV2beta1.PipelineTaskDetail_InputOutputs_IOArtifact, error) {
	for _, am := range ams {
		if am.Key == key {
			return am.ArtifactIO, nil
		}
	}
	return nil, fmt.Errorf("artifact not found")
}
