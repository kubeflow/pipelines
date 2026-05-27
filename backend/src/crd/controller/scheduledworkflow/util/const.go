// Copyright 2018 The Kubeflow Authors
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

package util

import (
	"time"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
)

const (
	ControllerAgentName string = "scheduled-workflow-controller"       // ControllerAgentName is the name of the controller.
	TimeZone            string = "CRON_SCHEDULE_TIMEZONE"              // TimeZone is the name of the cron schedule timezone env parameter
	V2Key               string = "pipelines.kubeflow.org/v2_component" // V2Key is the name of the v2 component labels
	V2PipelineKey       string = "pipelines.kubeflow.org/v2_pipeline"  // V2PipelineKey is the name of the v2 pipeline label
	BlockV1Pipelines    string = "BLOCK_V1_PIPELINES"                  // BlockV1Pipelines is the name of the env parameter to block v1 pipelines
	AllowedNamespaces   string = "V1_ALLOWED_NAMESPACES"               // AllowedNamespaces is the name of the env parameter to specify allowed namespaces for v1 pipelines
)

func GetLocation() (*time.Location, error) {
	locString := common.GetStringConfigWithDefault(TimeZone, "")
	if locString == "" {
		return time.Now().Location(), nil
	}
	return time.LoadLocation(locString)
}
