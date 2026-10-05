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

package util

// AnnotationKeyDriverRetryFinalizer identifies generation-specific retry hooks
// whose completed nodes must be replayed after a manual run retry.
const AnnotationKeyDriverRetryFinalizer = "pipelines.kubeflow.org/driver-retry-finalizer"

// These template inputs let admission-time workflow configuration disable
// task-derived driver recovery without depending on the driver's CLI spelling.
// They are template defaults only: DAG task arguments must never override them.
const (
	DriverRetryEnabledParameter = "driver-retry-enabled"
	DriverRetryAttemptParameter = "driver-retry-attempt"
	// DriverRetryGenerationEnv is resolved when each runtime pod is created,
	// keeping its API writes bound to that manual run retry generation.
	DriverRetryGenerationEnv = "KFP_DRIVER_RETRY_GENERATION"
)
