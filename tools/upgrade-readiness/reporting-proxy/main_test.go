// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
// Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.

package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic/fake"
	ktesting "k8s.io/client-go/testing"
)

const config = `[{"run_id":"run-id","workflow_name":"selected-wf","workflow_uid":"workflow-uid"}]`

type reportClient struct {
	api.ReportServiceClient
	report    func(context.Context, *api.ReportWorkflowRequest) (*emptypb.Empty, error)
	scheduled func(context.Context, *api.ReportScheduledWorkflowRequest) (*emptypb.Empty, error)
}

func (c reportClient) ReportWorkflowV1(ctx context.Context, req *api.ReportWorkflowRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return c.report(ctx, req)
}
func (c reportClient) ReportScheduledWorkflowV1(ctx context.Context, req *api.ReportScheduledWorkflowRequest, _ ...grpc.CallOption) (*emptypb.Empty, error) {
	return c.scheduled(ctx, req)
}

type runClient struct {
	api.RunServiceClient
	report func(context.Context, *api.ReportRunMetricsRequest) (*api.ReportRunMetricsResponse, error)
}

func (c runClient) ReportRunMetricsV1(ctx context.Context, req *api.ReportRunMetricsRequest, _ ...grpc.CallOption) (*api.ReportRunMetricsResponse, error) {
	return c.report(ctx, req)
}

func selectedReport(t *testing.T, mutate func(*workflowReport)) *api.ReportWorkflowRequest {
	t.Helper()
	wf := workflowReport{Kind: "Workflow", APIVersion: "argoproj.io/v1alpha1", Metadata: metav1.ObjectMeta{
		Namespace: fixtureNamespace, Name: "selected-wf", UID: types.UID("workflow-uid"), Labels: map[string]string{"pipeline/runid": "run-id"},
	}}
	wf.Status.Phase = "Succeeded"
	if mutate != nil {
		mutate(&wf)
	}
	raw, err := json.Marshal(wf)
	if err != nil {
		t.Fatal(err)
	}
	return &api.ReportWorkflowRequest{Workflow: string(raw)}
}

func TestConfigRejectsUnscopedTargets(t *testing.T) {
	for _, tc := range []struct{ ns, raw string }{
		{"production", config}, {fixtureNamespace, "[]"}, {fixtureNamespace, `[{}]`},
		{fixtureNamespace, `[{"run_id":"../run","workflow_name":"selected-wf","workflow_uid":"uid"}]`},
		{fixtureNamespace, strings.TrimSuffix(config, "]") + "," + strings.TrimPrefix(config, "[")},
	} {
		if _, err := newProxy(tc.ns, tc.raw); err == nil {
			t.Fatalf("accepted unsafe target %q", tc.ns)
		}
	}
}

func TestTerminalCaptureDeleteAndExactForward(t *testing.T) {
	p, err := newProxy(fixtureNamespace, config)
	if err != nil {
		t.Fatal(err)
	}
	deleted := 0
	p.deleteWorkflow = func(ctx context.Context, selected target) error {
		deleted++
		if selected.RunID != "run-id" || selected.WorkflowUID != "workflow-uid" {
			t.Fatal("wrong selected identity")
		}
		if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > operationTimeout {
			t.Fatal("unbounded context")
		}
		return nil
	}
	req := selectedReport(t, nil)
	upstream := status.Error(codes.NotFound, "already deleted")
	p.reports = reportClient{report: func(ctx context.Context, received *api.ReportWorkflowRequest) (*emptypb.Empty, error) {
		if received != req || deleted != 1 {
			t.Fatal("request changed or forwarded before deletion")
		}
		md, _ := metadata.FromOutgoingContext(ctx)
		if md.Get("authorization")[0] != "Bearer test-token" || md.Get("x-test")[0] != "unchanged" {
			t.Fatal("metadata lost")
		}
		return nil, upstream
	}}
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer test-token", "x-test", "unchanged"))
	for i := 0; i < 2; i++ {
		if _, err := p.ReportWorkflowV1(ctx, req); err != upstream {
			t.Fatalf("upstream error changed: %v", err)
		}
	}
	if deleted != 1 || p.targets["run-id"].Attempts != 2 || p.targets["run-id"].UpstreamCode != "NotFound" {
		t.Fatal("incorrect deletion evidence")
	}
	recorder := httptest.NewRecorder()
	p.serveEvidence(recorder, httptest.NewRequest(http.MethodGet, "/apis/v2beta1/reporting-fixture-evidence", nil))
	if !strings.Contains(recorder.Body.String(), `"deleted":true`) || strings.Contains(recorder.Body.String(), "test-token") || strings.Contains(recorder.Body.String(), "already deleted") {
		t.Fatal("missing or unsanitized evidence")
	}
}

