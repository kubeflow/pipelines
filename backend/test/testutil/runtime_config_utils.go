// Copyright 2026 The Kubeflow Authors
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

package testutil

import (
	"os"
)

const useCoordinatorRuntimeForTestsEnvVar = "KFP_USE_COORDINATOR_RUNTIME"

// WithCoordinatorRuntimeSelector returns a copy of the input parameters.
// Coordinator runtime selection is now server-wide when enabled, so tests no
// longer need to inject an internal runtime selector.
func WithCoordinatorRuntimeSelector(parameters map[string]interface{}) map[string]interface{} {
	return parameters
}

// ShouldUseCoordinatorRuntimeSelector returns true when shared tests should run
// against an API server configured for the coordinator runtime.
func ShouldUseCoordinatorRuntimeSelector() bool {
	return os.Getenv(useCoordinatorRuntimeForTestsEnvVar) == "true"
}

// MaybeWithCoordinatorRuntimeSelector returns the parameters unchanged. It is
// kept for compatibility with shared test helpers that historically injected a
// hidden runtime selector.
func MaybeWithCoordinatorRuntimeSelector(parameters map[string]interface{}) map[string]interface{} {
	return parameters
}
