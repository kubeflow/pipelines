// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package credentialrotation

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

var testContext = context.Background()
var testPair = Credentials{"synthetic-admin", "synthetic-secret"}

func secretFixture(name string, pair *Credentials) *corev1.Secret {
	s := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID(name + "-uid"), ResourceVersion: "1"}, Type: corev1.SecretTypeOpaque}
	if pair != nil {
		s.Data = credentialData(*pair)
	}
	return s
}
func podSpec(secret string) corev1.PodSpec {
	return corev1.PodSpec{Containers: []corev1.Container{{Name: "fixture", EnvFrom: []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: secret}}}}}}}
}
func deploymentFixture(name, secret string) *appsv1.Deployment {
	one := int32(1)
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: name, ResourceVersion: "1", Generation: 1}, Spec: appsv1.DeploymentSpec{Replicas: &one, Selector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": name}}, Template: corev1.PodTemplateSpec{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"app": name}}, Spec: podSpec(secret)}}, Status: appsv1.DeploymentStatus{ObservedGeneration: 1, Replicas: 1, UpdatedReplicas: 1, AvailableReplicas: 1}}
}
func workload(name string, phase corev1.PodPhase) corev1.Pod {
	return corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name, Labels: map[string]string{"pipeline/runid": "test-run"}}, Status: corev1.PodStatus{Phase: phase}}
}

type fakeAPI struct {
	secrets           map[string]*corev1.Secret
	deployments       map[string]*appsv1.Deployment
	pods              []corev1.Pod
	pages             []*corev1.PodList
	listError         error
	afterDrain        func()
	patches           int
	creates           int
	deploymentPatches int
}

func newFakeAPI() *fakeAPI {
	a := &fakeAPI{secrets: map[string]*corev1.Secret{AdminSecret: secretFixture(AdminSecret, nil)}, deployments: map[string]*appsv1.Deployment{"seaweedfs": deploymentFixture("seaweedfs", AdminSecret)}}
	a.deployments["seaweedfs"].Spec.Template.Annotations = map[string]string{RevisionAnnotation: string(a.secrets[AdminSecret].UID)}
	return a
}
func (a *fakeAPI) GetSecret(_ context.Context, name string) (*corev1.Secret, error) {
	if s := a.secrets[name]; s != nil {
		return s.DeepCopy(), nil
	}
	return nil, nil
}
func (a *fakeAPI) PatchSecret(_ context.Context, name, rv string, data map[string][]byte, annotations map[string]string) (*corev1.Secret, error) {
	s := a.secrets[name]
	if s == nil || s.ResourceVersion != rv {
		return nil, errors.New("resource version conflict")
	}
	if s.Data == nil {
		s.Data = map[string][]byte{}
	}
	for k, v := range data {
		s.Data[k] = append([]byte{}, v...)
	}
	if s.Annotations == nil {
		s.Annotations = map[string]string{}
	}
	for k, v := range annotations {
		s.Annotations[k] = v
	}
	number, _ := strconv.Atoi(rv)
	s.ResourceVersion = strconv.Itoa(number + 1)
	a.patches++
	return s.DeepCopy(), nil
}
func (a *fakeAPI) CreateSecret(_ context.Context, s *corev1.Secret) (*corev1.Secret, error) {
	if a.secrets[s.Name] != nil {
		return nil, errors.New("already exists")
	}
	a.creates++
	copy := s.DeepCopy()
	copy.ResourceVersion = "1"
	copy.UID = types.UID(s.Name + "-uid")
	a.secrets[s.Name] = copy
	return copy.DeepCopy(), nil
}
func (a *fakeAPI) GetDeployment(_ context.Context, name string) (*appsv1.Deployment, error) {
	if d := a.deployments[name]; d != nil {
		return d.DeepCopy(), nil
	}
	return nil, nil
}
func (a *fakeAPI) PatchDeploymentRevision(_ context.Context, name, rv, revision string) error {
	d := a.deployments[name]
	if d == nil || d.ResourceVersion != rv {
		return errors.New("resource version conflict")
	}
	a.deploymentPatches++
	if d.Spec.Template.Annotations == nil {
		d.Spec.Template.Annotations = map[string]string{}
	}
	d.Spec.Template.Annotations[RevisionAnnotation] = revision
	d.Generation++
	d.Status.ObservedGeneration = d.Generation
	number, _ := strconv.Atoi(rv)
	d.ResourceVersion = strconv.Itoa(number + 1)
	return nil
}
func (a *fakeAPI) ListPods(_ context.Context, options metav1.ListOptions) (*corev1.PodList, error) {
	if options.Limit > 0 {
		if a.listError != nil {
			return nil, a.listError
		}
		if a.afterDrain != nil {
			a.afterDrain()
		}
		if a.pages != nil {
			index := 0
			if options.Continue != "" {
				index, _ = strconv.Atoi(options.Continue)
			}
			if index >= len(a.pages) {
				return nil, errors.New("page outside fixture")
			}
			if a.pages[index] == nil {
				return nil, nil
			}
			return a.pages[index].DeepCopy(), nil
		}
	}
	all := append([]corev1.Pod{}, a.pods...)
	for name, d := range a.deployments {
		all = append(all, corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: name + "-pod", Labels: d.Spec.Template.Labels, Annotations: d.Spec.Template.Annotations}, Spec: d.Spec.Template.Spec, Status: corev1.PodStatus{Phase: corev1.PodRunning}})
	}
	selector, err := labels.Parse(options.LabelSelector)
	if err != nil {
		return nil, err
	}
	result := &corev1.PodList{}
	for _, p := range all {
		if selector.Matches(labels.Set(p.Labels)) {
			result.Items = append(result.Items, *p.DeepCopy())
		}
	}
	return result, nil
}

