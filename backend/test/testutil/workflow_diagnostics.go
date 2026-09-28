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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
)

const (
	diagnosticCommandLimit = 1 << 20
	diagnosticOutputLimit  = 8 << 20
	diagnosticPodLimit     = 20
)

// CollectWorkflowDiagnostics snapshots KFP workflow pods in an isolated test
// namespace before cleanup. It is best-effort and bounded to one minute; callers
// must retain its output without replacing the original test failure.
func CollectWorkflowDiagnostics(namespace string) string {
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	return collectWorkflowDiagnostics(ctx, namespace, runDiagnosticCommand)
}

type diagnosticCommand func(context.Context, ...string) (stdout, stderr []byte, err error)

// cappedDiagnosticBuffer discards excess bytes while allowing the command to
// finish. Bound memory during reads, not after CombinedOutput has allocated it.
type cappedDiagnosticBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (b *cappedDiagnosticBuffer) Write(p []byte) (int, error) {
	n := len(p)
	remaining := b.limit - b.buffer.Len()
	if n > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, _ = b.buffer.Write(p)
	return n, nil
}

func (b *cappedDiagnosticBuffer) Bytes() []byte  { return b.buffer.Bytes() }
func (b *cappedDiagnosticBuffer) String() string { return b.buffer.String() }

func runDiagnosticCommand(ctx context.Context, args ...string) ([]byte, []byte, error) {
	output := &cappedDiagnosticBuffer{limit: diagnosticCommandLimit}
	errors := &cappedDiagnosticBuffer{limit: diagnosticCommandLimit}
	cmd := exec.CommandContext(ctx, "kubectl", args...)
	cmd.WaitDelay = time.Second
	cmd.Stdout, cmd.Stderr = output, errors
	err := cmd.Run()
	if output.truncated || errors.truncated {
		return output.Bytes(), errors.Bytes(), fmt.Errorf("command output truncated at %d bytes (command error: %v)", diagnosticCommandLimit, err)
	}
	return output.Bytes(), errors.Bytes(), err
}

func collectWorkflowDiagnostics(ctx context.Context, namespace string, run diagnosticCommand) string {
	output := &cappedDiagnosticBuffer{limit: diagnosticOutputLimit}
	command := func(args ...string) ([]byte, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		commandCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		args = append([]string{"--namespace=" + namespace, "--request-timeout=10s"}, args...)
		fmt.Fprintf(output, "\n--- kubectl %s ---\n", strings.Join(args, " "))
		data, stderr, err := run(commandCtx, args...)
		_, _ = output.Write(data)
		if len(stderr) > 0 {
			fmt.Fprintf(output, "\nCommand stderr:\n%s", stderr)
		}
		if err != nil {
			fmt.Fprintf(output, "\nDiagnostic command failed: %v\n", err)
		}
		return data, err
	}

	// Capture status first, before potentially large workflow specs exhaust the
	// output budget. Discover pods separately so a missing workflow cannot hide them.
	_, _ = command("get", "workflows.argoproj.io", "-l", "pipeline/runid", "-o",
		`jsonpath={range .items[*]}{.metadata.name}{"\n"}{.status}{"\n"}{end}`)
	_, _ = command("get", "workflows.argoproj.io", "-l", "pipeline/runid", "-o", "yaml")
	data, err := command("get", "pods", "-l", "pipeline/runid", "-o", "json")
	var pods corev1.PodList
	if err == nil {
		err = json.Unmarshal(data, &pods)
	}
	if err != nil {
		fmt.Fprintf(output, "\nUnable to enumerate workflow pods: %v\n", err)
	} else if len(pods.Items) == 0 {
		fmt.Fprintln(output, "\nNo workflow pods found.")
	}
	// Unfinished executors are the most valuable evidence if the budget expires.
	sort.SliceStable(pods.Items, func(i, j int) bool {
		finished := func(p corev1.Pod) bool {
			return p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed
		}
		return !finished(pods.Items[i]) && finished(pods.Items[j])
	})
	if len(pods.Items) > diagnosticPodLimit {
		fmt.Fprintf(output, "\nLimiting pod diagnostics to %d of %d pods.\n", diagnosticPodLimit, len(pods.Items))
		pods.Items = pods.Items[:diagnosticPodLimit]
	}
	for _, pod := range pods.Items {
		if ctx.Err() != nil || output.truncated {
			break
		}
		_, _ = command("describe", "pod", pod.Name)
		containers := append(append([]corev1.Container(nil), pod.Spec.InitContainers...), pod.Spec.Containers...)
		statuses := append(append([]corev1.ContainerStatus(nil), pod.Status.InitContainerStatuses...), pod.Status.ContainerStatuses...)
		for _, container := range containers {
			if ctx.Err() != nil || output.truncated {
				break
			}
			args := []string{"logs", pod.Name, "-c", container.Name, "--timestamps=true", "--tail=200", "--limit-bytes=65536"}
			_, _ = command(args...)
			for _, status := range statuses {
				if status.Name == container.Name && status.RestartCount > 0 {
					_, _ = command(append(args, "--previous=true")...)
					break
				}
			}
		}
	}
	if ctx.Err() != nil {
		fmt.Fprintf(output, "\nDiagnostic collection stopped: %v\n", ctx.Err())
	}
	result := output.String()
	if output.truncated {
		result += "\nDiagnostic output truncated at 8 MiB.\n"
	}
	return result
}
