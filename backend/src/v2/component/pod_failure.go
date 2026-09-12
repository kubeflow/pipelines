// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package component

import (
	"context"
	"errors"
	"fmt"
	"os/exec"

	"github.com/kubeflow/pipelines/backend/src/v2/metadata"
	pb "github.com/kubeflow/pipelines/third_party/ml-metadata/go/ml_metadata"
	"google.golang.org/protobuf/types/known/structpb"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const executionErrorCustomPropertyKey = "error"

// attachFailureToExecution adds best-effort, structured failure details to the
// MLMD execution custom properties. This is intended to help users debug pod-
// and container-level failures (e.g., OOMKilled, ImagePullBackOff) without
// requiring kubectl access.
func (l *LauncherV2) attachFailureToExecution(ctx context.Context, execution *metadata.Execution, originalErr error) {
	e := execution.GetExecution()
	if e == nil {
		return
	}
	if e.CustomProperties == nil {
		e.CustomProperties = make(map[string]*pb.Value)
	}

	fields := make(map[string]*structpb.Value)
	fields["message"] = structpb.NewStringValue(originalErr.Error())

	if exitCode, ok := exitCodeFromError(originalErr); ok {
		fields["exitCode"] = structpb.NewNumberValue(float64(exitCode))
	}

	if l.options.Namespace != "" && l.options.PodName != "" && l.clientManager != nil && l.clientManager.K8sClient() != nil {
		pod, perr := l.clientManager.K8sClient().CoreV1().Pods(l.options.Namespace).Get(ctx, l.options.PodName, metav1.GetOptions{})
		if perr == nil && pod != nil {
			if podFields := podFailureFields(pod); len(podFields) > 0 {
				for k, v := range podFields {
					fields[k] = v
				}
				if reason := fields["reason"]; reason != nil && reason.GetStringValue() != "" {
					fields["message"] = structpb.NewStringValue(fmt.Sprintf("%s (kubernetes reason=%s)", originalErr.Error(), reason.GetStringValue()))
				}
			}
		}
	}

	e.CustomProperties[executionErrorCustomPropertyKey] = &pb.Value{
		Value: &pb.Value_StructValue{
			StructValue: &structpb.Struct{Fields: fields},
		},
	}
}

func exitCodeFromError(err error) (int, bool) {
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) && exitErr.ProcessState != nil {
		code := exitErr.ProcessState.ExitCode()
		if code >= 0 {
			return code, true
		}
	}
	return 0, false
}

// podFailureFields returns a best-effort set of structured fields describing a
// pod/container failure. It intentionally returns a small, stable set of keys
// to keep downstream consumers simple.
func podFailureFields(pod *corev1.Pod) map[string]*structpb.Value {
	if pod == nil {
		return nil
	}

	best := bestContainerFailure(pod)

	fields := map[string]*structpb.Value{
		"podName":   structpb.NewStringValue(pod.GetName()),
		"namespace": structpb.NewStringValue(pod.GetNamespace()),
	}

	if best.reason == "" && pod.Status.Reason != "" {
		best.reason = pod.Status.Reason
	}
	if best.message == "" && pod.Status.Message != "" {
		best.message = pod.Status.Message
	}

	if best.containerName != "" {
		fields["containerName"] = structpb.NewStringValue(best.containerName)
	}
	if best.reason != "" {
		fields["reason"] = structpb.NewStringValue(best.reason)
	}
	if best.message != "" {
		fields["kubernetesMessage"] = structpb.NewStringValue(best.message)
	}
	if best.exitCode != nil {
		fields["containerExitCode"] = structpb.NewNumberValue(float64(*best.exitCode))
	}

	if len(fields) <= 2 {
		return nil
	}
	return fields
}

type containerFailure struct {
	containerName string
	reason        string
	message       string
	exitCode      *int32
}

// bestContainerFailure finds the most relevant failure from a pod's container
// statuses, prioritizing waiting reasons (ImagePullBackOff, CrashLoopBackOff)
// and terminated reasons (OOMKilled, Error) over generic pod status.
func bestContainerFailure(pod *corev1.Pod) containerFailure {
	consider := func(name string, state corev1.ContainerState) containerFailure {
		if state.Waiting != nil {
			return containerFailure{
				containerName: name,
				reason:        state.Waiting.Reason,
				message:       state.Waiting.Message,
			}
		}
		if state.Terminated != nil {
			exitCode := state.Terminated.ExitCode
			return containerFailure{
				containerName: name,
				reason:        state.Terminated.Reason,
				message:       state.Terminated.Message,
				exitCode:      &exitCode,
			}
		}
		return containerFailure{}
	}

	for _, cs := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
		c := consider(cs.Name, cs.State)
		if c.reason != "" && cs.State.Waiting != nil {
			return c
		}
	}

	for _, cs := range append(pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses...) {
		c := consider(cs.Name, cs.State)
		if c.reason != "" && cs.State.Terminated != nil {
			return c
		}
		c = consider(cs.Name, cs.LastTerminationState)
		if c.reason != "" {
			return c
		}
	}

	return containerFailure{}
}
