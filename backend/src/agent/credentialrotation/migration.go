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

// Package credentialrotation migrates bundled SeaweedFS credentials without
// revoking a key while managed consumers or affected workloads still use it.
package credentialrotation

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

const (
	AdminSecret           = "seaweedfs-admin-credentials"
	LegacySecret          = "mlpipeline-minio-artifact"
	RevisionAnnotation    = "pipelines.kubeflow.org/storage-credential-revision"
	PhaseAnnotation       = "pipelines.kubeflow.org/storage-credential-phase"
	DrainCountAnnotation  = "pipelines.kubeflow.org/storage-credential-drain-count"
	DrainSampleAnnotation = "pipelines.kubeflow.org/storage-credential-drain-sample"
	PreparedPhase         = "ReplacementPrepared"
	DrainingPhase         = "DrainingWorkloads"
	CompletePhase         = "DefaultRevoked"
	// These publicly shipped values are used only for negative probes, never provisioning.
	DefaultAccessKey   = "minio"
	DefaultSecretKey   = "minio123"
	defaultFingerprint = "0c9d72cb381b9afa7023c36f330317ee8783fc8becf72e2c0610688e9afb544d"
)

var consumers = []string{"ml-pipeline", "ml-pipeline-ui", "kubeflow-pipelines-profile-controller"}

// Credentials are a validated access-key pair. They must never be logged.
type Credentials struct{ AccessKey, SecretKey string }

func (Credentials) String() string { return "[redacted credentials]" }

// Kubernetes exposes only namespace-scoped operations. Get methods return nil
// without an error for missing objects. Patch methods must enforce resourceVersion
// and merge only the provided fields, preserving unrelated data and annotations.
type Kubernetes interface {
	GetSecret(context.Context, string) (*corev1.Secret, error)
	PatchSecret(context.Context, string, string, map[string][]byte, map[string]string) (*corev1.Secret, error)
	CreateSecret(context.Context, *corev1.Secret) (*corev1.Secret, error)
	GetDeployment(context.Context, string) (*appsv1.Deployment, error)
	PatchDeploymentRevision(context.Context, string, string, string) error
	ListPods(context.Context, metav1.ListOptions) (*corev1.PodList, error)
}

// Storage verifies the candidate and probes both S3 and IAM before and after
// deleting the exact public key. Authentication errors alone prove rejection;
// transport errors and server failures must be returned, not treated as success.
type Storage interface {
	VerifyRegistration(context.Context) error
	Verify(context.Context) error
	DefaultAccepted(context.Context) (bool, error)
	RevokeDefault(context.Context) error
	Close()
}
type StorageFactory func(Credentials) (Storage, error)

// Waiting means reconciliation succeeded but expected external progress remains.
// It is healthy for readiness and is not migration completion.
type Waiting struct{ Message string }

func (e *Waiting) Error() string { return e.Message }
func IsWaiting(err error) bool   { var waiting *Waiting; return errors.As(err, &waiting) }
func wait(message string) error  { return &Waiting{Message: message} }

// Failure provides a safe error message without exposing an API/SDK response.
type Failure struct {
	Message string
	Cause   error
}

func (e *Failure) Error() string                { return e.Message }
func (e *Failure) Unwrap() error                { return e.Cause }
func failure(message string, cause error) error { return &Failure{Message: message, Cause: cause} }

// Migration retains clients across retries; durable state lives in AdminSecret.
// A single goroutine must call its methods.
type Migration struct {
	api     Kubernetes
	factory StorageFactory
	storage Storage
	pair    Credentials
	stage   string
	random  io.Reader
}

func New(api Kubernetes, factory StorageFactory) *Migration {
	return &Migration{api: api, factory: factory, random: rand.Reader, stage: "initialization"}
}
func (m *Migration) Stage() string { return m.stage }
func (m *Migration) Close() {
	if m.storage != nil {
		m.storage.Close()
		m.storage = nil
	}
}

