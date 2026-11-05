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
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
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
	Database            runInventoryReader
	Kubernetes          kubernetes.Interface
	Workflows           dynamic.Interface
	Namespace           string
	Deployment          string
	PodName             string
	RecurringDeployment string
	SourceSingleUser    bool
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
	blockers, err := gate.Blockers(ctx)
	if err != nil {
		return err
	}
	if len(blockers) != 0 {
		return fmt.Errorf("%s", strings.Join(blockers, "; "))
	}
	return nil
}

// Blockers lists KFP runs and owned Workflows that are active or lack a final report.
func (gate DrainGate) Blockers(ctx context.Context) ([]string, error) {
	if gate.Database == nil || gate.Workflows == nil {
		return nil, fmt.Errorf("MLMD drain check requires database and Workflow clients")
	}
	if gate.SourceSingleUser {
		if err := gate.verifySingleUserSource(ctx); err != nil {
			return nil, err
		}
	}
	runs, blockers, err := readRunInventory(ctx, gate.Database, gate.SourceSingleUser)
	if err != nil {
		return nil, err
	}
	workflows := gate.Workflows.Resource(workflowResource).Namespace(metav1.NamespaceAll)
	// Page cluster-wide results because installations can own runs in multiple namespaces.
	options := metav1.ListOptions{LabelSelector: util.LabelKeyWorkflowRunId, Limit: 500}
	for {
		page, err := workflows.List(ctx, options)
		if err != nil {
			return nil, fmt.Errorf("list KFP Workflows for MLMD drain gate: %w", err)
		}
		for _, workflow := range page.Items {
			runID := workflow.GetLabels()[util.LabelKeyWorkflowRunId]
			owner, owned := runs[runID]
			namespace := owner.namespace
			if !owned {
				continue
			}
			if namespace == "" {
				blockers = append(blockers, fmt.Sprintf("KFP run %s lacks a namespace for Workflow ownership verification", runID))
				continue
			}
			if namespace != workflow.GetNamespace() {
				continue
			}
			if owner.saved != nil {
				if owner.saved.Name != workflow.GetName() {
					blockers = append(blockers, fmt.Sprintf("Workflow %s/%s conflicts with the persisted name for KFP run %s", namespace, workflow.GetName(), runID))
					continue
				}
				if owner.saved.UID != "" && owner.saved.UID != workflow.GetUID() {
					blockers = append(blockers, fmt.Sprintf("Workflow %s/%s conflicts with the persisted UID for KFP run %s", namespace, workflow.GetName(), runID))
					continue
				}
			}
			phase, _, err := unstructured.NestedString(workflow.Object, "status", "phase")
			switch {
			case err != nil || (phase != "Succeeded" && phase != "Failed" && phase != "Error"):
				blockers = append(blockers, fmt.Sprintf("Workflow %s/%s is not terminal", namespace, workflow.GetName()))
			case workflow.GetLabels()[util.LabelKeyWorkflowPersistedFinalState] != "true":
				blockers = append(blockers, fmt.Sprintf("Workflow %s/%s has no stored terminal report", namespace, workflow.GetName()))
			}
		}
		options.Continue = page.GetContinue()
		if options.Continue == "" {
			break
		}
	}
	_, latestBlockers, err := readRunInventory(ctx, gate.Database, gate.SourceSingleUser)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]bool, len(blockers))
	for _, blocker := range blockers {
		seen[blocker] = true
	}
	for _, blocker := range latestBlockers {
		if !seen[blocker] {
			blockers = append(blockers, blocker)
		}
	}
	return blockers, nil
}

type runInventoryReader interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func loadRuns(ctx context.Context, db runInventoryReader, singleUser bool) (map[string]string, error) {
	runs, blockers, err := readRunInventory(ctx, db, singleUser)
	if err != nil {
		return nil, err
	}
	if len(blockers) != 0 {
		return nil, fmt.Errorf("%s", strings.Join(blockers, "; "))
	}
	namespaces := make(map[string]string, len(runs))
	for id, owner := range runs {
		namespaces[id] = owner.namespace
	}
	return namespaces, nil
}

type runOwnership struct {
	namespace string
	saved     *metav1.ObjectMeta
}

