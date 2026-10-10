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

package argocompiler

import wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"

const (
	paramExitTaskName   = "exit-task-name"
	paramExitTaskStatus = "exit-task-status"
)

// configureExitDriver forwards the controller's terminal task phase to the
// exit handler, which can start before native task status reconciliation.
func (c *workflowCompiler) configureExitDriver(task *wfapi.DAGTask, producerName, producerStatus string) {
	if producerName == "" {
		return
	}
	name := "exit-" + task.Template
	if _, exists := c.templates[name]; !exists {
		template := c.templates[task.Template].DeepCopy()
		template.Name = name
		template.Inputs.Parameters = append(template.Inputs.Parameters,
			wfapi.Parameter{Name: paramExitTaskName}, wfapi.Parameter{Name: paramExitTaskStatus})
		template.Container.Args = append(template.Container.Args,
			"--exit_task_name", inputParameter(paramExitTaskName),
			"--exit_task_status", inputParameter(paramExitTaskStatus))
		template.Metadata.Annotations[systemTemplateNameAnnotationKey] = name
		c.templates[name] = template
		c.wf.Spec.Templates = append(c.wf.Spec.Templates, *template)
	}
	task.Template = name
	task.Arguments.Parameters = append(task.Arguments.Parameters,
		wfapi.Parameter{Name: paramExitTaskName, Value: wfapi.AnyStringPtr(producerName)},
		wfapi.Parameter{Name: paramExitTaskStatus, Value: wfapi.AnyStringPtr(producerStatus)})
}

func setParameterValue(parameters []wfapi.Parameter, name, value string) []wfapi.Parameter {
	for i := range parameters {
		if parameters[i].Name == name {
			parameters[i].Value = wfapi.AnyStringPtr(value)
			return parameters
		}
	}
	return append(parameters, wfapi.Parameter{Name: name, Value: wfapi.AnyStringPtr(value)})
}