// CredentialsFromSecret validates values passed through the storage startup shell.
func CredentialsFromSecret(secret *corev1.Secret) (Credentials, bool, error) {
	if secret == nil {
		return Credentials{}, false, nil
	}
	access, hasAccess := secret.Data["accesskey"]
	key, hasKey := secret.Data["secretkey"]
	if !hasAccess && !hasKey {
		return Credentials{}, false, nil
	}
	for _, value := range [][]byte{access, key} {
		if len(value) == 0 {
			return Credentials{}, false, failure("Credential Secret is incomplete; preserve and repair it", nil)
		}
		for _, c := range value {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || strings.ContainsRune("/+=_.-", rune(c))) {
				return Credentials{}, false, failure("Credential Secret contains unsupported characters; preserve and repair it", nil)
			}
		}
	}
	return Credentials{string(access), string(key)}, true, nil
}
func credentialData(pair Credentials) map[string][]byte {
	return map[string][]byte{"accesskey": []byte(pair.AccessKey), "secretkey": []byte(pair.SecretKey)}
}
func IsDefault(pair Credentials) bool {
	digest := sha256.Sum256([]byte(pair.AccessKey + "\x00" + pair.SecretKey))
	return hex.EncodeToString(digest[:]) == defaultFingerprint
}
func (m *Migration) token(size int) (string, error) {
	data := make([]byte, size)
	if _, err := io.ReadFull(m.random, data); err != nil {
		return "", failure("Cannot generate replacement credentials", err)
	}
	return hex.EncodeToString(data), nil
}
func (m *Migration) getCredentials(ctx context.Context, name string) (Credentials, bool, error) {
	secret, err := m.api.GetSecret(ctx, name)
	if err != nil {
		return Credentials{}, false, failure("Cannot read credential Secret", err)
	}
	return CredentialsFromSecret(secret)
}

// Prepare persists or adopts a candidate before storage starts. It never changes
// the public legacy Secret until storage and IAM have accepted the candidate.
func (m *Migration) Prepare(ctx context.Context) (*corev1.Secret, error) {
	state, err := m.api.GetSecret(ctx, AdminSecret)
	if err != nil {
		return nil, failure("Cannot read administrator credential state", err)
	}
	if state == nil {
		return nil, wait("Waiting for the administrator credential Secret manifest")
	}
	if state.ResourceVersion == "" || state.UID == "" {
		return nil, failure("Incomplete durable administrator metadata; retrying", nil)
	}
	current, present, err := CredentialsFromSecret(state)
	if err != nil {
		return nil, err
	}
	if present && IsDefault(current) {
		return nil, failure("Administrator Secret contains the shipped default; refusing to use it", nil)
	}
	old, oldPresent, err := m.getCredentials(ctx, LegacySecret)
	if err != nil {
		return nil, err
	}
	if present && state.Annotations[RevisionAnnotation] != "" {
		if state.Annotations[PhaseAnnotation] == CompletePhase && oldPresent && !IsDefault(old) && old != current {
			return nil, failure("Legacy Secret changed to custom credentials; reconcile the two Secrets rather than skip migration", nil)
		}
		if (!oldPresent || IsDefault(old)) && old != current && state.Annotations[PhaseAnnotation] == CompletePhase {
			revision, err := m.token(16)
			if err != nil {
				return nil, err
			}
			result, err := m.api.PatchSecret(ctx, AdminSecret, state.ResourceVersion, nil, map[string]string{RevisionAnnotation: revision, PhaseAnnotation: PreparedPhase})
			if err != nil {
				return nil, failure("Cannot record credential drift; retrying", err)
			}
			return result, nil
		}
		return state, nil
	}
	if !present {
		if oldPresent && !IsDefault(old) {
			current = old
		} else {
			access, err := m.token(16)
			if err != nil {
				return nil, err
			}
			key, err := m.token(32)
			if err != nil {
				return nil, err
			}
			current = Credentials{"kfp-" + access, key}
		}
	}
	revision, err := m.token(16)
	if err != nil {
		return nil, err
	}
	result, err := m.api.PatchSecret(ctx, AdminSecret, state.ResourceVersion, credentialData(current), map[string]string{RevisionAnnotation: revision, PhaseAnnotation: PreparedPhase})
	if err != nil {
		return nil, failure("Cannot persist replacement credentials; retrying", err)
	}
	return result, nil
}

