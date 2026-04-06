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

	wfapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestApplyToTemplate_NilReceiver(t *testing.T) {
	tmpl := &wfapi.Template{}
	var d *driverPodConfig
	// Should not panic
	d.applyToTemplate(tmpl)
}

func TestApplyToTemplate_NilTemplate(t *testing.T) {
	d := &driverPodConfig{
		Labels: map[string]string{"app": "test"},
	}
	// Should not panic
	d.applyToTemplate(nil)
}

func TestApplyToTemplate_AddsLabelsAndAnnotations(t *testing.T) {
	d := &driverPodConfig{
		Labels:      map[string]string{"app": "driver", "team": "ml"},
		Annotations: map[string]string{"proxy.istio.io/config": "value"},
	}
	tmpl := &wfapi.Template{}
	d.applyToTemplate(tmpl)

	assert.Equal(t, "driver", tmpl.Metadata.Labels["app"])
	assert.Equal(t, "ml", tmpl.Metadata.Labels["team"])
	assert.Equal(t, "value", tmpl.Metadata.Annotations["proxy.istio.io/config"])
}

func TestApplyToTemplate_DoesNotOverwriteExistingLabels(t *testing.T) {
	d := &driverPodConfig{
		Labels:      map[string]string{"system-label": "admin-value", "new-label": "admin"},
		Annotations: map[string]string{"system-annotation": "admin-value", "new-annotation": "admin"},
	}
	tmpl := &wfapi.Template{
		Metadata: wfapi.Metadata{
			Labels:      map[string]string{"system-label": "system-value"},
			Annotations: map[string]string{"system-annotation": "system-value"},
		},
	}
	d.applyToTemplate(tmpl)

	// Existing system labels must NOT be overwritten by admin config
	assert.Equal(t, "system-value", tmpl.Metadata.Labels["system-label"])
	assert.Equal(t, "system-value", tmpl.Metadata.Annotations["system-annotation"])
	// New labels from admin config should be added
	assert.Equal(t, "admin", tmpl.Metadata.Labels["new-label"])
	assert.Equal(t, "admin", tmpl.Metadata.Annotations["new-annotation"])
}

func TestApplyToTemplate_OnlyLabelsNoAnnotations(t *testing.T) {
	d := &driverPodConfig{
		Labels: map[string]string{"app": "test"},
	}
	tmpl := &wfapi.Template{}
	d.applyToTemplate(tmpl)

	assert.Equal(t, "test", tmpl.Metadata.Labels["app"])
	assert.Nil(t, tmpl.Metadata.Annotations)
}

func TestApplyToTemplate_OnlyAnnotationsNoLabels(t *testing.T) {
	d := &driverPodConfig{
		Annotations: map[string]string{"note": "value"},
	}
	tmpl := &wfapi.Template{}
	d.applyToTemplate(tmpl)

	assert.Nil(t, tmpl.Metadata.Labels)
	assert.Equal(t, "value", tmpl.Metadata.Annotations["note"])
}

func TestGetDriverPodConfig_NilWhenEmpty(t *testing.T) {
	// getDriverPodConfig should return nil when no labels/annotations are configured
	// This test relies on common.GetDriverPodLabels/GetDriverPodAnnotations returning nil
	// when not initialized, which is the default state in tests
	config := getDriverPodConfig()
	assert.Nil(t, config)
}
