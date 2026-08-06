// Copyright 2023 The Kubeflow Authors
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
	"encoding/json"
	"fmt"

	wfapi "github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	"github.com/kubeflow/pipelines/kubernetes_platform/go/kubernetesplatform"
	log "github.com/sirupsen/logrus"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
)

func (c *workflowCompiler) AddKubernetesSpec(name string, kubernetesSpec *structpb.Struct) error {
	k8sExecConfig, err := unmarshalKubernetesExecutorConfig(name, kubernetesSpec)
	if err != nil {
		return err
	}
	if err := c.validateKubernetesSecurityContext(name, k8sExecConfig); err != nil {
		return err
	}
	c.kubernetesConfigs[name] = k8sExecConfig
	return c.saveKubernetesSpec(name, kubernetesSpec)
}

func unmarshalKubernetesExecutorConfig(name string, kubernetesSpec *structpb.Struct) (*kubernetesplatform.KubernetesExecutorConfig, error) {
	if kubernetesSpec == nil {
		return &kubernetesplatform.KubernetesExecutorConfig{}, nil
	}
	jsonBytes, err := kubernetesSpec.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal kubernetes spec for component %q: %w", name, err)
	}
	config := &kubernetesplatform.KubernetesExecutorConfig{}
	if err := protojson.Unmarshal(jsonBytes, config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal kubernetes spec for component %q: %w", name, err)
	}
	return config, nil
}

func (c *workflowCompiler) validateKubernetesSecurityContext(name string, config *kubernetesplatform.KubernetesExecutorConfig) error {
	securityContext := config.GetSecurityContext()
	if securityContext == nil {
		return nil
	}
	if c.defaultRunAsUser != nil {
		return nil
	}
	if securityContext.RunAsUser != nil && *securityContext.RunAsUser == 0 {
		return fmt.Errorf(
			"runAsUser=0 (root) is not allowed for component %q: "+
				"the container security context enforces non-root execution "+
				"(allowPrivilegeEscalation=false); use a non-root UID instead",
			name,
		)
	}
	return nil
}

// ApplyWorkflowSpecPatch applies a JSON patch to the compiled workflow specification.
// It validates the JSON and applies it using Kubernetes strategic merge patch.
// Only the workflow's "spec" field can be patched for security reasons.
func (c *workflowCompiler) ApplyWorkflowSpecPatch(patchJSON string) error {
	if c.wf == nil {
		return fmt.Errorf("workflow is nil")
	}

	// Check for empty patch string
	if patchJSON == "" {
		log.Debug("Empty workflow spec patch string provided, skipping patching")
		return nil
	}

	log.Debug("Applying workflow spec patch")

	// Validate that the patch is valid JSON by attempting to unmarshal it
	var specPatchValidation map[string]interface{}
	if err := json.Unmarshal([]byte(patchJSON), &specPatchValidation); err != nil {
		return fmt.Errorf("invalid JSON in COMPILED_PIPELINE_SPEC_PATCH: %w", err)
	}

	// Check if the patch is empty (no fields to patch)
	if len(specPatchValidation) == 0 {
		log.Debug("Empty workflow spec patch provided, skipping patching")
		return nil
	}

	// Convert the current workflow spec to JSON
	originalSpecJSON, err := json.Marshal(c.wf.Spec)
	if err != nil {
		return fmt.Errorf("failed to marshal workflow spec to JSON: %w", err)
	}

	// Apply the strategic merge patch to the spec directly
	patchedSpecJSON, err := strategicpatch.StrategicMergePatch(originalSpecJSON, []byte(patchJSON), wfapi.WorkflowSpec{})
	if err != nil {
		return fmt.Errorf("failed to apply strategic merge patch to workflow spec: %w", err)
	}

	// Unmarshal the patched spec back into the workflow
	if err := json.Unmarshal(patchedSpecJSON, &c.wf.Spec); err != nil {
		return fmt.Errorf("failed to unmarshal patched workflow spec: %w", err)
	}

	log.Debug("Successfully applied workflow spec patch")
	return nil
}