// UsesSecret covers all Pod container and volume Secret-reference forms.
func UsesSecret(spec corev1.PodSpec, name string) bool {
	containerUses := func(env []corev1.EnvVar, envFrom []corev1.EnvFromSource) bool {
		for _, item := range envFrom {
			if item.SecretRef != nil && item.SecretRef.Name == name {
				return true
			}
		}
		for _, item := range env {
			if item.ValueFrom != nil && item.ValueFrom.SecretKeyRef != nil && item.ValueFrom.SecretKeyRef.Name == name {
				return true
			}
		}
		return false
	}
	for _, item := range append(append([]corev1.Container{}, spec.Containers...), spec.InitContainers...) {
		if containerUses(item.Env, item.EnvFrom) {
			return true
		}
	}
	for _, item := range spec.EphemeralContainers {
		if containerUses(item.Env, item.EnvFrom) {
			return true
		}
	}
	for _, item := range spec.Volumes {
		if item.Secret != nil && item.Secret.SecretName == name {
			return true
		}
		if item.Projected != nil {
			for _, source := range item.Projected.Sources {
				if source.Secret != nil && source.Secret.Name == name {
					return true
				}
			}
		}
	}
	return false
}

func (m *Migration) rolloutReady(ctx context.Context, deployment *appsv1.Deployment, expected string) error {
	desired := int32(1)
	if deployment.Spec.Replicas != nil {
		desired = *deployment.Spec.Replicas
	}
	status := deployment.Status
	if status.ObservedGeneration < deployment.Generation || status.UpdatedReplicas != desired || status.Replicas != desired || status.AvailableReplicas != desired {
		return wait("Waiting for deployment " + deployment.Name + " to finish rolling out")
	}
	if expected == "" {
		return nil
	}
	selector := deployment.Spec.Selector
	if selector == nil || len(selector.MatchExpressions) != 0 || len(selector.MatchLabels) == 0 {
		return failure("Custom deployment selectors require explicit migration review", nil)
	}
	pods, err := m.api.ListPods(ctx, metav1.ListOptions{LabelSelector: labels.SelectorFromSet(selector.MatchLabels).String()})
	if err != nil {
		return failure("Cannot verify old consumer termination", err)
	}
	if pods == nil || pods.Continue != "" {
		return failure("Cannot verify complete consumer Pod inventory", nil)
	}
	for _, pod := range pods.Items {
		if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed && pod.Annotations[RevisionAnnotation] != expected {
			return wait("Waiting for old consumer pods to terminate")
		}
	}
	return nil
}
func (m *Migration) storageReady(ctx context.Context) error {
	deployment, err := m.api.GetDeployment(ctx, "seaweedfs")
	if err != nil {
		return failure("Cannot read storage deployment", err)
	}
	if deployment == nil || !UsesSecret(deployment.Spec.Template.Spec, AdminSecret) {
		return wait("Waiting for storage to use the replacement Secret")
	}
	if err := m.rolloutReady(ctx, deployment, ""); err != nil {
		return err
	}
	pods, err := m.api.ListPods(ctx, metav1.ListOptions{LabelSelector: "app=seaweedfs"})
	if err != nil {
		return failure("Cannot verify old storage termination", err)
	}
	if pods == nil || pods.Continue != "" {
		return failure("Cannot verify complete storage Pod inventory", nil)
	}
	if len(pods.Items) == 0 {
		return wait("Waiting for the storage Pod")
	}
	for _, pod := range pods.Items {
		if !UsesSecret(pod.Spec, AdminSecret) {
			return wait("Waiting for all old storage pods to terminate before revocation")
		}
	}
	return nil
}
func (m *Migration) rollDeployment(ctx context.Context, name, value string) (*appsv1.Deployment, error) {
	deployment, err := m.api.GetDeployment(ctx, name)
	if err != nil {
		return nil, failure("Cannot read managed deployment", err)
	}
	if deployment == nil {
		return nil, wait("Waiting for managed deployment " + name)
	}
	if deployment.Spec.Template.Annotations[RevisionAnnotation] != value {
		if deployment.ResourceVersion == "" {
			return nil, failure("Missing managed deployment resource version; retrying", nil)
		}
		if err := m.api.PatchDeploymentRevision(ctx, name, deployment.ResourceVersion, value); err != nil {
			return nil, failure("Cannot roll managed deployment", err)
		}
		return nil, wait("Rolling deployment " + name + " onto the replacement credential")
	}
	if err := m.rolloutReady(ctx, deployment, value); err != nil {
		return nil, err
	}
	return deployment, nil
}
func (m *Migration) publish(ctx context.Context, pair Credentials) error {
	old, err := m.api.GetSecret(ctx, LegacySecret)
	if err != nil {
		return failure("Cannot read legacy credential Secret", err)
	}
	existing, present, err := CredentialsFromSecret(old)
	if err != nil {
		return err
	}
	if present && existing == pair {
		return nil
	}
	if present && !IsDefault(existing) {
		return failure("Legacy Secret changed to custom credentials; reconcile the two Secrets rather than overwrite it", nil)
	}
	if old == nil {
		_, err = m.api.CreateSecret(ctx, &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: LegacySecret}, Type: corev1.SecretTypeOpaque, Data: credentialData(pair)})
	} else {
		if old.ResourceVersion == "" {
			return failure("Missing legacy Secret resource version; retrying", nil)
		}
		_, err = m.api.PatchSecret(ctx, LegacySecret, old.ResourceVersion, credentialData(pair), nil)
	}
	if err != nil {
		return failure("Cannot publish replacement credentials", err)
	}
	return nil
}
func (m *Migration) rollConsumers(ctx context.Context, revision string) ([]map[string]string, error) {
	var managed []map[string]string
	for _, name := range consumers {
		deployment, err := m.api.GetDeployment(ctx, name)
		if err != nil {
			return nil, failure("Cannot inspect managed consumer", err)
		}
		if deployment == nil || !UsesSecret(deployment.Spec.Template.Spec, LegacySecret) {
			continue
		}
		deployment, err = m.rollDeployment(ctx, name, revision)
		if err != nil {
			return nil, err
		}
		managed = append(managed, deployment.Spec.Selector.MatchLabels)
	}
	return managed, nil
}
func (m *Migration) phase(ctx context.Context, state *corev1.Secret, value string, details map[string]string) error {
	changes := map[string]string{PhaseAnnotation: value}
	for key, val := range details {
		changes[key] = val
	}
	changed := false
	for key, val := range changes {
		if state.Annotations[key] != val {
			changed = true
			break
		}
	}
	if !changed {
		return nil
	}
	_, err := m.api.PatchSecret(ctx, AdminSecret, state.ResourceVersion, nil, changes)
	if err != nil {
		return failure("Cannot persist migration progress", err)
	}
	return nil
}
func matchesManaged(pod corev1.Pod, managed []map[string]string, revision string) bool {
	if pod.Labels["pipeline/runid"] != "" || pod.Annotations[RevisionAnnotation] != revision {
		return false
	}
	for _, selector := range managed {
		if len(selector) > 0 && labels.SelectorFromSet(selector).Matches(labels.Set(pod.Labels)) {
			return true
		}
	}
	return false
}
func (m *Migration) drain(ctx context.Context, state *corev1.Secret, managed []map[string]string) error {
	count := 0
	sample := make([]string, 0, 5)
	token := ""
	finished := false
	for pageNumber := 0; pageNumber < 100; pageNumber++ {
		page, err := m.api.ListPods(ctx, metav1.ListOptions{Limit: 200, Continue: token})
		if err != nil {
			return failure("Cannot verify workload drain; keeping old credential valid", err)
		}
		if page == nil {
			return failure("Malformed Pod inventory; keeping old credential valid", nil)
		}
		for _, pod := range page.Items {
			if pod.Name == "" {
				return failure("Malformed Pod inventory; keeping old credential valid", nil)
			}
			if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
				continue
			}
			if pod.Labels["pipeline/runid"] == "" && !UsesSecret(pod.Spec, LegacySecret) {
				continue
			}
			if matchesManaged(pod, managed, state.Annotations[RevisionAnnotation]) {
				continue
			}
			count++
			if len(sample) < 5 {
				name := pod.Name
				if len(name) > 253 {
					name = name[:253]
				}
				sample = append(sample, name)
			}
		}
		if page.Continue == "" {
			finished = true
			break
		}
		if page.Continue == token {
			return failure("Pod inventory did not advance; keeping old credential valid", nil)
		}
		token = page.Continue
	}
	if !finished {
		return failure("Pod inventory exceeded bounded scan; keeping old credential valid", nil)
	}
	if count == 0 {
		return nil
	}
	names, err := json.Marshal(sample)
	if err != nil {
		return failure("Cannot encode migration progress", err)
	}
	if err := m.phase(ctx, state, DrainingPhase, map[string]string{DrainCountAnnotation: strconv.Itoa(count), DrainSampleAnnotation: string(names)}); err != nil {
		return err
	}
	return wait(fmt.Sprintf("Waiting for %d affected workload Pods to finish; old credential remains valid", count))
}

