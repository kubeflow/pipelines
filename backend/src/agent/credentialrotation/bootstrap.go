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

package credentialrotation

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

const (
	MigrationJobLabel  = "seaweedfs-credential-rotation"
	StateUIDAnnotation = "pipelines.kubeflow.org/storage-credential-secret-uid"
)

// MigrationJobName identifies one durable credential migration, independently
// of pod restarts and the image used to finish that migration.
func MigrationJobName(state *corev1.Secret) string {
	sum := sha256.Sum256([]byte(string(state.UID) + ":" + state.Annotations[RevisionAnnotation]))
	return "seaweedfs-credential-migration-" + hex.EncodeToString(sum[:8])
}

// EnsureMigrationJob starts an incomplete migration and leaves completed state
// alone. Foreground replacement handles immutable Job image upgrades without
// overlapping the previous migration pod. The caller retries Waiting results.
func EnsureMigrationJob(ctx context.Context, client kubernetes.Interface, namespace, image string, uid int64, state *corev1.Secret) error {
	if state == nil || state.Name != AdminSecret || state.UID == "" || state.Annotations[RevisionAnnotation] == "" {
		return errors.New("prepare the persistent credential Secret before starting migration")
	}
	if state.Annotations[PhaseAnnotation] == CompletePhase {
		return nil
	}
	if namespace == "" || image == "" || uid <= 0 {
		return errors.New("migration requires a namespace, image and non-root bootstrap identity")
	}
	jobs := client.BatchV1().Jobs(namespace)
	name := MigrationJobName(state)
	existing, err := jobs.Get(ctx, name, metav1.GetOptions{})
	if apierrors.IsNotFound(err) {
		_, err = jobs.Create(ctx, migrationJob(namespace, image, uid, state), metav1.CreateOptions{})
		if apierrors.IsAlreadyExists(err) {
			return wait("Migration Job was created concurrently; verifying its identity")
		}
		if err != nil {
			return failure("Cannot create migration Job; inspect bootstrap RBAC", err)
		}
		return nil
	}
	if err != nil {
		return failure("Cannot inspect migration Job; inspect bootstrap RBAC", err)
	}
	if existing.Labels["app"] != MigrationJobLabel || existing.Annotations[StateUIDAnnotation] != string(state.UID) || existing.Annotations[RevisionAnnotation] != state.Annotations[RevisionAnnotation] {
		return errors.New("migration Job name is occupied by an unrelated object; inspect its identity before retrying")
	}
	if existing.DeletionTimestamp != nil {
		return wait("Waiting for the previous migration Job and its pods to terminate")
	}
	replace := len(existing.Spec.Template.Spec.Containers) != 1 || existing.Spec.Template.Spec.Containers[0].Image != image
	for _, condition := range existing.Status.Conditions {
		if condition.Status == corev1.ConditionTrue && (condition.Type == batchv1.JobFailed || condition.Type == batchv1.JobComplete) {
			// A completed Job with incomplete durable state cannot make progress.
			replace = true
		}
	}
	if !replace {
		return nil
	}
	foreground := metav1.DeletePropagationForeground
	jobUID := existing.UID
	if jobUID == "" {
		return errors.New("migration Job has no UID; refusing unsafe replacement")
	}
	err = jobs.Delete(ctx, name, metav1.DeleteOptions{PropagationPolicy: &foreground, Preconditions: &metav1.Preconditions{UID: &jobUID}})
	if err != nil && !apierrors.IsNotFound(err) {
		return failure("Cannot replace migration Job; inspect bootstrap RBAC", err)
	}
	return wait("Replacing the previous migration Job; waiting for its pods to terminate")
}

func migrationJob(namespace, image string, uid int64, state *corev1.Secret) *batchv1.Job {
	backoff := int32(6)
	trueValue, falseValue := true, false
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name: MigrationJobName(state), Namespace: namespace,
			Labels:      map[string]string{"app": MigrationJobLabel},
			Annotations: map[string]string{StateUIDAnnotation: string(state.UID), RevisionAnnotation: state.Annotations[RevisionAnnotation]},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit: &backoff,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": MigrationJobLabel}, Annotations: map[string]string{"sidecar.istio.io/inject": "false"}},
				Spec: corev1.PodSpec{
					ServiceAccountName: MigrationJobLabel,
					RestartPolicy:      corev1.RestartPolicyOnFailure,
					SecurityContext:    &corev1.PodSecurityContext{RunAsNonRoot: &trueValue, RunAsUser: &uid, SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault}},
					Containers: []corev1.Container{{
						Name: "migrate", Image: image, ImagePullPolicy: corev1.PullIfNotPresent,
						Command:         []string{"/bin/credential-rotation", "--mode=migrate"},
						Env:             []corev1.EnvVar{{Name: "POD_NAMESPACE", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{FieldPath: "metadata.namespace"}}}},
						Ports:           []corev1.ContainerPort{{Name: "health", ContainerPort: 8080}},
						ReadinessProbe:  &corev1.Probe{ProbeHandler: corev1.ProbeHandler{HTTPGet: &corev1.HTTPGetAction{Path: "/readyz", Port: intstr.FromInt32(8080)}}, PeriodSeconds: 10},
						SecurityContext: &corev1.SecurityContext{AllowPrivilegeEscalation: &falseValue, ReadOnlyRootFilesystem: &trueValue, Capabilities: &corev1.Capabilities{Drop: []corev1.Capability{"ALL"}}},
						Resources:       corev1.ResourceRequirements{Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse("10m"), corev1.ResourceMemory: resource.MustParse("64Mi")}, Limits: corev1.ResourceList{corev1.ResourceMemory: resource.MustParse("256Mi")}},
					}},
				},
			},
		},
	}
}
