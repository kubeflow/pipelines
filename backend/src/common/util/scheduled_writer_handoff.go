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
	"io"
	"os"
	"reflect"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const scheduleWriterAnnotation = "pipelines.kubeflow.org/schedule-writer-protocol"
const scheduleWriterProtocol = "api-owned-v1"

var scheduleWriterDeployments = []string{"ml-pipeline", "ml-pipeline-scheduledworkflow"}

// KFP_SCHEDULE_WRITERS is trusted deployment configuration, shared by every
// API/controller process. It must include all writers sharing this installation.
const scheduleWritersEnv = "KFP_SCHEDULE_WRITERS"

type scheduleWriterIdentity struct {
	Role       string `json:"role"`
	Deployment string `json:"deployment"`
	App        string `json:"app"`
	Container  string `json:"container"`
}

type scheduleWriterConfiguration []scheduleWriterIdentity

func defaultScheduleWriters() scheduleWriterConfiguration {
	return scheduleWriterConfiguration{
		{Role: "api", Deployment: "ml-pipeline", App: "ml-pipeline", Container: "ml-pipeline-api-server"},
		{Role: "controller", Deployment: "ml-pipeline-scheduledworkflow", App: "ml-pipeline-scheduledworkflow", Container: "ml-pipeline-scheduledworkflow"},
	}
}

func configuredScheduleWriters() (scheduleWriterConfiguration, error) {
	raw := os.Getenv(scheduleWritersEnv)
	if raw == "" {
		return defaultScheduleWriters(), nil
	}
	var config scheduleWriterConfiguration
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return nil, apierrors.NewBadRequest(scheduleWritersEnv + ": supply a JSON array of role, deployment, app, and container identities: " + err.Error())
	}
	var extra interface{}
	if err := decoder.Decode(&extra); err != io.EOF {
		return nil, apierrors.NewBadRequest(scheduleWritersEnv + ": supply exactly one JSON array")
	}
	roles, deployments := map[string]bool{}, map[string]bool{}
	apps := map[string]scheduleWriterIdentity{}
	for _, writer := range config {
		if writer.Role != "api" && writer.Role != "controller" {
			return nil, apierrors.NewBadRequest(scheduleWritersEnv + ": each role must be api or controller")
		}
		if len(validation.IsDNS1123Subdomain(writer.Deployment)) != 0 || writer.App == "" || len(validation.IsValidLabelValue(writer.App)) != 0 || len(validation.IsDNS1123Label(writer.Container)) != 0 {
			return nil, apierrors.NewBadRequest(scheduleWritersEnv + ": deployment, app, and container must be nonempty valid Kubernetes names/label values")
		}
		if deployments[writer.Deployment] {
			return nil, apierrors.NewBadRequest(scheduleWritersEnv + ": deployment names must be unique")
		}
		if previous, exists := apps[writer.App]; exists && (previous.Role != writer.Role || previous.Container != writer.Container) {
			return nil, apierrors.NewBadRequest(scheduleWritersEnv + ": shared app labels must identify the same role and container")
		}
		roles[writer.Role], deployments[writer.Deployment], apps[writer.App] = true, true, writer
	}
	if !roles["api"] || !roles["controller"] {
		return nil, apierrors.NewBadRequest(scheduleWritersEnv + ": include both api and controller writers; empty or partial writer sets are unsafe")
	}
	sort.Slice(config, func(i, j int) bool { return config[i].Deployment < config[j].Deployment })
	return config, nil
}

func (config scheduleWriterConfiguration) identity(pod *corev1.Pod) (scheduleWriterIdentity, error) {
	for _, writer := range config {
		if pod.Labels["app"] == writer.App {
			return writer, nil
		}
	}
	return scheduleWriterIdentity{}, apierrors.NewBadRequest("schedule writer Pod has no configured app label; configure " + scheduleWritersEnv + " consistently on every writer")
}

// A changed app label must not hide an old writer whose container identity is
// still recognizable. Unrelated installation Pods do not need ownership reads.
func (config scheduleWriterConfiguration) possibleWriter(pod *corev1.Pod) bool {
	identities := append(defaultScheduleWriters(), config...)
	for _, writer := range identities {
		if pod.Labels["app"] == writer.App {
			return true
		}
		for _, container := range pod.Spec.Containers {
			if container.Name == writer.Container {
				return true
			}
		}
	}
	return false
}

// Resolve controller ownership before selecting a writer Deployment. Multiple
// retiring/current Deployments may share a role label; labels alone are not identity.
func scheduleWriterOwner(ctx context.Context, client kubernetes.Interface, namespace string, pod *corev1.Pod) (*metav1.OwnerReference, error) {
	owner := metav1.GetControllerOf(pod)
	if owner == nil || owner.APIVersion != "apps/v1" || owner.Kind != "ReplicaSet" || owner.UID == "" {
		return nil, fmt.Errorf("schedule writer Pod %s must be controlled by a Deployment ReplicaSet", pod.Name)
	}
	replicaSet, err := client.AppsV1().ReplicaSets(namespace).Get(ctx, owner.Name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		return nil, fmt.Errorf("schedule writer Pod %s references missing ReplicaSet %s; waiting for rollout and Pod deletion to settle", pod.Name, owner.Name)
	}
	if err != nil {
		return nil, fmt.Errorf("get writer ReplicaSet %s; apply the complete handoff RBAC: %w", owner.Name, err)
	}
	parent := metav1.GetControllerOf(replicaSet)
	if replicaSet.UID != owner.UID || parent == nil || parent.APIVersion != "apps/v1" || parent.Kind != "Deployment" || parent.UID == "" {
		return nil, fmt.Errorf("schedule writer Pod %s has no verified Deployment owner", pod.Name)
	}
	return parent, nil
}

