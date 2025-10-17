// Copyright 2025 The Kubeflow Authors
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

package validation

import (
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// ValidateNamespaceRequired validates that a namespace is provided when required by configuration.
// This ensures consistent authorization behavior across all KFP resources.
// Only validates in multi-user mode, since single-user mode always uses empty namespaces by design.
func ValidateNamespaceRequired(namespace string) error {
	if common.IsMultiUserMode() && common.IsNamespaceRequiredForPipelines() && namespace == "" {
		return util.NewInvalidInputError("Namespace is required for pipeline operations")
	}
	return nil
}
