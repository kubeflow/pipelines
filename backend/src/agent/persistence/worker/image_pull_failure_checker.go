// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

const (
	// argoWorkflowLabelKey is the label Argo sets on pods to identify the parent workflow.
	argoWorkflowLabelKey = "workflows.argoproj.io/workflow"
)

// ImagePullFailureCheckerInterface checks workflow pods for image pull failures
// and terminates the workflow if the grace period has elapsed.
type ImagePullFailureCheckerInterface interface {
	CheckAndTerminate(namespace string, workflowName string) error
}

// ImagePullFailureChecker checks pods belonging to a workflow for image pull
// failures and terminates the workflow after a configurable grace period.
type ImagePullFailureChecker struct {
	kubeClient      kubernetes.Interface
	executionClient util.ExecutionClient
	gracePeriod     time.Duration
}

// NewImagePullFailureChecker creates a new checker.
func NewImagePullFailureChecker(
	kubeClient kubernetes.Interface,
	executionClient util.ExecutionClient,
	gracePeriod time.Duration,
) *ImagePullFailureChecker {
	return &ImagePullFailureChecker{
		kubeClient:      kubeClient,
		executionClient: executionClient,
		gracePeriod:     gracePeriod,
	}
}

// CheckAndTerminate lists pods for the given workflow and terminates the workflow
// if any pod has been stuck in ImagePullBackOff or ErrImagePull longer than the
// grace period (measured from pod creation time).
func (c *ImagePullFailureChecker) CheckAndTerminate(namespace string, workflowName string) error {
	ctx := context.TODO()

	pods, err := c.kubeClient.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", argoWorkflowLabelKey, workflowName),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods for workflow %s/%s: %w", namespace, workflowName, err)
	}

	for i := range pods.Items {
		pod := &pods.Items[i]
		failedImage := getImagePullFailure(pod)
		if failedImage == "" {
			continue
		}

		podAge := time.Since(pod.CreationTimestamp.Time)
		if podAge < c.gracePeriod {
			log.Debugf("Pod %s/%s has image pull failure for %q (age: %v), waiting for grace period (%v)",
				pod.Namespace, pod.Name, failedImage, podAge.Round(time.Second), c.gracePeriod)
			continue
		}

		log.Infof("Terminating workflow %s/%s: pod %s has image pull failure for %q (age: %v exceeds grace period %v)",
			namespace, workflowName, pod.Name, failedImage, podAge.Round(time.Second), c.gracePeriod)
		return c.terminateWorkflow(ctx, namespace, workflowName)
	}

	return nil
}

// getImagePullFailure checks if any container in the pod has an image pull failure.
// Returns the failed image name, or empty string if no failure is found.
func getImagePullFailure(pod *corev1.Pod) string {
	for _, status := range pod.Status.InitContainerStatuses {
		if image := imagePullFailureFromStatus(status); image != "" {
			return image
		}
	}
	for _, status := range pod.Status.ContainerStatuses {
		if image := imagePullFailureFromStatus(status); image != "" {
			return image
		}
	}
	return ""
}

// imagePullFailureFromStatus returns the image name if the container is in
// ImagePullBackOff or ErrImagePull state, empty string otherwise.
func imagePullFailureFromStatus(status corev1.ContainerStatus) string {
	if status.State.Waiting != nil {
		reason := status.State.Waiting.Reason
		if reason == "ImagePullBackOff" || reason == "ErrImagePull" {
			return status.Image
		}
	}
	return ""
}

// terminateWorkflow terminates an Argo workflow by setting activeDeadlineSeconds to 0.
func (c *ImagePullFailureChecker) terminateWorkflow(ctx context.Context, namespace string, workflowName string) error {
	if c.executionClient == nil {
		return fmt.Errorf("execution client not configured, cannot terminate workflow %s/%s", namespace, workflowName)
	}

	patchObj := util.GetTerminatePatch(util.CurrentExecutionType())
	if patchObj == nil {
		return fmt.Errorf("unsupported execution type for termination")
	}

	patchBytes, err := json.Marshal(patchObj)
	if err != nil {
		return fmt.Errorf("failed to marshal termination patch: %w", err)
	}

	_, err = c.executionClient.Execution(namespace).Patch(
		ctx, workflowName, types.MergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return fmt.Errorf("failed to patch workflow %s/%s: %w", namespace, workflowName, err)
	}

	log.Infof("Successfully terminated workflow %s/%s due to image pull failure", namespace, workflowName)
	return nil
}
