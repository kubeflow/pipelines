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

package end2end

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/api/v2/go_http_client/run_model"
	commonutil "github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/kubeflow/pipelines/backend/test/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
)

var _ = Describe("Coordinator lease recovery", Serial, Label("CoordinatorRuntimeMultiReplica"), func() {
	It("recovers an active attempt on another API replica without repeating computation", func() {
		Expect(testutil.ShouldUseCoordinatorRuntimeSelector()).To(BeTrue(), "this label requires the coordinator runtime")
		ctx := context.Background()
		namespace := testutil.GetNamespace()
		deployments := k8Client.AppsV1().Deployments(namespace)
		deployment, err := deployments.Get(ctx, "ml-pipeline", metav1.GetOptions{})
		Expect(err).NotTo(HaveOccurred())
		originalReplicas := int32(1)
		if deployment.Spec.Replicas != nil {
			originalReplicas = *deployment.Spec.Replicas
		}
		DeferCleanup(func() {
			patch := []byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, originalReplicas))
			_, err := deployments.Patch(ctx, deployment.Name, types.MergePatchType, patch, metav1.PatchOptions{})
			Expect(err).NotTo(HaveOccurred())
		})
		_, err = deployments.Patch(ctx, deployment.Name, types.MergePatchType, []byte(`{"spec":{"replicas":2}}`), metav1.PatchOptions{})
		Expect(err).NotTo(HaveOccurred())
		Eventually(func(g Gomega) {
			current, err := deployments.Get(ctx, deployment.Name, metav1.GetOptions{})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(current.Status.ObservedGeneration).To(BeNumerically(">=", current.Generation))
			g.Expect(current.Status.ReadyReplicas).To(Equal(int32(2)))
			g.Expect(current.Status.Replicas).To(Equal(int32(2)))
		}, 3*time.Minute, time.Second).Should(Succeed())

		// The Kubernetes Service proxy survives deletion of either API Pod; a
		// kubectl port-forward can instead remain attached to the deleted Pod.
		run := &run_model.V2Run{}
		body, err := json.Marshal(map[string]any{
			"display_name":    fmt.Sprintf("lease-recovery-%d", time.Now().UnixNano()),
			"experiment_id":   *experimentID,
			"service_account": testutil.GetDefaultPipelineRunnerServiceAccount(),
			"pipeline_spec":   leaseRecoveryPipeline(),
		})
		Expect(err).NotTo(HaveOccurred())
		created, err := replicaAPIRequest(ctx, "POST", "runs", body)
		Expect(err).NotTo(HaveOccurred())
		Expect(json.Unmarshal(created, run)).To(Succeed())
		Expect(run.RunID).NotTo(BeEmpty())
		DeferCleanup(func() {
			_, err := replicaAPIRequest(ctx, "DELETE", "runs/"+run.RunID, nil)
			Expect(err).NotTo(HaveOccurred())
			_, err = replicaAPIRequest(ctx, "DELETE", "experiments/"+*experimentID, nil)
			Expect(err).NotTo(HaveOccurred())
		})

		pods, err := k8Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
			LabelSelector: commonutil.LabelKeyWorkflowRunId + "=" + run.RunID,
		})
		Expect(err).NotTo(HaveOccurred())
		observedPods := make(map[types.UID]replicaPodObservation)
		for _, pod := range pods.Items {
			observedPods[pod.UID] = replicaPodObservation{Pod: pod}
		}
		// Publication cleanup removes terminal Pods, so retain every UID and its
		// last status rather than counting only the Pods left after completion.
		podWatch, err := k8Client.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{
			LabelSelector:   commonutil.LabelKeyWorkflowRunId + "=" + run.RunID,
			ResourceVersion: pods.ResourceVersion,
		})
		Expect(err).NotTo(HaveOccurred())
		DeferCleanup(podWatch.Stop)

		var workload corev1.Pod
		Eventually(func(g Gomega) {
			g.Expect(recordReplicaPodEvents(podWatch.ResultChan(), observedPods)).To(Succeed())
			pods, err := k8Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
				LabelSelector: commonutil.LabelKeyWorkflowRunId + "=" + run.RunID,
			})
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(pods.Items).To(HaveLen(1))
			workload = pods.Items[0]
			g.Expect(workload.Status.Phase).To(Equal(corev1.PodRunning))
		}, 2*time.Minute, time.Second).Should(Succeed())

		var owner corev1.Pod
		Eventually(func(g Gomega) {
			pods, err := k8Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "app=ml-pipeline"})
			g.Expect(err).NotTo(HaveOccurred())
			found := false
			for _, pod := range pods.Items {
				if pod.DeletionTimestamp != nil {
					continue
				}
				logs, err := k8Client.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{Container: "ml-pipeline-api-server"}).DoRaw(ctx)
				g.Expect(err).NotTo(HaveOccurred())
				if strings.Contains(string(logs), "acquired run "+run.RunID+" ") {
					owner, found = pod, true
				}
			}
			g.Expect(found).To(BeTrue(), "the lease holder must be identifiable in worker acquisition diagnostics")
		}, 30*time.Second, time.Second).Should(Succeed())
		ownerUID := owner.UID
		Expect(k8Client.CoreV1().Pods(namespace).Delete(ctx, owner.Name, metav1.DeleteOptions{
			GracePeriodSeconds: new(int64),
			Preconditions:      &metav1.Preconditions{UID: &ownerUID},
		})).To(Succeed())

		Eventually(func(g Gomega) {
			g.Expect(recordReplicaPodEvents(podWatch.ResultChan(), observedPods)).To(Succeed())
			data, err := replicaAPIRequest(ctx, "GET", "runs/"+run.RunID+"?view=FULL", nil)
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(json.Unmarshal(data, run)).To(Succeed())
			g.Expect(run.State).NotTo(Equal(run_model.V2RuntimeStateFAILED.Pointer()), "run %s failed during takeover", run.RunID)
			g.Expect(run.State).To(Equal(run_model.V2RuntimeStateSUCCEEDED.Pointer()))
		}, 5*time.Minute, 2*time.Second).Should(Succeed())

		Expect(run.Tasks).To(ContainElement(SatisfyAll(
			HaveField("Name", "consumer"),
			HaveField("State", Equal(run_model.PipelineTaskTaskStateSUCCEEDED.Pointer())),
		)), "the new owner must schedule downstream work, not merely observe the producer")
		var consumerTaskID string
		for _, task := range run.Tasks {
			if task.Name == "consumer" {
				consumerTaskID = task.TaskID
			}
		}
		Expect(consumerTaskID).NotTo(BeEmpty())
		Eventually(func(g Gomega) {
			g.Expect(recordReplicaPodEvents(podWatch.ResultChan(), observedPods)).To(Succeed())
			g.Expect(replicaConsumerCleanupObserved(observedPods, consumerTaskID)).To(BeTrue(),
				"the watch must reach successful consumer deletion before the final Pod count is authoritative")
			g.Expect(observedPods).To(HaveLen(2), "the producer must not be submitted again during takeover")
			g.Expect(observedPods).To(HaveKey(workload.UID), "the original workload must survive loss of its coordinator")
			g.Expect(observedPods[workload.UID].Deleted).To(BeTrue())
			pod := observedPods[workload.UID].Pod
			g.Expect(pod.Status.Phase).To(Equal(corev1.PodSucceeded))
			for _, container := range pod.Status.ContainerStatuses {
				g.Expect(container.RestartCount).To(BeZero())
			}
		}, 30*time.Second, 100*time.Millisecond).Should(Succeed())
		AddReportEntry("recovered run", run)
		AddReportEntry("terminated owner", owner.Name)
		AddReportEntry("preserved workload UID", workload.UID)
		AddReportEntry("observed workload lifecycles", observedPods)
	})
})