type fakeStorage struct {
	accepted                                                  bool
	revokeCalls, verifyCalls, registrationCalls, closeCalls   int
	verifyError, registrationError, defaultError, revokeError error
}

func (s *fakeStorage) VerifyRegistration(context.Context) error {
	s.registrationCalls++
	return s.registrationError
}
func (s *fakeStorage) Verify(context.Context) error { s.verifyCalls++; return s.verifyError }
func (s *fakeStorage) DefaultAccepted(context.Context) (bool, error) {
	return s.accepted, s.defaultError
}
func (s *fakeStorage) RevokeDefault(context.Context) error {
	s.revokeCalls++
	if s.revokeError != nil {
		return s.revokeError
	}
	s.accepted = false
	return nil
}
func (s *fakeStorage) Close() { s.closeCalls++ }
func fixture() (*Migration, *fakeAPI, *fakeStorage) {
	a := newFakeAPI()
	s := &fakeStorage{accepted: true}
	m := New(a, func(Credentials) (Storage, error) { return s, nil })
	return m, a, s
}
func requireWaiting(t *testing.T, err error) {
	t.Helper()
	if !IsWaiting(err) {
		t.Fatalf("expected healthy waiting, got %v", err)
	}
}
func requireFailure(t *testing.T, err error) {
	t.Helper()
	if err == nil || IsWaiting(err) {
		t.Fatalf("expected unhealthy failure, got %v", err)
	}
}
func requireSuccess(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func TestPrepareGeneratesDurableCredentialsOnce(t *testing.T) {
	m, a, _ := fixture()
	first, err := m.Prepare(testContext)
	requireSuccess(t, err)
	pair, present, err := CredentialsFromSecret(first)
	requireSuccess(t, err)
	if !present || IsDefault(pair) {
		t.Fatal("candidate not generated safely")
	}
	second, err := m.Prepare(testContext)
	requireSuccess(t, err)
	other, _, _ := CredentialsFromSecret(second)
	if pair != other || a.patches != 1 {
		t.Fatal("retry replaced durable candidate")
	}
	if a.secrets[LegacySecret] != nil {
		t.Fatal("published before storage verification")
	}
}
func TestPreparePreservesCustomCredentials(t *testing.T) {
	for _, pair := range []Credentials{testPair, {DefaultAccessKey, "operator-secret"}} {
		t.Run(pair.AccessKey, func(t *testing.T) {
			m, a, _ := fixture()
			a.secrets[LegacySecret] = secretFixture(LegacySecret, &pair)
			state, err := m.Prepare(testContext)
			requireSuccess(t, err)
			got, _, _ := CredentialsFromSecret(state)
			if got != pair {
				t.Fatal("custom credentials changed")
			}
		})
	}
}
func TestPrepareRotatesOnlyExactPublicPair(t *testing.T) {
	m, a, _ := fixture()
	old := Credentials{DefaultAccessKey, DefaultSecretKey}
	a.secrets[LegacySecret] = secretFixture(LegacySecret, &old)
	state, err := m.Prepare(testContext)
	requireSuccess(t, err)
	got, _, _ := CredentialsFromSecret(state)
	if IsDefault(got) {
		t.Fatal("public default was adopted")
	}
	if !IsDefault(old) {
		t.Fatal("negative-probe fingerprint changed")
	}
}
func TestPrepareRejectsDefaultCandidateAndMalformedCredentials(t *testing.T) {
	cases := []map[string][]byte{credentialData(Credentials{DefaultAccessKey, DefaultSecretKey}), {"accesskey": []byte("only-one")}, {"accesskey": []byte("bad\nkey"), "secretkey": []byte("value")}, {"accesskey": []byte("valid"), "secretkey": {0xff}}}
	for i, data := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			m, a, _ := fixture()
			a.secrets[AdminSecret].Data = data
			_, err := m.Prepare(testContext)
			requireFailure(t, err)
			if a.patches != 0 {
				t.Fatal("modified malformed state")
			}
		})
	}
}
func TestPrepareWaitsForManifestAndFailsOnRandomFailure(t *testing.T) {
	m, a, _ := fixture()
	delete(a.secrets, AdminSecret)
	_, err := m.Prepare(testContext)
	requireWaiting(t, err)
	a.secrets[AdminSecret] = secretFixture(AdminSecret, nil)
	m.random = bytes.NewReader(nil)
	_, err = m.Prepare(testContext)
	requireFailure(t, err)
	if a.patches != 0 {
		t.Fatal("persisted incomplete random output")
	}
}
func TestStorageRestartPrecedesPublishing(t *testing.T) {
	m, a, s := fixture()
	a.deployments["seaweedfs"].Spec.Template.Annotations = nil
	err := m.Reconcile(testContext)
	requireWaiting(t, err)
	if s.registrationCalls != 1 || s.verifyCalls != 0 || a.secrets[LegacySecret] != nil || s.revokeCalls != 0 {
		t.Fatal("consumer publication bypassed storage reload")
	}
	requireSuccess(t, m.Reconcile(testContext))
	if a.deploymentPatches != 1 {
		t.Fatal("repeated storage restart")
	}
}
func TestStorageRegistrationFailureKeepsLegacyUnchanged(t *testing.T) {
	m, a, s := fixture()
	s.registrationError = errors.New("fixture response with secret")
	err := m.Reconcile(testContext)
	requireFailure(t, err)
	if strings.Contains(err.Error(), "fixture response") {
		t.Fatal("error leaked underlying response")
	}
	if a.secrets[LegacySecret] != nil || s.revokeCalls != 0 {
		t.Fatal("published unverified credential")
	}
}
func TestOldStoragePodBlocksRevocation(t *testing.T) {
	m, a, s := fixture()
	a.pods = []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "old-storage", Labels: map[string]string{"app": "seaweedfs"}}, Spec: podSpec(LegacySecret)}}
	requireWaiting(t, m.Reconcile(testContext))
	if s.revokeCalls != 0 {
		t.Fatal("revoked while old startup pod remains")
	}
}
func TestManagedConsumerRolloutBlocksRevocation(t *testing.T) {
	m, a, s := fixture()
	a.deployments["ml-pipeline"] = deploymentFixture("ml-pipeline", LegacySecret)
	requireWaiting(t, m.Reconcile(testContext))
	if s.revokeCalls != 0 {
		t.Fatal("revoked before managed rollout")
	}
	a.deployments["ml-pipeline"].Status.AvailableReplicas = 0
	requireWaiting(t, m.Reconcile(testContext))
	a.deployments["ml-pipeline"].Status.AvailableReplicas = 1
	a.pods = []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "old-api", Labels: map[string]string{"app": "ml-pipeline"}, DeletionTimestamp: &metav1.Time{}}, Spec: podSpec(LegacySecret), Status: corev1.PodStatus{Phase: corev1.PodRunning}}}
	requireWaiting(t, m.Reconcile(testContext))
	a.pods = nil
	requireSuccess(t, m.Reconcile(testContext))
	if s.revokeCalls != 1 {
		t.Fatal("expected revocation after rollout")
	}
}
func TestDrainResumesAndNewWorkExtendsWait(t *testing.T) {
	m, a, s := fixture()
	a.pods = []corev1.Pod{workload("old-task", corev1.PodRunning)}
	requireWaiting(t, m.Reconcile(testContext))
	state := a.secrets[AdminSecret]
	if state.Annotations[PhaseAnnotation] != DrainingPhase || state.Annotations[DrainCountAnnotation] != "1" {
		t.Fatal("drain progress missing")
	}
	m = New(a, func(Credentials) (Storage, error) { return s, nil })
	a.pods = []corev1.Pod{workload("old-task", corev1.PodSucceeded), workload("new-task", corev1.PodPending)}
	requireWaiting(t, m.Reconcile(testContext))
	if s.revokeCalls != 0 {
		t.Fatal("revoked during resumed drain")
	}
	a.pods[1].Status.Phase = corev1.PodFailed
	requireSuccess(t, m.Reconcile(testContext))
	if state.Annotations[PhaseAnnotation] != CompletePhase || state.Annotations[DrainCountAnnotation] != "0" || state.Annotations[DrainSampleAnnotation] != "[]" {
		t.Fatal("completion state missing")
	}
}
func TestDrainDetectsAllSecretReferenceForms(t *testing.T) {
	from := []corev1.EnvFromSource{{SecretRef: &corev1.SecretEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: LegacySecret}}}}
	env := []corev1.EnvVar{{Name: "fixture", ValueFrom: &corev1.EnvVarSource{SecretKeyRef: &corev1.SecretKeySelector{LocalObjectReference: corev1.LocalObjectReference{Name: LegacySecret}, Key: "accesskey"}}}}
	specs := []corev1.PodSpec{{Containers: []corev1.Container{{EnvFrom: from}}}, {Containers: []corev1.Container{{Env: env}}}, {InitContainers: []corev1.Container{{EnvFrom: from}}}, {InitContainers: []corev1.Container{{Env: env}}}, {EphemeralContainers: []corev1.EphemeralContainer{{EphemeralContainerCommon: corev1.EphemeralContainerCommon{EnvFrom: from}}}}, {EphemeralContainers: []corev1.EphemeralContainer{{EphemeralContainerCommon: corev1.EphemeralContainerCommon{Env: env}}}}, {Volumes: []corev1.Volume{{VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: LegacySecret}}}}}, {Volumes: []corev1.Volume{{VolumeSource: corev1.VolumeSource{Projected: &corev1.ProjectedVolumeSource{Sources: []corev1.VolumeProjection{{Secret: &corev1.SecretProjection{LocalObjectReference: corev1.LocalObjectReference{Name: LegacySecret}}}}}}}}}}
	for i, spec := range specs {
		for _, phase := range []corev1.PodPhase{corev1.PodPending, corev1.PodRunning, corev1.PodUnknown, ""} {
			t.Run(fmt.Sprintf("%d-%s", i, phase), func(t *testing.T) {
				m, a, s := fixture()
				a.pods = []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "custom-consumer", DeletionTimestamp: &metav1.Time{}}, Spec: spec, Status: corev1.PodStatus{Phase: phase}}}
				requireWaiting(t, m.Reconcile(testContext))
				if s.revokeCalls != 0 {
					t.Fatal("missed active Secret consumer")
				}
			})
		}
	}
}
func TestDrainExemptsOnlyVerifiedManagedConsumers(t *testing.T) {
	m, a, s := fixture()
	state, err := m.Prepare(testContext)
	requireSuccess(t, err)
	managed := deploymentFixture("ml-pipeline", LegacySecret)
	managed.Spec.Template.Annotations = map[string]string{RevisionAnnotation: state.Annotations[RevisionAnnotation]}
	a.deployments["ml-pipeline"] = managed
	a.pods = []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "unrelated"}, Status: corev1.PodStatus{Phase: corev1.PodRunning}}}
	requireSuccess(t, m.Reconcile(testContext))
	s.accepted = true
	managed.Spec.Template.Labels["pipeline/runid"] = "real-run"
	requireWaiting(t, m.Reconcile(testContext))
	if s.revokeCalls != 1 {
		t.Fatal("run-labeled pod bypassed draining")
	}
}
func TestDrainRejectsWrongRevisionExemption(t *testing.T) {
	m, a, s := fixture()
	state, err := m.Prepare(testContext)
	requireSuccess(t, err)
	managed := deploymentFixture("ml-pipeline", LegacySecret)
	managed.Spec.Template.Annotations = map[string]string{RevisionAnnotation: state.Annotations[RevisionAnnotation]}
	a.deployments["ml-pipeline"] = managed
	a.pages = []*corev1.PodList{{Items: []corev1.Pod{{ObjectMeta: metav1.ObjectMeta{Name: "not-verified", Labels: map[string]string{"app": "ml-pipeline"}}, Spec: podSpec(LegacySecret)}}}}
	requireWaiting(t, m.Reconcile(testContext))
	if s.revokeCalls != 0 {
		t.Fatal("wrong-revision Pod exempted")
	}
}
func TestDrainPaginationAndBoundedSample(t *testing.T) {
	m, a, s := fixture()
	a.pages = []*corev1.PodList{{ListMeta: metav1.ListMeta{Continue: "1"}, Items: []corev1.Pod{workload("first", corev1.PodRunning)}}, {}}
	for i := 0; i < 8; i++ {
		a.pages[1].Items = append(a.pages[1].Items, workload(fmt.Sprintf("task-%d", i), corev1.PodRunning))
	}
	requireWaiting(t, m.Reconcile(testContext))
	state := a.secrets[AdminSecret]
	if state.Annotations[DrainCountAnnotation] != "9" || strings.Count(state.Annotations[DrainSampleAnnotation], "task-") != 4 {
		t.Fatal("drain count/sample not bounded")
	}
	if s.revokeCalls != 0 {
		t.Fatal("revoked during paginated drain")
	}
}
func TestDrainInventoryFailuresNeverRevoke(t *testing.T) {
	for _, kind := range []string{"nil", "malformed", "api", "repeat", "bound"} {
		t.Run(kind, func(t *testing.T) {
			m, a, s := fixture()
			switch kind {
			case "nil":
				a.pages = []*corev1.PodList{nil}
			case "malformed":
				a.pages = []*corev1.PodList{{Items: []corev1.Pod{{}}}}
			case "api":
				a.listError = errors.New("fixture unavailable")
			case "repeat":
				a.pages = []*corev1.PodList{{ListMeta: metav1.ListMeta{Continue: "1"}}, {ListMeta: metav1.ListMeta{Continue: "1"}}}
			case "bound":
				for i := 0; i < 100; i++ {
					a.pages = append(a.pages, &corev1.PodList{ListMeta: metav1.ListMeta{Continue: strconv.Itoa(i + 1)}})
				}
			}
			requireFailure(t, m.Reconcile(testContext))
			if s.revokeCalls != 0 {
				t.Fatal("revocation after inventory failure")
			}
		})
	}
}
func TestAlreadyRejectedDefaultDoesNotBlockOnWorkloads(t *testing.T) {
	m, a, s := fixture()
	s.accepted = false
	a.pods = []corev1.Pod{workload("active", corev1.PodRunning)}
	requireSuccess(t, m.Reconcile(testContext))
	if a.secrets[AdminSecret].Annotations[PhaseAnnotation] != CompletePhase {
		t.Fatal("completed migration blocked on unrelated drain")
	}
}
func TestDriftReusesCandidateAndRollsConsumersAgain(t *testing.T) {
	m, a, s := fixture()
	requireSuccess(t, m.Reconcile(testContext))
	original := a.secrets[AdminSecret].DeepCopy()
	pair, _, _ := CredentialsFromSecret(original)
	a.secrets[LegacySecret] = secretFixture(LegacySecret, &Credentials{DefaultAccessKey, DefaultSecretKey})
	s.accepted = true
	state, err := m.Prepare(testContext)
	requireSuccess(t, err)
	got, _, _ := CredentialsFromSecret(state)
	if pair != got || state.Annotations[RevisionAnnotation] == original.Annotations[RevisionAnnotation] || state.Annotations[PhaseAnnotation] != PreparedPhase {
		t.Fatal("drift did not create new rollout revision with same credential")
	}
}
func TestCustomLegacyEditIsNotOverwritten(t *testing.T) {
	m, a, s := fixture()
	_, err := m.Prepare(testContext)
	requireSuccess(t, err)
	custom := Credentials{"operator-new", "operator-new-secret"}
	a.secrets[LegacySecret] = secretFixture(LegacySecret, &custom)
	requireFailure(t, m.Reconcile(testContext))
	got, _, _ := CredentialsFromSecret(a.secrets[LegacySecret])
	if got != custom || s.revokeCalls != 0 {
		t.Fatal("overwrote concurrent custom credentials")
	}
}
func TestConcurrentEditAfterDrainPreventsRevocation(t *testing.T) {
	m, a, s := fixture()
	a.afterDrain = func() { a.secrets[LegacySecret].Data = credentialData(Credentials{"concurrent", "custom"}) }
	requireFailure(t, m.Reconcile(testContext))
	if s.revokeCalls != 0 {
		t.Fatal("revoked after concurrent credential edit")
	}
}
func TestRevocationFailureAndWaitRemainIncomplete(t *testing.T) {
	for _, revokeError := range []error{errors.New("unavailable"), &Waiting{Message: "cache reload pending"}} {
		m, a, s := fixture()
		s.revokeError = revokeError
		err := m.Reconcile(testContext)
		if err == nil || IsWaiting(err) != IsWaiting(revokeError) {
			t.Fatal("incorrect readiness classification")
		}
		if a.secrets[AdminSecret].Annotations[PhaseAnnotation] == CompletePhase {
			t.Fatal("marked incomplete revocation complete")
		}
	}
}
func TestClientReuseAndClose(t *testing.T) {
	m, _, s := fixture()
	requireSuccess(t, m.Reconcile(testContext))
	requireSuccess(t, m.Reconcile(testContext))
	if s.closeCalls != 0 {
		t.Fatal("closed reusable SDK client")
	}
	m.Close()
	m.Close()
	if s.closeCalls != 1 {
		t.Fatal("client close not idempotent")
	}
}
func TestPrepareHonorsResourceVersionConflicts(t *testing.T) {
	m, a, _ := fixture()
	state, err := m.Prepare(testContext)
	requireSuccess(t, err)
	_, err = a.PatchSecret(testContext, AdminSecret, "1", nil, map[string]string{"other": "writer"})
	if err == nil {
		t.Fatal("stale version unexpectedly accepted")
	}
	if state.ResourceVersion != "2" {
		t.Fatal("missing optimistic concurrency update")
	}
}
func TestWaitingAndFailureClassification(t *testing.T) {
	if !IsWaiting(fmt.Errorf("wrapped: %w", wait("normal wait"))) {
		t.Fatal("wrapped wait not healthy")
	}
	if IsWaiting(io.EOF) || IsWaiting(nil) {
		t.Fatal("failure/completion classified as wait")
	}
	err := failure("safe progress", errors.New("secret fixture"))
	if strings.Contains(err.Error(), "secret fixture") {
		t.Fatal("unsafe API response leaked")
	}
}

