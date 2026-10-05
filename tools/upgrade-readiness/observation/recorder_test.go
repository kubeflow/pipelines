// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package readinessobservation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func observer(t *testing.T, duration time.Duration) *recorder {
	t.Helper()
	r, err := newRecorder(filepath.Join(privateDir(t), "observation.json"), duration)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func privateDir(t *testing.T) string {
	t.Helper()
	directory := t.TempDir()
	if err := os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	return directory
}

func readReport(t *testing.T, path string) report {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) > maxBytes {
		t.Fatal("snapshot exceeded byte bound")
	}
	var result report
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestDisabledAndInvalidConfiguration(t *testing.T) {
	t.Setenv("KFP_READINESS_OBSERVATION_FILE", "")
	Start(context.Background())
	Request("create_run", "shared", "tenant", false)
	if active.Load() != nil {
		t.Fatal("observation enabled without explicit opt-in")
	}
	if testing.AllocsPerRun(10, func() { Request("create_run", "shared", "tenant", false) }) != 0 {
		t.Fatal("disabled observer allocated request state")
	}
	path := filepath.Join(t.TempDir(), "observation.json")
	t.Setenv("KFP_READINESS_OBSERVATION_FILE", path)
	for _, value := range []string{"", "0", "-1", "604801", "1h", "nan"} {
		t.Setenv("KFP_READINESS_OBSERVATION_SECONDS", value)
		Start(context.Background())
		if active.Load() != nil {
			t.Fatal("invalid duration enabled observation")
		}
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatal("invalid configuration created an output")
		}
	}
}

