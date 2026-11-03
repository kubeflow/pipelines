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

package testutil

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v4/pkg/apis/workflow/v1alpha1"
	argoclient "github.com/argoproj/argo-workflows/v4/pkg/client/clientset/versioned"
	"github.com/argoproj/argo-workflows/v4/workflow/packer"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// CreateArgoClient builds an Argo Workflows clientset from the same Kubernetes
// configuration the other test clients use.
func CreateArgoClient() (argoclient.Interface, error) {
	restConfig, err := util.GetKubernetesConfig()
	if err != nil {
		return nil, err
	}
	return argoclient.NewForConfig(restConfig)
}

// RunWorkflowSelector is the label selector matching the Argo Workflow (and
// the task pods it creates) for a pipeline run.
func RunWorkflowSelector(runID string) string {
	return util.LabelKeyWorkflowRunId + "=" + runID
}

// GetWorkflowByRunID returns the Argo Workflow backing a pipeline run with its
// node status decompressed, so callers can inspect every node.
func GetWorkflowByRunID(ctx context.Context, client argoclient.Interface, namespace, runID string) (*v1alpha1.Workflow, error) {
	workflows, err := client.ArgoprojV1alpha1().Workflows(namespace).List(ctx, metav1.ListOptions{LabelSelector: RunWorkflowSelector(runID)})
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows for run %s in namespace %s: %w", runID, namespace, err)
	}
	switch len(workflows.Items) {
	case 0:
		return nil, fmt.Errorf("no workflow found for run %s in namespace %s", runID, namespace)
	case 1:
	default:
		return nil, fmt.Errorf("expected one workflow for run %s in namespace %s, found %d", runID, namespace, len(workflows.Items))
	}
	workflow := &workflows.Items[0]
	if err := packer.DecompressWorkflow(ctx, workflow); err != nil {
		return nil, fmt.Errorf("failed to decompress node status of workflow %s: %w", workflow.Name, err)
	}
	return workflow, nil
}
