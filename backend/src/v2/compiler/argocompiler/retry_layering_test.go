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
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	k8score "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// effectiveRetryStrategy mirrors how the Argo controller resolves a template's
// retry strategy: templateDefaults supplies a value only when the template does
// not set its own. KFP's bundled manifests configure limit 2 / OnError, so a
// template compiled without a strategy retries even though nothing in the
// compiled workflow says so.
func effectiveRetryStrategy(tmpl wfapi.Template) (limit string, policy string) {
	if tmpl.RetryStrategy == nil {
		return "2", "OnError"
	}
	limit = "<unset>"
	if tmpl.RetryStrategy.Limit != nil {
		limit = tmpl.RetryStrategy.Limit.String()
	}
	policy = string(tmpl.RetryStrategy.RetryPolicy)
	if policy == "" {
		policy = "OnError"
	}
	return limit, policy
}

func dagTemplate(name string) wfapi.Template {
	return wfapi.Template{Name: name, DAG: &wfapi.DAGTemplate{}}
}

func podTemplate(name string) wfapi.Template {
	return wfapi.Template{Name: name, Container: &k8score.Container{}}
}

func TestNeutralizeDAGRetries_DAGsStopRetryingPodsDoNot(t *testing.T) {
	wf := &wfapi.Workflow{
		Spec: wfapi.WorkflowSpec{
			Templates: []wfapi.Template{
				dagTemplate("entrypoint"),
				dagTemplate("root"),
				dagTemplate("system-container-executor"),
				podTemplate("system-dag-driver"),
				podTemplate("system-container-driver"),
				podTemplate("system-container-impl"),
			},
		},
	}

	neutralizeDAGRetries(wf)

	// Every DAG template must own an explicit zero limit, otherwise the
	// deployment default re-runs its whole subgraph on top of the task's retries.
	for _, name := range []string{"entrypoint", "root", "system-container-executor"} {
		tmpl := templateByName(t, wf, name)
		limit, _ := effectiveRetryStrategy(tmpl)
		assert.Equal(t, "0", limit, "DAG template %q must not retry", name)
	}

	// Pod templates keep inheriting the deployment default: retrying a pod
	// re-runs one container, which is what that default is for.
	for _, name := range []string{"system-dag-driver", "system-container-driver", "system-container-impl"} {
		tmpl := templateByName(t, wf, name)
		limit, policy := effectiveRetryStrategy(tmpl)
		assert.Equal(t, "2", limit, "pod template %q should keep the deployment default", name)
		assert.Equal(t, "OnError", policy, "pod template %q should keep the deployment default", name)
	}
}

func TestNeutralizeDAGRetries_PreservesAnExplicitStrategy(t *testing.T) {
	existing := &wfapi.RetryStrategy{
		Limit:       &intstr.IntOrString{Type: intstr.String, StrVal: "{{inputs.parameters.retry-max-count}}"},
		RetryPolicy: wfapi.RetryPolicyOnFailure,
	}
	impl := podTemplate("retry-system-container-impl-onfailure")
	impl.RetryStrategy = existing

	wf := &wfapi.Workflow{
		Spec: wfapi.WorkflowSpec{Templates: []wfapi.Template{dagTemplate("root"), impl}},
	}

	neutralizeDAGRetries(wf)

	limit, policy := effectiveRetryStrategy(templateByName(t, wf, "retry-system-container-impl-onfailure"))
	assert.Equal(t, "{{inputs.parameters.retry-max-count}}", limit)
	assert.Equal(t, "OnFailure", policy, "the task's selected policy must survive")
}

func templateByName(t *testing.T, wf *wfapi.Workflow, name string) wfapi.Template {
	t.Helper()
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == name {
			return tmpl
		}
	}
	t.Fatalf("template %q not found", name)
	return wfapi.Template{}
}
