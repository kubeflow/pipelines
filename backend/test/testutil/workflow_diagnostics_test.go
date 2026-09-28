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
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCollectWorkflowDiagnostics(t *testing.T) {
	pods := corev1.PodList{Items: []corev1.Pod{
		{ObjectMeta: metav1.ObjectMeta{Name: "driver"}, Status: corev1.PodStatus{Phase: corev1.PodSucceeded}},
		{
			ObjectMeta: metav1.ObjectMeta{Name: "executor"},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "init"}},
				Containers:     []corev1.Container{{Name: "main"}, {Name: "wait"}},
			},
			Status: corev1.PodStatus{
				Phase:                 corev1.PodPending,
				InitContainerStatuses: []corev1.ContainerStatus{{Name: "init", RestartCount: 1}},
				ContainerStatuses:     []corev1.ContainerStatus{{Name: "main", RestartCount: 1}},
			},
		},
	}}
	data, err := json.Marshal(pods)
	require.NoError(t, err)
	var commands []string
	output := collectWorkflowDiagnostics(context.Background(), "test-ns", func(ctx context.Context, args ...string) ([]byte, []byte, error) {
		deadline, ok := ctx.Deadline()
		require.True(t, ok)
		require.LessOrEqual(t, time.Until(deadline), 10*time.Second)
		require.Equal(t, []string{"--namespace=test-ns", "--request-timeout=10s"}, args[:2])
		command := strings.Join(args[2:], " ")
		commands = append(commands, command)
		switch {
		case command == "get pods -l pipeline/runid -o json":
			return data, []byte("successful kubectl warning"), nil
		case strings.HasPrefix(command, "get workflows"):
			return nil, []byte("workflow unavailable"), errors.New("workflow request failed")
		case command == "describe pod executor":
			return []byte("Events: Failed to pull image"), nil, nil
		case strings.Contains(command, "-c init"):
			return nil, nil, errors.New("container not started")
		default:
			return []byte("diagnostic evidence"), nil, nil
		}
	})
	require.Contains(t, commands[0], "jsonpath=")
	require.Equal(t, "get workflows.argoproj.io -l pipeline/runid -o yaml", commands[1])
	require.Equal(t, "describe pod executor", commands[3])
	for _, container := range []string{"init", "main", "wait"} {
		require.Contains(t, commands, "logs executor -c "+container+" --timestamps=true --tail=200 --limit-bytes=65536")
	}
	for _, container := range []string{"init", "main"} {
		require.Contains(t, commands, "logs executor -c "+container+" --timestamps=true --tail=200 --limit-bytes=65536 --previous=true")
	}
	require.NotContains(t, strings.Join(commands, "\n"), "-c wait --timestamps=true --tail=200 --limit-bytes=65536 --previous=true")
	require.Contains(t, output, "Diagnostic command failed: workflow request failed")
	require.Contains(t, output, "Events: Failed to pull image")
	require.Contains(t, output, "container not started")
	require.Contains(t, output, "diagnostic evidence")
	require.Contains(t, output, "successful kubectl warning")
}

func TestWorkflowDiagnosticsDiscoveryFailures(t *testing.T) {
	for _, tc := range []struct{ response, want string }{
		{`{"items":[]}`, "No workflow pods found"},
		{`invalid json`, "Unable to enumerate workflow pods"},
	} {
		t.Run(tc.want, func(t *testing.T) {
			calls := 0
			output := collectWorkflowDiagnostics(context.Background(), "ns", func(context.Context, ...string) ([]byte, []byte, error) {
				calls++
				return []byte(tc.response), nil, nil
			})
			require.Equal(t, 3, calls)
			require.Contains(t, output, tc.want)
		})
	}
}

func TestWorkflowDiagnosticsBudget(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	output := collectWorkflowDiagnostics(ctx, "ns", func(ctx context.Context, _ ...string) ([]byte, []byte, error) {
		calls++
		cancel()
		return nil, nil, ctx.Err()
	})
	require.Equal(t, 1, calls)
	require.Contains(t, output, "Diagnostic collection stopped: context canceled")
}

func TestWorkflowDiagnosticsPodLimit(t *testing.T) {
	pods := corev1.PodList{Items: make([]corev1.Pod, diagnosticPodLimit+1)}
	data, err := json.Marshal(pods)
	require.NoError(t, err)
	calls := 0
	output := collectWorkflowDiagnostics(context.Background(), "ns", func(context.Context, ...string) ([]byte, []byte, error) {
		calls++
		return data, nil, nil
	})
	require.Equal(t, 3+diagnosticPodLimit, calls)
	require.Contains(t, output, "Limiting pod diagnostics to 20 of 21 pods")
}

func TestCappedDiagnosticBuffer(t *testing.T) {
	b := &cappedDiagnosticBuffer{limit: 5}
	for _, chunk := range []string{"123", "456", "789"} {
		n, err := b.Write([]byte(chunk))
		require.NoError(t, err)
		require.Equal(t, len(chunk), n)
	}
	require.Equal(t, "12345", b.String())
	require.True(t, b.truncated)
}

func TestWorkflowDiagnosticsOutputLimit(t *testing.T) {
	pods := corev1.PodList{Items: make([]corev1.Pod, diagnosticPodLimit)}
	data, err := json.Marshal(pods)
	require.NoError(t, err)
	calls := 0
	output := collectWorkflowDiagnostics(context.Background(), "ns", func(_ context.Context, args ...string) ([]byte, []byte, error) {
		calls++
		if args[3] == "pods" {
			return data, nil, nil
		}
		return []byte(strings.Repeat("x", diagnosticCommandLimit)), nil, nil
	})
	require.Contains(t, output, "Diagnostic output truncated at 8 MiB")
	require.Less(t, len(output), diagnosticOutputLimit+100)
	require.Less(t, calls, diagnosticPodLimit)
}

func TestDiagnosticCommandKeepsStderrSeparate(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	script := "#!/bin/sh\nprintf '{\"items\":[]}'\nprintf 'warning' >&2\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "kubectl"), []byte(script), 0700))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stdout, stderr, err := runDiagnosticCommand(ctx)
	require.NoError(t, err)
	require.JSONEq(t, `{"items":[]}`, string(stdout))
	require.Equal(t, "warning", string(stderr))
}

func TestDiagnosticCommandDeadlineAndOutputCap(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	for _, tc := range []struct {
		name, script, want string
		timeout            time.Duration
	}{
		{"deadline", "exec sleep 30", "", 100 * time.Millisecond},
		{"output cap", "head -c 2097152 /dev/zero", "output truncated", 5 * time.Second},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, os.WriteFile(filepath.Join(dir, "kubectl"), []byte("#!/bin/sh\n"+tc.script+"\n"), 0700))
			ctx, cancel := context.WithTimeout(context.Background(), tc.timeout)
			defer cancel()
			start := time.Now()
			output, _, err := runDiagnosticCommand(ctx)
			require.Error(t, err)
			if tc.want != "" {
				require.ErrorContains(t, err, tc.want)
			} else {
				require.ErrorIs(t, ctx.Err(), context.DeadlineExceeded)
			}
			require.LessOrEqual(t, len(output), diagnosticCommandLimit)
			require.Less(t, time.Since(start), 5*time.Second)
		})
	}
}