func TestUnselectedReportsNeverDelete(t *testing.T) {
	cases := []func(*workflowReport){
		func(w *workflowReport) { w.Status.Phase = "Running" },
		func(w *workflowReport) { w.Metadata.Namespace = "other" },
		func(w *workflowReport) { w.Metadata.Name = "other" },
		func(w *workflowReport) { w.Metadata.UID = "replacement" },
		func(w *workflowReport) { w.Metadata.Labels["pipeline/runid"] = "other" },
		func(w *workflowReport) { w.Kind = "Pod" },
		func(w *workflowReport) { w.APIVersion = "other/v1" },
	}
	for _, mutate := range cases {
		p, _ := newProxy(fixtureNamespace, config)
		p.deleteWorkflow = func(context.Context, target) error { t.Fatal("deleted an unselected Workflow"); return nil }
		req := selectedReport(t, mutate)
		calls := 0
		upstream := status.Error(codes.PermissionDenied, "denied")
		p.reports = reportClient{report: func(_ context.Context, got *api.ReportWorkflowRequest) (*emptypb.Empty, error) {
			calls++
			if got != req {
				t.Fatal("changed request")
			}
			return nil, upstream
		}}
		if _, err := p.ReportWorkflowV1(context.Background(), req); err != upstream || calls != 1 {
			t.Fatal("failed to pass through error")
		}
	}
}

func TestDeleteFailureDoesNotForwardOrClaimSuccess(t *testing.T) {
	p, _ := newProxy(fixtureNamespace, config)
	p.deleteWorkflow = func(context.Context, target) error { return errors.New("sensitive infrastructure detail") }
	p.reports = reportClient{report: func(context.Context, *api.ReportWorkflowRequest) (*emptypb.Empty, error) {
		t.Fatal("forwarded without establishing deletion")
		return nil, nil
	}}
	_, err := p.ReportWorkflowV1(context.Background(), selectedReport(t, nil))
	if status.Code(err) != codes.Unavailable || strings.Contains(err.Error(), "sensitive") {
		t.Fatal("unsafe fixture error")
	}
	if p.targets["run-id"].Deleted || !p.targets["run-id"].DeletionFailed {
		t.Fatal("incorrect deletion evidence")
	}
}

func TestDeleteUsesUIDPreconditionAndWaitsForAbsence(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "argoproj.io/v1alpha1", "kind": "Workflow", "metadata": map[string]interface{}{
			"name": "selected-wf", "namespace": fixtureNamespace, "uid": "workflow-uid",
		},
	}}
	client := fake.NewSimpleDynamicClient(runtime.NewScheme(), obj)
	checked := false
	client.PrependReactor("delete", "workflows", func(action ktesting.Action) (bool, runtime.Object, error) {
		options := action.(ktesting.DeleteAction).GetDeleteOptions()
		if action.GetNamespace() != fixtureNamespace || options.Preconditions == nil || options.Preconditions.UID == nil || *options.Preconditions.UID != "workflow-uid" {
			t.Fatal("delete lacked immutable namespace/UID scope")
		}
		checked = true
		return false, nil, nil
	})
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	if err := deleteSelected(ctx, client, target{RunID: "run-id", WorkflowName: "selected-wf", WorkflowUID: "workflow-uid"}); err != nil {
		t.Fatal(err)
	}
	if !checked || len(client.Actions()) != 2 || client.Actions()[1].GetVerb() != "get" {
		t.Fatal("did not confirm deletion")
	}
	// A Workflow absent before interception does not prove the intended race.
	if err := deleteSelected(ctx, client, target{WorkflowName: "selected-wf", WorkflowUID: "workflow-uid"}); err == nil {
		t.Fatal("accepted preexisting absence")
	}
}

func TestScheduledAndMetricsForwardAuthentication(t *testing.T) {
	p, _ := newProxy(fixtureNamespace, config)
	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs("authorization", "Bearer unchanged"))
	check := func(ctx context.Context) {
		md, _ := metadata.FromOutgoingContext(ctx)
		if md.Get("authorization")[0] != "Bearer unchanged" {
			t.Fatal("changed caller")
		}
	}
	scheduled := &api.ReportScheduledWorkflowRequest{ScheduledWorkflow: "exact"}
	metrics := &api.ReportRunMetricsRequest{RunId: "run-id"}
	upstream := status.Error(codes.Unauthenticated, "rejected")
	p.reports = reportClient{scheduled: func(ctx context.Context, req *api.ReportScheduledWorkflowRequest) (*emptypb.Empty, error) {
		check(ctx)
		if req != scheduled {
			t.Fatal("changed scheduled report")
		}
		return nil, upstream
	}}
	p.runs = runClient{report: func(ctx context.Context, req *api.ReportRunMetricsRequest) (*api.ReportRunMetricsResponse, error) {
		check(ctx)
		if req != metrics {
			t.Fatal("changed metrics")
		}
		return nil, upstream
	}}
	if _, err := p.ReportScheduledWorkflowV1(ctx, scheduled); err != upstream {
		t.Fatal("changed scheduled error")
	}
	if _, err := p.ReportRunMetricsV1(ctx, metrics); err != upstream {
		t.Fatal("changed metrics error")
	}
}
