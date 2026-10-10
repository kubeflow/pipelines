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

import (
	"fmt"
	"strings"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/src/v2/driver/driverflags"
)

const paramDriverRetryStatus = "driver-retry-status"

func validateDriverRetryPolicy(policy *pipelinespec.PipelineTaskSpec_RetryPolicy) error {
	if policy == nil {
		return nil
	}
	if policy.GetMaxRetryCount() < 0 {
		return fmt.Errorf("retry max count must be non-negative, got %d", policy.GetMaxRetryCount())
	}
	if _, supported := pipelinespec.PipelineTaskSpec_RetryPolicy_Policy_name[int32(policy.GetPolicy())]; !supported {
		return fmt.Errorf("unsupported retry policy %d; use Always, OnFailure, OnError, or OnTransientError", policy.GetPolicy())
	}
	return nil
}

// addTaskRetryDriverTemplate keeps driver and executor retry budgets separate.
// Only tasks with an explicit or inherited policy use the retrying variant.
func (c *workflowCompiler) addTaskRetryDriverTemplate(baseName string, task *pipelinespec.PipelineTaskSpec) string {
	if task.GetRetryPolicy() == nil {
		return baseName
	}
	policy := protoRetryPolicyToArgo(task.GetRetryPolicy().GetPolicy())
	if policy == "" {
		// A driver reports API errors by exiting unsuccessfully. OnError alone
		// would retry infrastructure errors but omit these recoverable failures.
		policy = string(wfapi.RetryPolicyAlways)
	}
	name := "retry-" + baseName + "-" + strings.ToLower(policy)
	if _, exists := c.templates[name]; exists {
		return name
	}

	template := c.templates[baseName].DeepCopy()
	template.Name = name
	template.Inputs.Parameters = append(template.Inputs.Parameters, []wfapi.Parameter{
		{Name: util.DriverRetryEnabledParameter, Default: wfapi.AnyStringPtr("true")},
		{Name: util.DriverRetryAttemptParameter, Default: wfapi.AnyStringPtr("{{retries}}")},
		{Name: paramRetryMaxCount, Default: wfapi.AnyStringPtr("0")},
		{Name: paramRetryBackOffDuration, Default: wfapi.AnyStringPtr("0")},
		{Name: paramRetryBackOffFactor, Default: wfapi.AnyStringPtr("2")},
		{Name: paramRetryBackOffMaxDuration, Default: wfapi.AnyStringPtr("3600")},
	}...)
	template.RetryStrategy = c.getTaskRetryStrategyFromInput(
		inputParameter(paramRetryMaxCount),
		inputParameter(paramRetryBackOffDuration),
		inputParameter(paramRetryBackOffFactor),
		inputParameter(paramRetryBackOffMaxDuration),
		policy,
	)
	template.Container.Args = append(template.Container.Args,
		"--"+driverflags.DriverRetryEnabledArg+"="+inputValue(util.DriverRetryEnabledParameter),
		"--"+driverflags.DriverRetryAttemptArg, inputValue(util.DriverRetryAttemptParameter),
		"--driver_retry_max_count", inputValue(paramRetryMaxCount),
		"--driver_retry_generation", "{{workflow.annotations."+util.AnnotationKeyRetryGeneration+"}}",
	)
	template.Metadata.Annotations[util.AnnotationKeyTaskDriverRetry] = "true"
	template.Metadata.Annotations[systemTemplateNameAnnotationKey] = name

	c.templates[name] = template
	c.wf.Spec.Templates = append(c.wf.Spec.Templates, *template)
	return name
}

func (c *workflowCompiler) getDriverRetryParametersWithValues(task *pipelinespec.PipelineTaskSpec) []wfapi.Parameter {
	parameters := c.getTaskRetryParametersWithValues(task)
	// Unlike durations, the proto's scalar factor cannot distinguish omitted
	// from zero. Match the SDK and the documented default in either case.
	if task.GetRetryPolicy().GetBackoffFactor() == 0 {
		for i := range parameters {
			if parameters[i].Name == paramRetryBackOffFactor {
				parameters[i].Value = wfapi.AnyStringPtr("2")
			}
		}
	}
	return parameters
}

// configureDriverRetryFinalizer runs after the retry node becomes terminal,
// including when its policy or backoff deadline stops retries before the limit.
// Successful drivers hand off directly to their executor without a finalizer.
func (c *workflowCompiler) configureDriverRetryFinalizer(task *wfapi.DAGTask) {
	base := c.templates[task.Template]
	if base.Metadata.Annotations[util.AnnotationKeyTaskDriverRetry] != "true" {
		return
	}
	name := "finalize-" + task.Template
	if _, exists := c.templates[name]; !exists {
		tmpl := base.DeepCopy()
		tmpl.Name = name
		tmpl.Metadata.Annotations[systemTemplateNameAnnotationKey] = name
		tmpl.Metadata.Annotations[util.AnnotationKeyDriverRetryFinalizer] = "true"
		tmpl.RetryStrategy = nil
		// Finalizers publish no driver outputs. Remove their path references
		// from the inherited arguments before clearing the output declarations.
		for _, output := range tmpl.Outputs.Parameters {
			for i, arg := range tmpl.Container.Args {
				tmpl.Container.Args[i] = strings.ReplaceAll(arg, outputPath(output.Name), "")
			}
		}
		tmpl.Outputs = wfapi.Outputs{}
		// A hook is outside the retry node, so {{retries}} is unavailable.
		// The API finalizes the current private owner under the generation fence.
		for i := range tmpl.Inputs.Parameters {
			if tmpl.Inputs.Parameters[i].Name == util.DriverRetryAttemptParameter {
				tmpl.Inputs.Parameters[i].Default = wfapi.AnyStringPtr("0")
			}
		}
		tmpl.Inputs.Parameters = append(tmpl.Inputs.Parameters, wfapi.Parameter{Name: paramDriverRetryStatus})
		tmpl.Container.Args = append(tmpl.Container.Args,
			"--"+driverflags.DriverRetryFinalizeArg+"=true",
			"--"+driverflags.DriverRetryStatusArg, inputValue(paramDriverRetryStatus),
		)
		c.templates[name] = tmpl
		c.wf.Spec.Templates = append(c.wf.Spec.Templates, *tmpl)
	}
	arguments := task.Arguments.DeepCopy()
	arguments.Parameters = append(arguments.Parameters, wfapi.Parameter{
		Name: paramDriverRetryStatus, Value: wfapi.AnyStringPtr("{{tasks." + task.Name + ".status}}"),
	})
	if task.Hooks == nil {
		task.Hooks = wfapi.LifecycleHooks{}
	}
	task.Hooks[wfapi.ExitLifecycleEvent] = wfapi.LifecycleHook{
		Template: name, Arguments: *arguments,
		Expression: "tasks['" + task.Name + "'].status in ['Failed', 'Error']",
	}
}