// Reconcile advances the migration once. A nil result means verified completion;
// IsWaiting means healthy but incomplete. Other errors should clear readiness.
func (m *Migration) Reconcile(ctx context.Context) error {
	m.stage = "credential preparation"
	state, err := m.Prepare(ctx)
	if err != nil {
		return err
	}
	pair, present, err := CredentialsFromSecret(state)
	if err != nil {
		return err
	}
	if !present || state.UID == "" || state.Annotations[RevisionAnnotation] == "" {
		return failure("Incomplete durable administrator state; retrying", nil)
	}
	if err := m.storageReady(ctx); err != nil {
		return err
	}
	if m.storage == nil || m.pair != pair {
		m.Close()
		client, err := m.factory(pair)
		if err != nil {
			return failure("Cannot initialize storage verification", err)
		}
		if client == nil {
			return failure("Storage verification client is unavailable", nil)
		}
		m.storage, m.pair = client, pair
	}
	m.stage = "S3 registration verification"
	if err := m.storage.VerifyRegistration(ctx); err != nil {
		return failure("Cannot verify replacement S3 credential", err)
	}
	m.stage = "storage IAM cache reload"
	if _, err := m.rollDeployment(ctx, "seaweedfs", string(state.UID)); err != nil {
		return err
	}
	m.stage = "IAM authentication verification"
	if err := m.storage.Verify(ctx); err != nil {
		return failure("Cannot verify replacement IAM credential", err)
	}
	m.stage = "consumer credential rollout"
	if err := m.publish(ctx, pair); err != nil {
		return err
	}
	managed, err := m.rollConsumers(ctx, state.Annotations[RevisionAnnotation])
	if err != nil {
		return err
	}
	m.stage = "active workload drain"
	accepted, err := m.storage.DefaultAccepted(ctx)
	if err != nil {
		return failure("Cannot verify old credential acceptance", err)
	}
	if accepted {
		if err := m.drain(ctx, state, managed); err != nil {
			return err
		}
	}
	for _, name := range []string{AdminSecret, LegacySecret} {
		current, present, err := m.getCredentials(ctx, name)
		if err != nil {
			return err
		}
		if !present || current != pair {
			return failure("Credentials changed during rollout; restarting verification", nil)
		}
	}
	m.stage = "default credential revocation"
	if err := m.storage.RevokeDefault(ctx); err != nil {
		if IsWaiting(err) {
			return err
		}
		return failure("Cannot revoke the old storage credential", err)
	}
	if err := m.storage.Verify(ctx); err != nil {
		return failure("Cannot verify storage after revocation", err)
	}
	return m.phase(ctx, state, CompletePhase, map[string]string{DrainCountAnnotation: "0", DrainSampleAnnotation: "[]"})
}