func TestOutputRequiresNewFileAndPrivateDirectory(t *testing.T) {
	directory := privateDir(t)
	path := filepath.Join(directory, "observation.json")
	if err := os.WriteFile(path, []byte("preserve"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := newRecorder(path, time.Hour); err == nil {
		t.Fatal("overwrote existing evidence")
	}
	link := filepath.Join(directory, "link")
	if err := os.Symlink(path, link); err != nil {
		t.Fatal(err)
	}
	if _, err := newRecorder(link, time.Hour); err == nil {
		t.Fatal("followed output symlink")
	}
	if _, err := newRecorder("relative.json", time.Hour); err == nil {
		t.Fatal("accepted relative output")
	}
	parentLink := filepath.Join(t.TempDir(), "parent")
	if err := os.Symlink(directory, parentLink); err != nil {
		t.Fatal(err)
	}
	if _, err := newRecorder(filepath.Join(parentLink, "new.json"), time.Hour); err == nil {
		t.Fatal("followed parent directory symlink")
	}
	if err := os.Chmod(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := newRecorder(filepath.Join(directory, "new.json"), time.Hour); err == nil {
		t.Fatal("accepted shared output directory")
	}
	data, _ := os.ReadFile(path)
	if string(data) != "preserve" {
		t.Fatal("changed preexisting file")
	}
}

func TestUploadCapturesOnlyNamespaceAndNoRequestPayload(t *testing.T) {
	r := observer(t, time.Hour)
	active.Store(r)
	defer active.Store(nil)
	request := httptest.NewRequest("POST", "http://private.test/upload?namespace=tenant&name=private-name&token=private-query", strings.NewReader("private-manifest"))
	request.Header.Set("Authorization", "Bearer private-credential")
	Upload("upload_pipeline", "v2beta1", request)
	Upload("upload_pipeline", "v1beta1", httptest.NewRequest("POST", "/upload", nil))
	r.consume(<-r.queue)
	r.consume(<-r.queue)
	r.checkpoint("active")
	result := readReport(t, r.path)
	if result.Records[0].Namespace != "tenant" || *result.Records[0].NamespaceOmitted || !*result.Records[1].NamespaceOmitted {
		t.Fatal("namespace omission/value was not preserved")
	}
	data, _ := os.ReadFile(r.path)
	for _, secret := range []string{"private.test", "private-name", "private-query", "private-manifest", "private-credential", "Authorization"} {
		if strings.Contains(string(data), secret) {
			t.Fatalf("request data leaked: %s", secret)
		}
	}
	if result.Records[0].Caller != "" {
		t.Fatal("upload header was treated as authenticated identity")
	}
}

func TestUploadNamespaceParsingIsBoundedAndMatchesSource(t *testing.T) {
	for _, fixture := range []struct {
		query, namespace string
		omitted, valid   bool
	}{
		{"", "", true, true},
		{"namespace=", "", false, true},
		{"n%61mespace=tenant", "tenant", false, true},
		{"other=private&namespace=first&namespace=second", "first", false, true},
		{"namespace=%zz&namespace=valid", "valid", false, true},
		{"namespace=ignored;value&namespace=valid", "valid", false, true},
		{"other=private", "", true, true},
		{strings.Repeat("x", 8193), "", false, false},
		{strings.Repeat("other=value&", 65) + "namespace=tenant", "", false, false},
	} {
		namespace, omitted, valid := uploadNamespace(fixture.query)
		if namespace != fixture.namespace || omitted != fixture.omitted || valid != fixture.valid {
			t.Fatalf("unexpected namespace interpretation: %q %t %t", namespace, omitted, valid)
		}
	}
	r := observer(t, time.Hour)
	active.Store(r)
	defer active.Store(nil)
	request := httptest.NewRequest("POST", "/upload?"+strings.Repeat("private=value&", 10000), nil)
	Upload("upload_pipeline", "v2beta1", request)
	r.consume(<-r.queue)
	if r.state.Health.InvalidFields != 1 || r.state.Records[0].NamespaceOmitted != nil {
		t.Fatal("bounded parsing gap was reported as namespace omission")
	}
	r.closed.Store(true)
	allocations := testing.AllocsPerRun(10, func() { Upload("upload_pipeline", "v2beta1", request) })
	if allocations != 0 {
		t.Fatalf("inactive observer performed upload parsing: %f allocations", allocations)
	}
	if testing.AllocsPerRun(10, func() { Request("create_run", "shared", "tenant", false) }) != 0 {
		t.Fatal("inactive observer allocated request state")
	}
}

func TestNonblockingBoundedQueueEvenWhenWriterFails(t *testing.T) {
	r := observer(t, time.Hour)
	active.Store(r)
	defer active.Store(nil)
	for range queueCapacity + 10 {
		Authorization("user@example.com", "tenant", "runs", "create", "allowed")
	}
	if len(r.queue) != queueCapacity || r.dropped.Load() != 10 {
		t.Fatal("queue bounds or dropped counter incorrect")
	}
	r.write = func(string, []byte) error { return errors.New("private filesystem error") }
	r.checkpoint("active")
	if r.state.Health.WriteFailures != 1 {
		t.Fatal("writer failure was not counted")
	}
	r.write = atomicSnapshot
	r.consume(<-r.queue)
	r.checkpoint("active")
	result := readReport(t, r.path)
	if result.Health.WriteFailures != 1 || result.Health.DroppedQueue != 10 || result.Records[0].Caller != "user@example.com" {
		t.Fatal("recovered snapshot lost failure/identity evidence")
	}
}

func TestRequestDoesNotWaitForBlockedWriter(t *testing.T) {
	r := observer(t, time.Hour)
	active.Store(r)
	defer active.Store(nil)
	writing := make(chan struct{})
	release := make(chan struct{})
	r.write = func(string, []byte) error {
		select {
		case <-writing:
		default:
			close(writing)
		}
		<-release
		return errors.New("writer unavailable")
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.run(ctx)
	<-writing
	finished := make(chan struct{})
	go func() {
		for range 1000 {
			Request("create_run", "shared", "tenant", false)
		}
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		close(release)
		t.Fatal("request waited for observer disk IO")
	}
	close(release)
	cancel()
	<-r.done
	if r.dropped.Load() == 0 {
		t.Fatal("blocked writer did not disclose dropped observations")
	}
}

func TestRecordBoundAndInvalidFields(t *testing.T) {
	r := observer(t, time.Hour)
	for range maxRecords + 7 {
		r.consume(event{Operation: "authorization", Caller: strings.Repeat("a", 256), Namespace: "tenant", Resource: "runs", Verb: "create", Result: "allowed"})
	}
	r.consume(event{Operation: "private-operation"})
	r.checkpoint("active")
	result := readReport(t, r.path)
	if len(result.Records) != maxRecords || result.Health.DroppedLimit != 7 || result.Health.InvalidFields != 1 || result.Operations["authorization"] != maxRecords+7 {
		t.Fatal("bounded record counts are incorrect")
	}
	other := observer(t, time.Hour)
	other.enqueue(event{Operation: "authorization", Caller: strings.Repeat("private", 10000), Namespace: "unsafe/path", Resource: "private-resource", Verb: "private-verb", Result: "private-error"})
	other.consume(<-other.queue)
	other.checkpoint("active")
	safe := readReport(t, other.path)
	if safe.Records[0].Caller != "" || safe.Records[0].Namespace != "" || safe.Records[0].Resource != "other" || safe.Records[0].Verb != "other" || safe.Records[0].Result != "incomplete" || safe.Health.InvalidFields != 3 {
		t.Fatal("unbounded values were retained")
	}
}

func TestCompletedIntervalAndInactiveOperations(t *testing.T) {
	r := observer(t, 30*time.Millisecond)
	r.enqueue(event{Operation: "create_run", APIVersion: "shared"})
	go r.run(context.Background())
	select {
	case <-r.done:
	case <-time.After(time.Second):
		t.Fatal("observer failed to complete bounded interval")
	}
	result := readReport(t, r.path)
	if result.Status != "completed" || result.EndedAt == "" || result.CheckpointAt == "" || result.SourceRevision != SourceRevision {
		t.Fatal("interval/source provenance missing")
	}
	if result.Operations["create_run"] != 1 || oneOf("create_run", result.UnobservedOperations...) || !oneOf("read_run_log", result.UnobservedOperations...) {
		t.Fatal("inactive operations were not reported")
	}
	for _, required := range []string{"frontend_operations", "caller_groups", "authenticated_identity_on_bypasses", "target_policy_evaluation"} {
		if !oneOf(required, result.UnsupportedChecks...) {
			t.Fatal("unsupported coverage missing: " + required)
		}
	}
	info, err := os.Stat(r.path)
	if err != nil || info.Mode().Perm() != 0600 {
		t.Fatal("snapshot permissions are not private")
	}
	r.enqueue(event{Operation: "retry_run"})
	if len(r.queue) != 0 {
		t.Fatal("accepted observations after interval")
	}
}

func TestShutdownDrainsQueueAndMarksInterrupted(t *testing.T) {
	t.Setenv("KFP_READINESS_OBSERVATION_FILE", filepath.Join(privateDir(t), "observation.json"))
	t.Setenv("KFP_READINESS_OBSERVATION_SECONDS", "60")
	Start(context.Background())
	r := active.Load()
	if r == nil {
		t.Fatal("valid opt-in failed")
	}
	Authorization("user@example.com", "tenant", "runs", "create", "denied")
	Shutdown()
	result := readReport(t, r.path)
	if result.Status != "interrupted" || result.Operations["authorization"] != 1 || len(result.Records) != 1 {
		t.Fatal("shutdown did not flush or misrepresented interval completion")
	}
	if active.Load() != nil {
		t.Fatal("shutdown retained active observer")
	}
}
