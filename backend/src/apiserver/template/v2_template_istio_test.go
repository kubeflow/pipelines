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

package template

import (
	"os"
	"strings"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The compiled-template default decides whether a workflow pod joins the mesh at all, so an
// unrecognised value must fall back to upstream behaviour rather than to injection.
func TestIstioSidecarInjectDefault(t *testing.T) {
	previous := viper.Get(WorkflowIstioSidecarInject)
	defer viper.Set(WorkflowIstioSidecarInject, previous)

	for _, test := range []struct {
		name  string
		value string
		want  string
	}{
		{"enabled", "true", util.AnnotationValueIstioSidecarInjectEnabled},
		{"explicitly disabled", "false", util.AnnotationValueIstioSidecarInjectDisabled},
		{"empty", "", util.AnnotationValueIstioSidecarInjectDisabled},
		{"wrong case", "TRUE", util.AnnotationValueIstioSidecarInjectDisabled},
		{"not a boolean", "mesh", util.AnnotationValueIstioSidecarInjectDisabled},
	} {
		t.Run(test.name, func(t *testing.T) {
			viper.Set(WorkflowIstioSidecarInject, test.value)
			assert.Equal(t, test.want, istioSidecarInjectDefault())
		})
	}
}

// A run and a recurring run that disagree would put half a pipeline outside the mesh, and the
// meshed half would then fail under STRICT mTLS with no message naming injection. Both call sites
// are asserted in the source, because reaching them through the compiler needs a full pipeline
// fixture and would not fail on the case that matters: a rebase re-hardcoding one of the two.
func TestIstioSidecarInjectDefaultIsUsedByBothCompilationPaths(t *testing.T) {
	source, err := os.ReadFile("v2_template.go")
	require.NoError(t, err)
	call := "SetAnnotationsToAllTemplatesIfKeyNotExist(util.AnnotationKeyIstioSidecarInject, "
	assert.Equal(t, 2, strings.Count(string(source), call+"istioSidecarInjectDefault())"),
		"both the run and the recurring-run compilation paths must read the configured default")
	assert.NotContains(t, string(source),
		call+"util.AnnotationValueIstioSidecarInjectDisabled)",
		"no compilation path may hardcode the injection default")
}