type replicaPodObservation struct {
	Pod     corev1.Pod
	Deleted bool
}

// recordReplicaPodEvents retains deleted workloads so cleanup cannot hide a replay.
func recordReplicaPodEvents(events <-chan watch.Event, pods map[types.UID]replicaPodObservation) error {
	for {
		select {
		case event, open := <-events:
			if !open {
				return fmt.Errorf("workload Pod watch closed before takeover observation completed")
			}
			switch event.Type {
			case watch.Error:
				return apierrors.FromObject(event.Object)
			case watch.Added, watch.Modified, watch.Deleted:
				pod, ok := event.Object.(*corev1.Pod)
				if !ok {
					return fmt.Errorf("workload Pod watch returned %T instead of a Pod", event.Object)
				}
				pods[pod.UID] = replicaPodObservation{Pod: *pod.DeepCopy(), Deleted: event.Type == watch.Deleted}
			}
		default:
			return nil
		}
	}
}

// replicaConsumerCleanupObserved establishes a stream barrier after downstream execution.
func replicaConsumerCleanupObserved(pods map[types.UID]replicaPodObservation, taskID string) bool {
	for _, observation := range pods {
		if observation.Pod.Labels["pipelines.kubeflow.org/task_id"] == taskID &&
			observation.Pod.Status.Phase == corev1.PodSucceeded && observation.Deleted {
			return true
		}
	}
	return false
}