func readRunInventory(ctx context.Context, db runInventoryReader, singleUser bool) (map[string]runOwnership, []string, error) {
	const query = `SELECT r.UUID, COALESCE(NULLIF(NULLIF(r.Namespace, ''), '-'), NULLIF(e.Namespace, '-'), ''),
	r.State, r.FinishedAtInSec, r.Name,
	CASE WHEN ? AND COALESCE(NULLIF(NULLIF(r.Namespace, ''), '-'), NULLIF(e.Namespace, '-'), '') = ''
		THEN COALESCE(NULLIF(r.WorkflowRuntimeManifest, ''), r.PipelineRuntimeManifest) END
	FROM run_details r LEFT JOIN experiments e ON e.UUID=r.ExperimentUUID ORDER BY r.UUID`
	rows, err := db.QueryContext(ctx, query, singleUser)
	if err != nil {
		return nil, nil, fmt.Errorf("read KFP runs for MLMD drain gate: %w", err)
	}
	defer rows.Close()
	runs := make(map[string]runOwnership)
	var blockers []string
	for rows.Next() {
		var id, namespace string
		var state, name, manifest sql.NullString
		var finished sql.NullInt64
		if err := rows.Scan(&id, &namespace, &state, &finished, &name, &manifest); err != nil {
			return nil, nil, err
		}
		owner := runOwnership{namespace: namespace}
		if namespace == "" && manifest.String != "" {
			var saved struct {
				Metadata metav1.ObjectMeta `json:"metadata"`
			}
			if err := json.Unmarshal([]byte(manifest.String), &saved); err == nil && name.String != "" &&
				saved.Metadata.Name == name.String && saved.Metadata.Namespace != "" &&
				saved.Metadata.Labels[util.LabelKeyWorkflowRunId] == id {
				owner.namespace = saved.Metadata.Namespace
				owner.saved = &saved.Metadata
			}
		}
		runs[id] = owner
		switch strings.ToUpper(state.String) {
		case "SUCCEEDED", "FAILED", "SKIPPED", "CANCELED", "ERROR", "DONE", "DISABLED":
			if !finished.Valid || finished.Int64 <= 0 {
				blockers = append(blockers, fmt.Sprintf("KFP run %s has no persisted finish time and lacks a stored terminal report", id))
			}
		default:
			blockers = append(blockers, fmt.Sprintf("KFP run %s is not terminal (%q) and lacks a stored terminal report", id, state.String))
		}
	}
	return runs, blockers, rows.Err()
}

// verifySingleUserSource checks the deployment's effective MULTIUSER setting
// for the explicit historical fallback. Unreadable sources fail closed.
func (gate DrainGate) verifySingleUserSource(ctx context.Context) error {
	if gate.Kubernetes == nil || gate.Namespace == "" || gate.Deployment == "" {
		return fmt.Errorf("single-user source verification requires the API Deployment settings")
	}
	deployment, err := gate.Kubernetes.AppsV1().Deployments(gate.Namespace).Get(ctx, gate.Deployment, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("inspect API Deployment for single-user source: %w", err)
	}
	configMaps := gate.Kubernetes.CoreV1().ConfigMaps(gate.Namespace)
	for _, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name != "ml-pipeline-api-server" {
			continue
		}
		const key = "MULTIUSER"
		var value string
		explicit := false
		for _, entry := range container.Env {
			if entry.Name != key {
				continue
			}
			explicit = true
			value = entry.Value
			if entry.ValueFrom != nil {
				ref := entry.ValueFrom.ConfigMapKeyRef
				if ref == nil {
					return fmt.Errorf("cannot verify API Deployment's MULTIUSER value source")
				}
				cm, err := configMaps.Get(ctx, ref.Name, metav1.GetOptions{})
				if apierrors.IsNotFound(err) && ref.Optional != nil && *ref.Optional {
					explicit = false
					continue
				}
				if err != nil {
					return fmt.Errorf("read API MULTIUSER configuration: %w", err)
				}
				value, explicit = cm.Data[ref.Key]
				if !explicit && (ref.Optional == nil || !*ref.Optional) {
					return fmt.Errorf("API MULTIUSER configuration key is missing")
				}
			}
		}
		if !explicit {
			for _, source := range container.EnvFrom {
				if !strings.HasPrefix(key, source.Prefix) {
					continue
				}
				ref := source.ConfigMapRef
				if ref == nil {
					return fmt.Errorf("cannot verify API Deployment's MULTIUSER value from an envFrom secret")
				}
				cm, err := configMaps.Get(ctx, ref.Name, metav1.GetOptions{})
				if apierrors.IsNotFound(err) && ref.Optional != nil && *ref.Optional {
					continue
				}
				if err != nil {
					return fmt.Errorf("read API deployment configuration: %w", err)
				}
				if setting, ok := cm.Data[strings.TrimPrefix(key, source.Prefix)]; ok {
					value = setting
				}
			}
		}
		if value != "" {
			enabled, err := strconv.ParseBool(value)
			if err != nil || enabled {
				return fmt.Errorf("historical single-user fallback requires a single-user API Deployment")
			}
		}
		return nil
	}
	return fmt.Errorf("cannot verify source mode without the API server container")
}
