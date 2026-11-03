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
	"strings"
	"time"

	"github.com/kubeflow/pipelines/backend/test/logger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
)

const imagePullFailureGrace = 2 * time.Minute

type imagePullObservation struct {
	firstSeen     time.Time
	lastPullError string
}

// NewRunImagePullCheck returns a polling check for persistent run image-pull
// failures. Opt in only for tests that do not intentionally fail image pulls.
func NewRunImagePullCheck(client kubernetes.Interface, namespace, runID string) func() error {
	return newRunImagePullCheck(client, namespace, runID, time.Now)
}

func newRunImagePullCheck(client kubernetes.Interface, namespace, runID string, now func() time.Time) func() error {
	firstObserved := make(map[string]imagePullObservation)
	return func() error {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		pods, err := client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: "pipeline/runid=" + runID})
		if err != nil {
			// An observation gap cannot establish a continuous image-pull failure.
			clear(firstObserved)
			logger.Log("Image-pull diagnostics unavailable for run %s: cannot list pods: %v", runID, err)
			return nil
		}
		observed := now()
		current := make(map[string]imagePullObservation)
		for _, pod := range pods.Items {
			if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
				continue
			}
			statuses := append(append([]corev1.ContainerStatus(nil), pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
			for _, status := range statuses {
				waiting := status.State.Waiting
				if waiting == nil || (waiting.Reason != "ErrImagePull" && waiting.Reason != "ImagePullBackOff") {
					continue
				}
				key := string(pod.UID) + "/" + status.Name + "/" + status.Image
				observation, exists := firstObserved[key]
				if !exists {
					observation.firstSeen = observed
				}
				if waiting.Reason == "ErrImagePull" {
					observation.lastPullError = waiting.Message
				}
				current[key] = observation
				if observed.Sub(observation.firstSeen) >= imagePullFailureGrace {
					message := fmt.Sprintf("persistent image-pull failure for pod %s/%s container %s image %s for %s: %s: %s", namespace, pod.Name, status.Name, status.Image, observed.Sub(observation.firstSeen).Round(time.Second), waiting.Reason, waiting.Message)
					if observation.lastPullError != "" && waiting.Reason != "ErrImagePull" {
						message += "; previous ErrImagePull: " + observation.lastPullError
					}
					// Kubelet's BackOff message may omit the registry response. Events
					// are best-effort enrichment, not a prerequisite for diagnosis.
					events, eventErr := client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
						FieldSelector: fields.OneTermEqualSelector("involvedObject.uid", string(pod.UID)).String(),
					})
					if eventErr != nil {
						message += fmt.Sprintf("; pod events unavailable: %v", eventErr)
					} else {
						for _, event := range events.Items {
							if event.InvolvedObject.UID == pod.UID && event.Reason == "Failed" && strings.Contains(event.Message, status.Image) {
								message += "; " + event.Message
							}
						}
					}
					return fmt.Errorf("%s; check the CI runtime-image archive and registry availability", message)
				}
			}
		}
		firstObserved = current
		return nil
	}
}
