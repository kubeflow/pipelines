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
	"reflect"
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
	ktesting "k8s.io/client-go/testing"
)

func bootstrapState(phase string) *corev1.Secret {
	return &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: AdminSecret, Namespace: "kubeflow", UID: types.UID("persistent-state"), Annotations: map[string]string{PhaseAnnotation: phase, RevisionAnnotation: "migration-revision"}}}
}

func TestBootstrapCompletedMigrationNeedsNoJob(t *testing.T) {
	client := fake.NewSimpleClientset()
	if err := EnsureMigrationJob(context.Background(), client, "kubeflow", "image:new", 1000, bootstrapState(CompletePhase)); err != nil {
		t.Fatal(err)
	}
	if len(client.Actions()) != 0 {
		t.Fatal("completed migration unexpectedly called Kubernetes")
	}
}

func TestBootstrapCreatesOnePersistentJob(t *testing.T) {
	ctx := context.Background()
	client := fake.NewSimpleClientset()
	state := bootstrapState(PreparedPhase)
	before := state.DeepCopy()
	for range 2 {
		if err := EnsureMigrationJob(ctx, client, "kubeflow", "image:new", 1000, state); err != nil {
			t.Fatal(err)
		}
	}
	jobs, err := client.BatchV1().Jobs("kubeflow").List(ctx, metav1.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs.Items) != 1 {
		t.Fatalf("want one Job, got %d", len(jobs.Items))
	}
	job := jobs.Items[0]
	if job.Spec.ActiveDeadlineSeconds != nil || job.Spec.TTLSecondsAfterFinished != nil {
		t.Fatal("migration must not have a forced deadline or automatic deletion")
	}
	if job.Spec.Template.Spec.RestartPolicy != corev1.RestartPolicyOnFailure {
		t.Fatal("migration cannot resume after a container failure")
	}
	if len(job.OwnerReferences) != 0 || !reflect.DeepEqual(state, before) {
		t.Fatal("bootstrap must not give credentials a Job owner or mutate them")
	}
	pod := job.Spec.Template.Spec
	if pod.ServiceAccountName != MigrationJobLabel || pod.Containers[0].Command[1] != "--mode=migrate" {
		t.Fatal("wrong migration identity or command")
	}
	if pod.Containers[0].ReadinessProbe.HTTPGet.Path != "/readyz" {
		t.Fatal("missing independent controller readiness")
	}
	if !*pod.Containers[0].SecurityContext.ReadOnlyRootFilesystem || !*pod.SecurityContext.RunAsNonRoot {
		t.Fatal("missing restricted security context")
	}
}

func TestBootstrapUsesAllocatedOpenShiftIdentity(t *testing.T) {
	uid := int64(1000780000)
	client := fake.NewSimpleClientset()
	state := bootstrapState(PreparedPhase)
	if err := EnsureMigrationJob(context.Background(), client, "kubeflow", "image:new", uid, state); err != nil {
		t.Fatal(err)
	}
	job, _ := client.BatchV1().Jobs("kubeflow").Get(context.Background(), MigrationJobName(state), metav1.GetOptions{})
	if got := *job.Spec.Template.Spec.SecurityContext.RunAsUser; got != uid {
		t.Fatalf("allocated UID changed: %d", got)
	}
}

func TestBootstrapForegroundReplacementWaitsBeforeCreating(t *testing.T) {
	for _, reason := range []string{"image", "failed", "incomplete-success"} {
		t.Run(reason, func(t *testing.T) {
			ctx := context.Background()
			state := bootstrapState(DrainingPhase)
			existing := migrationJob("kubeflow", "image:new", 1000, state)
			existing.UID = types.UID("old-job")
			switch reason {
			case "image":
				existing.Spec.Template.Spec.Containers[0].Image = "image:old"
			case "failed":
				existing.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobFailed, Status: corev1.ConditionTrue}}
			case "incomplete-success":
				existing.Status.Conditions = []batchv1.JobCondition{{Type: batchv1.JobComplete, Status: corev1.ConditionTrue}}
			}
			client := fake.NewSimpleClientset(existing)
			deletes := 0
			client.PrependReactor("delete", "jobs", func(action ktesting.Action) (bool, runtime.Object, error) {
				deletes++
				options := action.(ktesting.DeleteAction).GetDeleteOptions()
				if options.PropagationPolicy == nil || *options.PropagationPolicy != metav1.DeletePropagationForeground || options.Preconditions == nil || options.Preconditions.UID == nil || *options.Preconditions.UID != existing.UID {
					t.Fatal("replacement deletion is not foreground and UID-conditional")
				}
				deleting := existing.DeepCopy()
				now := metav1.Now()
				deleting.DeletionTimestamp = &now
				if err := client.Tracker().Update(batchv1.SchemeGroupVersion.WithResource("jobs"), deleting, "kubeflow"); err != nil {
					t.Fatal(err)
				}
				return true, nil, nil
			})
			for range 2 {
				if err := EnsureMigrationJob(ctx, client, "kubeflow", "image:new", 1000, state); !IsWaiting(err) {
					t.Fatalf("want healthy replacement wait, got %v", err)
				}
			}
			if deletes != 1 {
				t.Fatalf("repeated deletion while waiting: %d", deletes)
			}
			for _, action := range client.Actions() {
				if action.GetVerb() == "create" {
					t.Fatal("created replacement before foreground deletion completed")
				}
			}
			if err := client.Tracker().Delete(batchv1.SchemeGroupVersion.WithResource("jobs"), "kubeflow", existing.Name); err != nil {
				t.Fatal(err)
			}
			if err := EnsureMigrationJob(ctx, client, "kubeflow", "image:new", 1000, state); err != nil {
				t.Fatal(err)
			}
			replacement, err := client.BatchV1().Jobs("kubeflow").Get(ctx, existing.Name, metav1.GetOptions{})
			if err != nil || replacement.Spec.Template.Spec.Containers[0].Image != "image:new" {
				t.Fatalf("replacement was not created: %v", err)
			}
		})
	}
}

func TestBootstrapRefusesUnrelatedJobCollision(t *testing.T) {
	state := bootstrapState(PreparedPhase)
	job := migrationJob("kubeflow", "image:old", 1000, state)
	job.UID = "unrelated"
	job.Annotations[StateUIDAnnotation] = "another-secret"
	client := fake.NewSimpleClientset(job)
	err := EnsureMigrationJob(context.Background(), client, "kubeflow", "image:new", 1000, state)
	if err == nil || IsWaiting(err) {
		t.Fatalf("want actionable identity failure, got %v", err)
	}
	for _, action := range client.Actions() {
		if action.GetVerb() != "get" {
			t.Fatalf("mutated unrelated Job: %s", action.GetVerb())
		}
	}
}

func TestBootstrapRejectsMissingIdentityAndRoot(t *testing.T) {
	for _, input := range []struct {
		state *corev1.Secret
		uid   int64
	}{{nil, 1000}, {&corev1.Secret{}, 1000}, {bootstrapState(PreparedPhase), 0}} {
		client := fake.NewSimpleClientset()
		if err := EnsureMigrationJob(context.Background(), client, "kubeflow", "image:new", input.uid, input.state); err == nil || IsWaiting(err) {
			t.Fatalf("invalid bootstrap accepted: %v", err)
		}
		if len(client.Actions()) != 0 {
			t.Fatal("invalid bootstrap called Kubernetes")
		}
	}
}