func (config scheduleWriterConfiguration) ownedIdentity(ctx context.Context, client kubernetes.Interface, namespace string, pod *corev1.Pod, deployments map[string]*appsv1.Deployment) (scheduleWriterIdentity, error) {
	owner, err := scheduleWriterOwner(ctx, client, namespace, pod)
	if err != nil {
		return scheduleWriterIdentity{}, err
	}
	for _, writer := range config {
		if writer.Deployment != owner.Name || writer.App != pod.Labels["app"] {
			continue
		}
		deployment := deployments[writer.Deployment]
		selector, err := metav1.LabelSelectorAsSelector(deployment.Spec.Selector)
		if deployment.UID != owner.UID || err != nil || deployment.Spec.Selector == nil || selector.Empty() || !selector.Matches(labels.Set(pod.Labels)) {
			return scheduleWriterIdentity{}, fmt.Errorf("schedule writer Pod %s does not match Deployment %s UID and selector", pod.Name, deployment.Name)
		}
		return writer, nil
	}
	return scheduleWriterIdentity{}, fmt.Errorf("schedule writer Pod %s belongs to unaccounted Deployment %s; include every writer in the managed handoff", pod.Name, owner.Name)
}

// scheduleWriterClaim binds registration to the currently running container.
// An in-place Pod image change or a restarted container cannot inherit its
// predecessor's registration, even though the Pod UID stays the same.
func scheduleWriterClaim(pod *corev1.Pod) (string, error) {
	config, err := configuredScheduleWriters()
	if err != nil {
		return "", err
	}
	writer, err := config.identity(pod)
	if err != nil {
		return "", err
	}
	containerName := writer.Container
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
		claimParts := []interface{}{scheduleWriterProtocol, string(pod.UID), image, status.ImageID, status.ContainerID, status.RestartCount}
		if !reflect.DeepEqual(config, defaultScheduleWriters()) {
			claimParts = append(claimParts, config)
		}
		identity, err := json.Marshal(claimParts)
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
	config, err := configuredScheduleWriters()
	if err != nil {
		return err
	}
	writer, err := config.identity(pod)
	if err != nil {
		return err
	}
	containerName := writer.Container
	found := false
	for _, container := range pod.Spec.Containers {
		found = found || container.Name == containerName
	}
	if !found {
		return apierrors.NewBadRequest("schedule writer Pod has no configured container; correct KFP_SCHEDULE_WRITERS on every writer")
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
			current, readErr := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
			if readErr == nil && (current.UID != pod.UID || current.ResourceVersion != pod.ResourceVersion) {
				err = apierrors.NewConflict(schema.GroupResource{Resource: "pods"}, podName, err)
			}
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
	config, err := configuredScheduleWriters()
	if err != nil {
		return err
	}
	automatic := os.Getenv(scheduleWritersEnv) == ""
	var discovered map[string]*appsv1.Deployment
	if automatic {
		config, discovered, err = discoverScheduleWriters(ctx, client, namespace)
		if err != nil {
			return err
		}
	}
	deployments := make(map[string]*appsv1.Deployment, len(config))
	for _, writer := range config {
		name := writer.Deployment
		deployment, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			// Peer Deployments may be created after this Pod during installation.
			// Keep this rollout wait distinct from a missing self Pod identity.
			return fmt.Errorf("managed writer Deployment %s does not exist yet; waiting for installation or rollout", name)
		}
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
	pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("list managed writer Pods: %w", err)
	}
	counts := make(map[string]int32)
	for _, pod := range pods.Items {
		// Completed Pods cannot write and may remain until cluster garbage collection.
		if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			continue
		}
		if !config.possibleWriter(&pod) {
			continue
		}
		writer, err := config.ownedIdentity(ctx, client, namespace, &pod, deployments)
		if err != nil {
			return err
		}
		ready := false
		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				ready = true
			}
		}
		if !ready {
			return fmt.Errorf("managed writer Pod %s is not Ready; wait for its readiness probe", pod.Name)
		}
		claim, err := scheduleWriterClaim(&pod)
		if err != nil || pod.Annotations[scheduleWriterAnnotation] != claim {
			return fmt.Errorf("managed writer Pod %s is incompatible or terminating; wait for its replacement and verify all configured writer Deployments use compatible images and identical KFP_SCHEDULE_WRITERS", pod.Name)
		}
		counts[writer.Deployment]++
	}
	for _, writer := range config {
		name := writer.Deployment
		previous := deployments[name]
		current, err := client.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("managed writer Deployment %s disappeared during handoff; wait for installation or rollout to recreate it", name)
		}
		if err != nil {
			return fmt.Errorf("recheck managed writer Deployment %s: %w", name, err)
		}
		if previous.UID != current.UID || previous.ResourceVersion != current.ResourceVersion || counts[name] != previous.Status.Replicas {
			return fmt.Errorf("managed writer Deployment %s changed during handoff", name)
		}
	}
	if automatic {
		currentConfig, current, err := discoverScheduleWriters(ctx, client, namespace)
		if err != nil {
			return err
		}
		if !reflect.DeepEqual(config, currentConfig) {
			return fmt.Errorf("managed writer Deployment inventory changed during handoff; retry discovery")
		}
		for name, previous := range discovered {
			if current[name].UID != previous.UID || current[name].ResourceVersion != previous.ResourceVersion || deployments[name].UID != previous.UID || deployments[name].ResourceVersion != previous.ResourceVersion {
				return fmt.Errorf("managed writer Deployment %s changed during discovery; retry handoff", name)
			}
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
