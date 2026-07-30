/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v2beta1

import (
	"strings"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"k8s.io/apimachinery/pkg/util/validation"
)

// PipelineVersionName encapsulates the DNS-1123 validated composite or bare name for a PipelineVersion CR.
// +kubebuilder:object:generate=false
type PipelineVersionName struct {
	pipelineName string
	versionName  string
}

func NewPipelineVersionName(pipelineName, versionName string) (PipelineVersionName, error) {
	if errs := validation.IsDNS1123Subdomain(versionName); len(errs) > 0 {
		return PipelineVersionName{}, util.NewInvalidInputError(
			"Invalid pipeline version name %q: %s. Use 'display_name' for human-readable labels",
			versionName, strings.Join(errs, "; "),
		)
	}
	if pipelineName != "" {
		compositeName := pipelineName + "-" + versionName
		if errs := validation.IsDNS1123Subdomain(compositeName); len(errs) > 0 {
			return PipelineVersionName{}, util.NewInvalidInputError(
				"The combined pipeline and version name %q exceeds the Kubernetes 253-character naming limit: %s",
				compositeName, strings.Join(errs, "; "),
			)
		}
	}
	return PipelineVersionName{pipelineName: pipelineName, versionName: versionName}, nil
}

func (n PipelineVersionName) Name() string {
	if n.pipelineName != "" {
		return n.pipelineName + "-" + n.versionName
	}
	return n.versionName
}
