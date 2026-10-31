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

package plugins

import "fmt"

// ValidateDriverRetry rejects dispatchers whose hooks cannot safely be replayed.
// Callers must also propagate dispatcher initialization errors instead of
// replacing a failed plugin with a no-op dispatcher.
func ValidateDriverRetry(dispatcher TaskPluginDispatcher) error {
	switch typed := dispatcher.(type) {
	case nil, NoOpDispatcher:
		return nil
	case *NoOpDispatcher:
		if typed != nil {
			return nil
		}
	case *TaskPluginDispatcherImpl:
		if typed != nil && len(typed.handlers) == 0 {
			return nil
		}
	}
	return fmt.Errorf("task-derived driver retries cannot be used with task plugins that do not support replay; disable driver retries or submit the run through the KFP API server")
}
