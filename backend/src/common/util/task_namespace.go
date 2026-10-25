// Copyright 2026 The Kubeflow Authors
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

package util

import "strings"

// ValidateTaskPipelineNamespace checks a namespaced pipeline name against the
// task namespace. Call again after resolving an omitted task namespace.
func ValidateTaskPipelineNamespace(pipelineName, namespace string) error {
	if !strings.HasPrefix(pipelineName, "namespace/") {
		return nil
	}
	parts := strings.SplitN(pipelineName, "/", 4)
	if len(parts) != 4 {
		return NewInvalidInputError("invalid PipelineName for namespaced pipelines, need to follow 'namespace/${namespace}/pipeline/${pipelineName}': %s", pipelineName)
	}
	if namespace != "" && parts[1] != namespace {
		return NewInvalidInputError("the namespace %s extracted from pipelineName is not equal to the namespace %s in task", parts[1], namespace)
	}
	return nil
}
