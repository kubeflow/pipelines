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
	"encoding/json"
	"strings"
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argologging "github.com/argoproj/argo-workflows/v4/util/logging"
	argocommon "github.com/argoproj/argo-workflows/v4/workflow/common"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	apiserverplugins "github.com/kubeflow/pipelines/backend/src/apiserver/plugins"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestCompilePluginDriversDisableTaskRetryFlags(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	viper.Reset()
	t.Cleanup(viper.Reset)

	for _, component := range []string{"comp-container", "comp-exit-dag"} {
		t.Run(component, func(t *testing.T) {
			retry := &pipelinespec.PipelineTaskSpec_RetryPolicy{MaxRetryCount: 3}
			wf := compileDriverRetryContract(t, retry, func(spec *pipelinespec.PipelineSpec) {
				spec.Components["comp-exit-dag"] = &pipelinespec.ComponentSpec{Implementation: &pipelinespec.ComponentSpec_Dag{Dag: &pipelinespec.DagSpec{Tasks: map[string]*pipelinespec.PipelineTaskSpec{
					"cleanup": {ComponentRef: &pipelinespec.ComponentRef{Name: "comp-container"}},
				}}}}
				spec.Root.GetDag().Tasks["notify"] = exitStatusTask(component, "ordinary")
				spec.Root.GetDag().Tasks["notify"].RetryPolicy = retry
			})
			limit := intstr.FromInt32(2)
			wf.Spec.TemplateDefaults = &wfapi.Template{RetryStrategy: &wfapi.RetryStrategy{Limit: &limit, RetryPolicy: wfapi.RetryPolicyOnError}}
			before := wf.DeepCopy()
			workflow := util.NewWorkflow(wf)
			require.NoError(t, apiserverplugins.InjectPluginRuntimeEnv(workflow, []corev1.EnvVar{{Name: "PLUGIN_CONFIG", Value: "enabled"}}))
			require.NoError(t, workflow.Validate(true, false))
			assert.Equal(t, before.Spec.TemplateDefaults, wf.Spec.TemplateDefaults)

			disabled, exitDisabled := 0, 0
			for i, tmpl := range wf.Spec.Templates {
				if before.Spec.Templates[i].Metadata.Annotations[util.AnnotationKeyTaskDriverRetry] != "true" {
					assert.Equal(t, before.Spec.Templates[i].RetryStrategy, tmpl.RetryStrategy, tmpl.Name)
					continue
				}
				disabled++
				if strings.HasPrefix(tmpl.Name, "exit-") {
					exitDisabled++
				}
				assert.Nil(t, tmpl.RetryStrategy, tmpl.Name)
				assert.NotContains(t, tmpl.Metadata.Annotations, util.AnnotationKeyTaskDriverRetry)
				serialized, err := json.Marshal(tmpl)
				require.NoError(t, err)
				assert.NotContains(t, string(serialized), "{{retries}}", "disabled driver templates must not retain retry-only expressions")
				arguments := wfapi.Arguments{}
				for _, input := range tmpl.Inputs.Parameters {
					if input.Default == nil && input.Value == nil {
						arguments.Parameters = append(arguments.Parameters, wfapi.Parameter{Name: input.Name, Value: wfapi.AnyStringPtr("0")})
					}
				}
				resolved, err := argocommon.ProcessArgs(argologging.NewSlogLogger(argologging.Info, argologging.Text).NewBackgroundContext(), &tmpl, &arguments, nil, argocommon.Parameters{"retries": "7"}, false, "", nil)
				require.NoError(t, err)
				assert.Equal(t, before.Spec.Templates[i].Plugin, tmpl.Plugin, "plugin admission must not rewrite request arguments")
				args := requireDriverPluginArgs(t, resolved)
				assert.Equal(t, "false", args["driver_retry_enabled"])
				assert.Equal(t, "0", args["driver_retry_attempt"])

			}
			assert.GreaterOrEqual(t, disabled, 3, "container, DAG, and exit drivers must be checked")
			assert.Equal(t, 1, exitDisabled)
		})
	}
}