func TestMissingStateMetadataNeverWrites(t *testing.T) {
	for _, field := range []string{"uid", "resourceVersion"} {
		t.Run(field, func(t *testing.T) {
			m, api, _ := fixture()
			if field == "uid" {
				api.secrets[AdminSecret].UID = ""
			} else {
				api.secrets[AdminSecret].ResourceVersion = ""
			}
			_, err := m.Prepare(testContext)
			requireFailure(t, err)
			if api.patches != 0 {
				t.Fatal("wrote credentials without durable concurrency metadata")
			}
		})
	}
}

func TestCompletedStateCustomEditRequiresCoordination(t *testing.T) {
	m, api, _ := fixture()
	requireSuccess(t, m.Reconcile(testContext))
	candidate := api.secrets[AdminSecret].DeepCopy()
	custom := Credentials{"operator-new", "operator-new-secret"}
	api.secrets[LegacySecret] = secretFixture(LegacySecret, &custom)
	patches := api.patches
	_, err := m.Prepare(testContext)
	requireFailure(t, err)
	if api.patches != patches {
		t.Fatal("modified completed state with conflicting custom credentials")
	}
	current, _, _ := CredentialsFromSecret(api.secrets[AdminSecret])
	original, _, _ := CredentialsFromSecret(candidate)
	legacy, _, _ := CredentialsFromSecret(api.secrets[LegacySecret])
	if current != original || legacy != custom || api.secrets[AdminSecret].Annotations[PhaseAnnotation] != CompletePhase {
		t.Fatal("changed administrator or operator credential during conflict")
	}
}

func TestCompletedPrepareIsReadOnly(t *testing.T) {
	m, api, _ := fixture()
	requireSuccess(t, m.Reconcile(testContext))
	patches := api.patches
	state, err := m.Prepare(testContext)
	requireSuccess(t, err)
	if api.patches != patches || state.Annotations[PhaseAnnotation] != CompletePhase {
		t.Fatal("completed bootstrap changed durable state")
	}
}
