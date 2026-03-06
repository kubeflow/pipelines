// Copyright 2025 The Kubeflow Authors
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

package argocompiler

import (
	"testing"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/config/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyDriverPodConfig_NilConfig(t *testing.T) {
	tmpl := &wfapi.Template{}
	var d *common.DriverPodConfig
	// Should not panic
	applyDriverPodConfig(d, tmpl)
}

func TestApplyDriverPodConfig_NilTemplate(t *testing.T) {
	d := &common.DriverPodConfig{
		Labels: map[string]string{"app": "test"},
	}
	// Should not panic
	applyDriverPodConfig(d, nil)
}

func TestApplyDriverPodConfig_AddsLabelsAndAnnotations(t *testing.T) {
	d := &common.DriverPodConfig{
		Labels:      map[string]string{"app": "driver", "team": "ml"},
		Annotations: map[string]string{"proxy.istio.io/config": "value"},
	}
	tmpl := &wfapi.Template{}
	applyDriverPodConfig(d, tmpl)

	assert.Equal(t, "driver", tmpl.Metadata.Labels["app"])
	assert.Equal(t, "ml", tmpl.Metadata.Labels["team"])
	assert.Equal(t, "value", tmpl.Metadata.Annotations["proxy.istio.io/config"])
}

func TestApplyDriverPodConfig_DoesNotOverwriteExistingLabels(t *testing.T) {
	d := &common.DriverPodConfig{
		Labels:      map[string]string{"system-label": "admin-value", "new-label": "admin"},
		Annotations: map[string]string{"system-annotation": "admin-value", "new-annotation": "admin"},
	}
	tmpl := &wfapi.Template{
		Metadata: wfapi.Metadata{
			Labels:      map[string]string{"system-label": "system-value"},
			Annotations: map[string]string{"system-annotation": "system-value"},
		},
	}
	applyDriverPodConfig(d, tmpl)

	// Existing system labels must NOT be overwritten by admin config
	assert.Equal(t, "system-value", tmpl.Metadata.Labels["system-label"])
	assert.Equal(t, "system-value", tmpl.Metadata.Annotations["system-annotation"])
	// New labels from admin config should be added
	assert.Equal(t, "admin", tmpl.Metadata.Labels["new-label"])
	assert.Equal(t, "admin", tmpl.Metadata.Annotations["new-annotation"])
}

func TestApplyDriverPodConfig_OnlyLabelsNoAnnotations(t *testing.T) {
	d := &common.DriverPodConfig{
		Labels: map[string]string{"app": "test"},
	}
	tmpl := &wfapi.Template{}
	applyDriverPodConfig(d, tmpl)

	assert.Equal(t, "test", tmpl.Metadata.Labels["app"])
	assert.Nil(t, tmpl.Metadata.Annotations)
}

func TestApplyDriverPodConfig_OnlyAnnotationsNoLabels(t *testing.T) {
	d := &common.DriverPodConfig{
		Annotations: map[string]string{"note": "value"},
	}
	tmpl := &wfapi.Template{}
	applyDriverPodConfig(d, tmpl)

	assert.Nil(t, tmpl.Metadata.Labels)
	assert.Equal(t, "value", tmpl.Metadata.Annotations["note"])
}

// TestContainerDriverTemplate_IncludesDriverPodConfig verifies the full compilation
// path: a compiler configured with driver pod labels/annotations produces a container
// driver template whose metadata carries them. This guards the integration point in
// addContainerDriverTemplate, which unit tests of applyDriverPodConfig cannot.
func TestContainerDriverTemplate_IncludesDriverPodConfig(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	c := &workflowCompiler{
		templates: make(map[string]*wfapi.Template),
		wf: &wfapi.Workflow{
			Spec: wfapi.WorkflowSpec{
				Templates: []wfapi.Template{},
			},
		},
		spec: &pipelinespec.PipelineSpec{
			PipelineInfo: &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		},
		job: &pipelinespec.PipelineJob{},
		driverPodConfig: &common.DriverPodConfig{
			Labels:      map[string]string{"sidecar.istio.io/inject": "true"},
			Annotations: map[string]string{"proxy.istio.io/config": "hold"},
		},
	}

	name := c.addContainerDriverTemplate()
	tmpl, exists := c.templates[name]
	require.True(t, exists, "system-container-driver template should exist")
	require.NotNil(t, tmpl)
	assert.Equal(t, "true", tmpl.Metadata.Labels["sidecar.istio.io/inject"])
	assert.Equal(t, "hold", tmpl.Metadata.Annotations["proxy.istio.io/config"])
}

// TestDAGDriverTemplate_IncludesDriverPodConfig mirrors the container driver check for
// the DAG driver template, guarding the integration point in addDAGDriverTemplate.
func TestDAGDriverTemplate_IncludesDriverPodConfig(t *testing.T) {
	proxy.InitializeConfigWithEmptyForTests()
	c := &workflowCompiler{
		templates: make(map[string]*wfapi.Template),
		wf: &wfapi.Workflow{
			Spec: wfapi.WorkflowSpec{
				Templates: []wfapi.Template{},
			},
		},
		spec: &pipelinespec.PipelineSpec{
			PipelineInfo: &pipelinespec.PipelineInfo{Name: "test-pipeline"},
		},
		job: &pipelinespec.PipelineJob{},
		driverPodConfig: &common.DriverPodConfig{
			Labels:      map[string]string{"sidecar.istio.io/inject": "true"},
			Annotations: map[string]string{"proxy.istio.io/config": "hold"},
		},
	}

	name := c.addDAGDriverTemplate()
	tmpl, exists := c.templates[name]
	require.True(t, exists, "system-dag-driver template should exist")
	require.NotNil(t, tmpl)
	assert.Equal(t, "true", tmpl.Metadata.Labels["sidecar.istio.io/inject"])
	assert.Equal(t, "hold", tmpl.Metadata.Annotations["proxy.istio.io/config"])
}
