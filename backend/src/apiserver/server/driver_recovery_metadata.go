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
	"context"

	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc/metadata"
)

// Only single-task RPCs honor this projection. Run/list/bulk responses must not
// multiply durable replay payloads by the number of tasks in a fan-out.
func driverRecoveryView(ctx context.Context) string {
	md, _ := metadata.FromIncomingContext(ctx)
	values := md.Get(util.DriverRecoveryViewHeader)
	if len(values) == 1 {
		return values[0]
	}
	return ""
}

func taskStatusMetadataForView(stored model.JSONData, view string) model.JSONData {
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
		switch key {
		case util.DriverCheckpointKey, util.DriverCachedOutputsKey:
			if view != util.DriverRecoveryViewFull {
				continue
			}
		case util.DriverRetryGenerationKey, util.DriverRetryAttemptKey:
			if view != util.DriverRecoveryViewFull && view != util.DriverRecoveryViewOwnership {
				continue
			}
		case util.DriverRetrySourceTaskKey, util.DriverRetrySourceAttemptKey:
			continue // transient request authority is never response metadata
		}
		filtered[key] = value
	}
	result["customProperties"] = filtered
	return result
}
