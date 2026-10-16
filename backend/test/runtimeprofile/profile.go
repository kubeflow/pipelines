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

// Package runtimeprofile captures opt-in E2E timing evidence before run cleanup.
package runtimeprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

type phase struct {
	Name    string    `json:"name"`
	Started time.Time `json:"started_at"`
	Seconds float64   `json:"seconds"`
}

// Profile is local to a selected spec; nil profiles perform no extra work.
type Profile struct {
	Invocation   map[string]string `json:"invocation,omitempty"`
	bytesWritten int64
	filesWritten int

	Directory string   `json:"-"`
	Fixture   string   `json:"fixture"`
	Phases    []phase  `json:"phases"`
	Errors    []string `json:"errors,omitempty"`
	RunIDs    []string `json:"run_ids,omitempty"`
	command   func(context.Context, ...string) ([]byte, error)
}

// Start enables profiling only for the selected slow fixtures and an explicit directory.
func Start(fixture string) *Profile {
	root := os.Getenv("KFP_E2E_PROFILE_DIR")
	switch filepath.Base(fixture) {
	case "flip_coin.yaml", "collected_parameters.yaml", "if_else_with_oneof_artifacts.yaml":
	default:
		return nil
	}
	if root == "" {
		return nil
	}
	directory, err := os.MkdirTemp(root, filepath.Base(fixture)+"-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "E2E profile unavailable: %v\n", err)
		return nil
	}
	invocation := make(map[string]string)
	for _, key := range []string{"GITHUB_REPOSITORY", "GITHUB_RUN_ID", "GITHUB_RUN_ATTEMPT", "GITHUB_SHA", "SOURCE_SHA"} {
		if value := os.Getenv(key); value != "" {
			invocation[key] = value
		}
	}
	return &Profile{Directory: directory, Fixture: fixture, Invocation: invocation, command: func(ctx context.Context, args ...string) ([]byte, error) {
		return exec.CommandContext(ctx, "kubectl", append([]string{"--request-timeout=5s"}, args...)...).Output()
	}}
}

// Phase measures a harness operation without changing its result or polling behavior.
func (p *Profile) Phase(name string) func() {
	if p == nil {
		return func() {}
	}
	started := time.Now()
	return func() { p.Phases = append(p.Phases, phase{name, started.UTC(), time.Since(started).Seconds()}) }
}

func (p *Profile) save(name string, data []byte) {
	if name != "timings.json" && (p.bytesWritten+int64(len(data)) > 32<<20 || p.filesWritten >= 200) {
		p.Errors = append(p.Errors, name+": evidence limit exceeded; file omitted")
		return
	}
	p.bytesWritten += int64(len(data))
	p.filesWritten++
	if err := os.WriteFile(filepath.Join(p.Directory, name), data, 0600); err != nil {
		p.Errors = append(p.Errors, err.Error())
	}
}

// Metrics optionally snapshots a preconfigured API metrics endpoint; failures are diagnostic only.
func (p *Profile) Metrics(name string) {
	if p == nil || os.Getenv("KFP_E2E_PROFILE_METRICS_URL") == "" {
		return
	}
	defer p.Phase("profile_metrics_" + name)()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, os.Getenv("KFP_E2E_PROFILE_METRICS_URL"), nil)
	if err != nil {
		p.Errors = append(p.Errors, "metrics request invalid")
		return
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		p.Errors = append(p.Errors, "metrics request failed")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		p.Errors = append(p.Errors, fmt.Sprintf("metrics status %d", response.StatusCode))
		return
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		p.Errors = append(p.Errors, "metrics read failed")
		return
	}
	p.save("metrics-"+name+".txt", data)
}

type object struct {
	Metadata struct {
		Name              string `json:"name"`
		Namespace         string `json:"namespace"`
		CreationTimestamp string `json:"creationTimestamp"`
		UID               string `json:"uid"`
	} `json:"metadata"`
	Status json.RawMessage `json:"status"`
}
type objectList struct {
	Items []object `json:"items"`
}

