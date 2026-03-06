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

package plugins

import "context"

// TaskInfo holds task-level information passed to OnTaskStart/OnTaskEnd hooks.
type TaskInfo struct{}

// TaskStartResult holds the result of an OnTaskStart hook.
type TaskStartResult struct{}

// TaskPluginHandler defines the task-level plugin hooks invoked by the driver
// and launcher.
type TaskPluginHandler interface {
	OnTaskStart(ctx context.Context, taskInfo TaskInfo, config interface{}) (*TaskStartResult, error)
	OnTaskEnd(ctx context.Context, taskInfo TaskInfo, metrics map[string]float64, params map[string]string, config interface{}) error
}
