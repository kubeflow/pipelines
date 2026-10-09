// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync/atomic"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const scheduleWriterAnnotation = "pipelines.kubeflow.org/schedule-writer-protocol"
const scheduleWriterProtocol = "api-owned-v1"

var scheduleWriterDeployments = []string{"ml-pipeline", "ml-pipeline-scheduledworkflow"}

// scheduleWriterClaim binds registration to the currently running container.
// An in-place Pod image change or a restarted container cannot inherit its
// predecessor's registration, even though the Pod UID stays the same.
func scheduleWriterClaim(pod *corev1.Pod) (string, error) {
	containerName := pod.Labels["app"]
	switch containerName {
	case "ml-pipeline":
		containerName = "ml-pipeline-api-server"
	case "ml-pipeline-scheduledworkflow":
	default:
		return "", fmt.Errorf("pod is not a managed schedule writer")
	}
	if pod.UID == "" || pod.DeletionTimestamp != nil || pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
		return "", fmt.Errorf("schedule writer Pod is not live")
	}
	image := ""
	for _, container := range pod.Spec.Containers {
		if container.Name == containerName {
			image = container.Image
		}
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.Name != containerName {
			continue
		}
		// ContainerStatus.Image may be normalized or resolved by the runtime.
		// Bind the claim to its immutable runtime IDs, not image-name equality.
		if image == "" || status.ImageID == "" || status.ContainerID == "" || status.State.Running == nil {
			return "", fmt.Errorf("schedule writer container is not running with an identified image and container")
		}
		identity, err := json.Marshal([]interface{}{scheduleWriterProtocol, string(pod.UID), image, status.ImageID, status.ContainerID, status.RestartCount})
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s:%x", scheduleWriterProtocol, sha256.Sum256(identity)), nil
	}
	return "", fmt.Errorf("schedule writer container identity is unavailable")
}

// RegisterManagedScheduleWriter records a claim by the running, compatible
// process, never by a Deployment template. UID and resource-version tests prevent
// registration of a replacement Pod or overwriting a concurrent metadata edit.
func RegisterManagedScheduleWriter(ctx context.Context, client kubernetes.Interface, namespace, podName string) error {
	if namespace == "" || podName == "" {
		return fmt.Errorf("managed schedule writer requires its Pod namespace and name")
	}
	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("get schedule writer Pod: %w", err)
	}
	containerName := pod.Labels["app"]
	switch containerName {
	case "ml-pipeline":
		containerName = "ml-pipeline-api-server"
	case "ml-pipeline-scheduledworkflow":
	default:
		return apierrors.NewBadRequest("schedule writer Pod has no managed app label; apply the standard API/controller Deployment")
	}
	found := false
	for _, container := range pod.Spec.Containers {
		found = found || container.Name == containerName
	}
	if !found {
		return apierrors.NewBadRequest("schedule writer Pod has no managed container; apply the standard API/controller Deployment")
	}
	if pod.UID == "" || pod.ResourceVersion == "" || pod.DeletionTimestamp != nil {
		return fmt.Errorf("schedule writer Pod is not a live identified Pod")
	}
	claim, err := scheduleWriterClaim(pod)
	if err != nil {
		return err
	}
	if pod.Annotations[scheduleWriterAnnotation] == claim {
		return nil
	}
	annotations := make(map[string]string, len(pod.Annotations)+1)
	for key, value := range pod.Annotations {
		annotations[key] = value
	}
	annotations[scheduleWriterAnnotation] = claim
	patch, err := json.Marshal([]map[string]interface{}{
		{"op": "test", "path": "/metadata/uid", "value": string(pod.UID)},
		{"op": "test", "path": "/metadata/resourceVersion", "value": pod.ResourceVersion},
		{"op": "add", "path": "/metadata/annotations", "value": annotations},
	})
	if err != nil {
		return err
	}
	_, err = client.CoreV1().Pods(namespace).Patch(ctx, podName, types.JSONPatchType, patch, metav1.PatchOptions{})
	if err != nil {
		// JSON Patch test failures use HTTP 422, not 409. A concurrent Pod
		// status/metadata update invalidates our resourceVersion precondition.
		if apierrors.IsInvalid(err) {
			err = apierrors.NewConflict(schema.GroupResource{Resource: "pods"}, podName, err)
		}
		return fmt.Errorf("register schedule writer Pod: %w", err)
	}
	return nil
}