// Event fields deliberately exclude messages and arbitrary metadata. Reasons and
// timestamps identify scheduling/pull/start delays without copying pod data.
type event struct {
	Reason         string `json:"reason"`
	Type           string `json:"type"`
	FirstTimestamp string `json:"firstTimestamp"`
	LastTimestamp  string `json:"lastTimestamp"`
	EventTime      string `json:"eventTime"`
	Count          int    `json:"count"`
	Series         *struct {
		Count            int    `json:"count"`
		LastObservedTime string `json:"lastObservedTime"`
	} `json:"series,omitempty"`
}
type eventList struct {
	Items []event `json:"items"`
}

func (p *Profile) query(ctx context.Context, name string, args ...string) []byte {
	data, err := p.command(ctx, args...)
	if err != nil {
		p.Errors = append(p.Errors, name+": "+err.Error())
		return nil
	}
	return data
}

// Capture saves workflow and pod status plus bounded timestamped logs before cleanup.
// Its 30-second budget and duration are separate from the measured test phases.
func (p *Profile) Capture(namespace string, runIDs []string) {
	if p == nil {
		return
	}
	defer p.Phase("profile_capture")()
	p.RunIDs = append([]string(nil), runIDs...)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for index, runID := range runIDs {
		if ctx.Err() != nil {
			break
		}
		prefix := fmt.Sprintf("run-%d", index)
		data := p.query(ctx, prefix+" workflows", "get", "workflows", "-n", namespace, "-l", "pipeline/runid="+runID, "-o", "json")
		var workflows objectList
		if err := json.Unmarshal(data, &workflows); err != nil {
			p.Errors = append(p.Errors, prefix+": workflow status unavailable")
			continue
		}
		if len(workflows.Items) == 0 {
			p.Errors = append(p.Errors, prefix+": no matching workflows; evidence incomplete")
		}
		snapshot, _ := json.MarshalIndent(workflows, "", "  ")
		p.save(prefix+"-workflows.json", snapshot)
		for wi, workflow := range workflows.Items {
			data = p.query(ctx, prefix+" pods", "get", "pods", "-n", namespace, "-l", "workflows.argoproj.io/workflow="+workflow.Metadata.Name, "-o", "json")
			var pods objectList
			if err := json.Unmarshal(data, &pods); err != nil {
				p.Errors = append(p.Errors, prefix+": pod status unavailable")
				continue
			}
			if len(pods.Items) == 0 {
				p.Errors = append(p.Errors, prefix+": no matching pods; evidence incomplete")
			}
			snapshot, _ = json.MarshalIndent(pods, "", "  ")
			p.save(fmt.Sprintf("%s-%d-pods.json", prefix, wi), snapshot)
			for pi, pod := range pods.Items {
				if ctx.Err() != nil {
					break
				}
				if pod.Metadata.UID == "" {
					p.Errors = append(p.Errors, pod.Metadata.Name+": missing pod UID; event evidence incomplete")
				} else {
					eventName := fmt.Sprintf("%s-%d-pod-%d-events.json", prefix, wi, pi)
					eventData := p.query(ctx, eventName, "get", "events", "-n", namespace, "--field-selector", "involvedObject.uid="+pod.Metadata.UID, "-o", "json")
					var events eventList
					if err := json.Unmarshal(eventData, &events); err != nil {
						p.Errors = append(p.Errors, eventName+": events unavailable")
					} else {
						encoded, _ := json.MarshalIndent(events, "", "  ")
						p.save(eventName, encoded)
					}
				}
				logName := fmt.Sprintf("%s-%d-pod-%d.log", prefix, wi, pi)
				data = p.query(ctx, logName, "logs", "-n", namespace, pod.Metadata.Name, "--all-containers=true", "--prefix=true", "--timestamps=true", "--tail=2000", "--limit-bytes=262144")
				if data != nil {
					p.save(logName, data)
				}
			}
		}
	}
	if ctx.Err() != nil {
		p.Errors = append(p.Errors, "profile capture deadline exceeded; evidence incomplete")
	}
}

// Finish persists the timing report even when evidence collection was incomplete.
func (p *Profile) Finish() {
	if p == nil {
		return
	}
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return
	}
	p.save("timings.json", data)
}

// Run records the already-fetched API result without making another API request.
func (p *Profile) Run(value any) {
	if p == nil {
		return
	}
	defer p.Phase("profile_run_snapshot")()
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		p.Errors = append(p.Errors, "run result could not be encoded")
		return
	}
	p.save("api-run.json", data)
}
