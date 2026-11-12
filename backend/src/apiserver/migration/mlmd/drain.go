// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mlmd

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var workflowResource = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "workflows"}

// DrainGate checks the maintenance cutover immediately before Phase 2/3.
// PodName is empty for the CLI and the current API pod name for startup.
type DrainGate struct {
	Database            *sql.DB
	Kubernetes          kubernetes.Interface
	Workflows           dynamic.Interface
	Namespace           string
	Deployment          string
	PodName             string
	RecurringDeployment string
}

// Check refuses migration until the old writer is fenced and final reports
// have been stored. Listing failures are fatal: they cannot prove quiescence.
func (gate DrainGate) Check(ctx context.Context) error {
	if gate.Database == nil || gate.Kubernetes == nil || gate.Workflows == nil || gate.Namespace == "" || gate.Deployment == "" {
		return fmt.Errorf("MLMD drain gate requires database and Kubernetes clients, namespace, and deployment name")
	}
	if err := gate.CheckDrained(ctx); err != nil {
		return err
	}
	recurringName := gate.RecurringDeployment
	if recurringName == "" {
		recurringName = "ml-pipeline-scheduledworkflow"
	}
	recurring, err := gate.Kubernetes.AppsV1().Deployments(gate.Namespace).Get(ctx, recurringName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("inspect recurring-run controller for MLMD maintenance: %w", err)
	}
	if recurring.Spec.Replicas == nil || *recurring.Spec.Replicas != 0 || recurring.Status.ObservedGeneration < recurring.Generation {
		return fmt.Errorf("recurring-run controller must be scaled to zero before MLMD migration")
	}
	recurringSelector, err := metav1.LabelSelectorAsSelector(recurring.Spec.Selector)
	if err != nil {
		return fmt.Errorf("invalid recurring-run controller selector: %w", err)
	}
	recurringPods, err := gate.Kubernetes.CoreV1().Pods(gate.Namespace).List(ctx, metav1.ListOptions{LabelSelector: recurringSelector.String()})
	if err != nil {
		return fmt.Errorf("list recurring-run controller pods for MLMD maintenance: %w", err)
	}
	if len(recurringPods.Items) != 0 {
		return fmt.Errorf("recurring-run controller pods are still running; wait before MLMD migration")
	}
	deployment, err := gate.Kubernetes.AppsV1().Deployments(gate.Namespace).Get(ctx, gate.Deployment, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("inspect API Deployment for MLMD writer fence: %w", err)
	}
	if deployment.Status.ObservedGeneration < deployment.Generation {
		return fmt.Errorf("API Deployment has not observed its writer-fence generation")
	}
	if deployment.Status.Replicas != deployment.Status.UpdatedReplicas {
		return fmt.Errorf("API Deployment rollout is incomplete; old replicas may restart")
	}
	selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
	if err != nil {
		return fmt.Errorf("invalid API Deployment selector: %w", err)
	}
	pods, err := gate.Kubernetes.CoreV1().Pods(gate.Namespace).List(ctx, metav1.ListOptions{LabelSelector: selector.String()})
	if err != nil {
		return fmt.Errorf("list API pods for MLMD writer fence: %w", err)
	}
	if gate.PodName == "" {
		if deployment.Spec.Replicas == nil || *deployment.Spec.Replicas != 0 || len(pods.Items) != 0 {
			return fmt.Errorf("old API Deployment must be scaled to zero with no remaining pods before CLI migration")
		}
	} else {
		var ownImageID, ownImage string
		for _, pod := range pods.Items {
			if pod.Name == gate.PodName && pod.DeletionTimestamp == nil {
				for _, container := range pod.Spec.Containers {
					if container.Name == "ml-pipeline-api-server" {
						ownImage = container.Image
					}
				}
				for _, container := range pod.Status.ContainerStatuses {
					if container.Name == "ml-pipeline-api-server" {
						ownImageID = container.ImageID
					}
				}
			}
		}
		if ownImageID == "" || ownImage == "" {
			return fmt.Errorf("current API pod image digest unavailable; cannot prove old writer fence")
		}
		var deploymentImage string
		for _, container := range deployment.Spec.Template.Spec.Containers {
			if container.Name == "ml-pipeline-api-server" {
				deploymentImage = container.Image
			}
		}
		if deploymentImage != ownImage {
			return fmt.Errorf("API Deployment template differs from current pod; old replicas may restart")
		}
		for _, pod := range pods.Items {
			if pod.DeletionTimestamp != nil {
				return fmt.Errorf("API pod %s is still terminating", pod.Name)
			}
			matched := false
			for _, container := range pod.Status.ContainerStatuses {
				if container.Name == "ml-pipeline-api-server" && container.ImageID != ownImageID {
					return fmt.Errorf("old API pod %s is still running; wait for rollout to finish", pod.Name)
				}
				if container.Name == "ml-pipeline-api-server" {
					matched = true
				}
			}
			if !matched {
				return fmt.Errorf("API pod %s has no verifiable image digest", pod.Name)
			}
		}
	}
	return gate.CheckDrained(ctx)
}

// CheckDrained validates stored run and Workflow terminal reports before the
// old Deployment is scaled down. Check must still run after the writer fence.
func (gate DrainGate) CheckDrained(ctx context.Context) error {
	if gate.Database == nil || gate.Workflows == nil {
		return fmt.Errorf("MLMD drain check requires database and Workflow clients")
	}
	if err := gate.checkRuns(ctx); err != nil {
		return err
	}
	workflows, err := gate.Workflows.Resource(workflowResource).Namespace(metav1.NamespaceAll).List(ctx, metav1.ListOptions{LabelSelector: util.LabelKeyWorkflowRunId})
	if err != nil {
		return fmt.Errorf("list KFP Workflows for MLMD drain gate: %w", err)
	}
	for _, workflow := range workflows.Items {
		phase, _, err := unstructured.NestedString(workflow.Object, "status", "phase")
		if err != nil || (phase != "Succeeded" && phase != "Failed" && phase != "Error") {
			return fmt.Errorf("Workflow %s/%s is not terminal; wait before MLMD migration", workflow.GetNamespace(), workflow.GetName())
		}
		if workflow.GetLabels()[util.LabelKeyWorkflowPersistedFinalState] != "true" {
			return fmt.Errorf("Workflow %s/%s has no stored terminal report; wait for persistence", workflow.GetNamespace(), workflow.GetName())
		}
	}
	return gate.checkRuns(ctx)
}

func (gate DrainGate) checkRuns(ctx context.Context) error {
	var active int
	const query = `SELECT COUNT(*) FROM run_details WHERE NOT (
		UPPER(COALESCE(State, '')) IN ('SUCCEEDED','FAILED','SKIPPED','CANCELED','ERROR','DONE','DISABLED')
		AND FinishedAtInSec > 0)`
	if err := gate.Database.QueryRowContext(ctx, query).Scan(&active); err != nil {
		return fmt.Errorf("inspect KFP runs for MLMD drain gate: %w", err)
	}
	if active > 0 {
		return fmt.Errorf("%d KFP runs lack a stored terminal report; wait for them to finish before MLMD migration", active)
	}
	return nil
}
