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
)

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
		"--driver_retry_enabled=true",
		"--driver_retry_attempt", "{{retries}}",
		"--driver_retry_max_count", inputValue(paramRetryMaxCount),
		"--driver_retry_generation", "{{workflow.annotations."+util.AnnotationKeyRetryGeneration+"}}",
	)
	template.Metadata.Annotations[util.AnnotationKeyTaskDriverRetry] = "true"
	template.Metadata.Annotations[systemTemplateNameAnnotationKey] = name

	if c.wf.Annotations == nil {
		c.wf.Annotations = make(map[string]string)
	}
	// RetryRun replaces this generation before an explicit retry. Automatic
	// driver retries retain it and therefore reuse their output allocations.
	c.wf.Annotations[util.AnnotationKeyRetryGeneration] = "0"
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