// replicaAPIRequest routes through the Service rather than one ephemeral API Pod.
func replicaAPIRequest(ctx context.Context, method, path string, body []byte) ([]byte, error) {
	resourcePath, query, _ := strings.Cut(path, "?")
	request := k8Client.CoreV1().RESTClient().Verb(method).
		Namespace(testutil.GetNamespace()).Resource("services").Name("ml-pipeline:8888").
		SubResource("proxy").Suffix("apis/v2", resourcePath)
	if query == "view=FULL" {
		request = request.Param("view", "FULL")
	}
	if body != nil {
		request = request.Body(body).SetHeader("Content-Type", "application/json")
	}
	data, err := request.DoRaw(ctx)
	if method == "DELETE" && apierrors.IsNotFound(err) {
		return nil, nil
	}
	return data, err
}

// leaseRecoveryPipeline keeps computation alive through a lease expiry and tests
// that its result still enables a downstream task on the replacement owner.
func leaseRecoveryPipeline() map[string]any {
	parameter := map[string]any{"parameterType": "STRING"}
	return map[string]any{
		"pipelineInfo":  map[string]any{"name": "lease-recovery"},
		"schemaVersion": "2.1.0",
		"components": map[string]any{
			"producer": map[string]any{
				"executorLabel":     "producer",
				"outputDefinitions": map[string]any{"parameters": map[string]any{"value": parameter}},
			},
			"consumer": map[string]any{
				"executorLabel":    "consumer",
				"inputDefinitions": map[string]any{"parameters": map[string]any{"upstream": parameter}},
			},
		},
		"deploymentSpec": map[string]any{"executors": map[string]any{
			"producer": map[string]any{"container": map[string]any{
				"image": "docker.io/alpine:3.23", "command": []string{"sh", "-ec"},
				"args": []string{`echo started; sleep 90; mkdir -p "$(dirname "$0")"; printf done > "$0"`, "{{$.outputs.parameters['value'].output_file}}"},
			}},
			"consumer": map[string]any{"container": map[string]any{
				"image": "docker.io/alpine:3.23", "command": []string{"sh", "-ec"},
				"args": []string{`test "$0" = done; echo recovered`, "{{$.inputs.parameters['upstream']}}"},
			}},
		}},
		"root": map[string]any{"dag": map[string]any{"tasks": map[string]any{
			"producer": map[string]any{
				"taskInfo":     map[string]any{"name": "producer"},
				"componentRef": map[string]any{"name": "producer"},
			},
			"consumer": map[string]any{
				"taskInfo":       map[string]any{"name": "consumer"},
				"componentRef":   map[string]any{"name": "consumer"},
				"dependentTasks": []string{"producer"},
				"inputs": map[string]any{"parameters": map[string]any{"upstream": map[string]any{
					"taskOutputParameter": map[string]any{"producerTask": "producer", "outputParameterKey": "value"},
				}}},
			},
		}}},
	}
}