// ManagedScheduleWritersReady fences the standard installation's old writers.
// APIs may serve ordinary traffic while this returns an error. This does not
// discover other installations or processes sharing the same database.
func ManagedScheduleWritersReady(ctx context.Context, client kubernetes.Interface, namespace string) error {
	if namespace == "" {
		return fmt.Errorf("managed schedule writer namespace is required")
	}
	deployments := make(map[string]*appsv1.Deployment, len(scheduleWriterDeployments))
	for _, name := range scheduleWriterDeployments {
		deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("get managed writer Deployment %s: %w", name, err)
		}
		desired := int32(1)
		if deployment.Spec.Replicas != nil {
			desired = *deployment.Spec.Replicas
		}
		if deployment.DeletionTimestamp != nil || deployment.Status.ObservedGeneration < deployment.Generation ||
			deployment.Status.UpdatedReplicas != desired || deployment.Status.Replicas != desired {
			return fmt.Errorf("managed writer Deployment %s has not finished rolling out", name)
		}
		deployments[name] = deployment
	}
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app in (ml-pipeline,ml-pipeline-scheduledworkflow)"})
	if err != nil {
		return fmt.Errorf("list managed writer Pods: %w", err)
	}
	counts := make(map[string]int32)
	for _, pod := range pods.Items {
		// Completed Pods cannot write and may remain until cluster garbage collection.
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			continue
		}
		claim, err := scheduleWriterClaim(&pod)
		if err != nil || pod.Annotations[scheduleWriterAnnotation] != claim {
			return fmt.Errorf("managed writer Pod %s is incompatible or terminating", pod.Name)
		}
		counts[pod.Labels["app"]]++
	}
	for _, name := range scheduleWriterDeployments {
		previous := deployments[name]
		current, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return fmt.Errorf("recheck managed writer Deployment %s: %w", name, err)
		}
		if previous.UID != current.UID || previous.ResourceVersion != current.ResourceVersion || counts[name] != previous.Status.Replicas {
			return fmt.Errorf("managed writer Deployment %s changed during handoff", name)
		}
	}
	return nil
}

// NewManagedScheduleWriterHandoff returns a process-local, one-way upgrade
// fence. A successful handoff is retained across later compatible rollouts;
// downgrading to a legacy writer requires a separately coordinated rollback.
// Failed checks are retried, and concurrent callers share the first success.
func NewManagedScheduleWriterHandoff(client kubernetes.Interface, namespace, podName string) func(context.Context) error {
	var complete atomic.Bool
	gate := make(chan struct{}, 1)
	gate <- struct{}{}
	return func(ctx context.Context) error {
		if complete.Load() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-gate:
		}
		defer func() { gate <- struct{}{} }()
		if complete.Load() {
			return nil
		}
		if err := RegisterManagedScheduleWriter(ctx, client, namespace, podName); err != nil {
			return err
		}
		if err := ManagedScheduleWritersReady(ctx, client, namespace); err != nil {
			return err
		}
		complete.Store(true)
		return nil
	}
}

// WaitForManagedScheduleWriters registers this process before waiting for peers.
// Invalid identity and access configuration fail startup; rollout progress and
// transient API failures retry until cancellation.
func WaitForManagedScheduleWriters(ctx context.Context, client kubernetes.Interface, namespace, podName string, onWait func(error)) error {
	if namespace == "" || podName == "" {
		return fmt.Errorf("managed schedule writer identity is missing; set POD_NAMESPACE and POD_NAME from the Downward API")
	}
	ready := NewManagedScheduleWriterHandoff(client, namespace, podName)
	return wait.PollUntilContextCancel(ctx, 5*time.Second, true, func(ctx context.Context) (bool, error) {
		err := ready(ctx)
		if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) || apierrors.IsNotFound(err) || apierrors.IsBadRequest(err) || apierrors.IsInvalid(err) {
			return false, fmt.Errorf("managed schedule writer configuration is invalid; verify POD_NAME/POD_NAMESPACE and apply the complete managed-writer RBAC and Deployments: %w", err)
		}
		if err != nil && onWait != nil {
			onWait(err)
		}
		return err == nil, nil
	})
}
