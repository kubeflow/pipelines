// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy at https://www.apache.org/licenses/LICENSE-2.0
// Distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND.

// Command reporting-proxy injects a deletion race into disposable upgrade tests.
// It must never be deployed to a production persistence-agent path.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

const fixtureNamespace = "kfp-readiness-test"
const operationTimeout = 30 * time.Second

var workflows = schema.GroupVersionResource{Group: "argoproj.io", Version: "v1alpha1", Resource: "workflows"}

type target struct {
	RunID        string `json:"run_id"`
	WorkflowName string `json:"workflow_name"`
	WorkflowUID  string `json:"workflow_uid"`
}

type evidence struct {
	target
	Deleted        bool   `json:"deleted"`
	UpstreamCode   string `json:"upstream_code"`
	Attempts       int    `json:"attempts"`
	DeletionFailed bool   `json:"deletion_failed"`
}

type workflowReport struct {
	Kind       string            `json:"kind"`
	APIVersion string            `json:"apiVersion"`
	Metadata   metav1.ObjectMeta `json:"metadata"`
	Status     struct {
		Phase string `json:"phase"`
	} `json:"status"`
}

type proxy struct {
	api.UnimplementedReportServiceServer
	api.UnimplementedRunServiceServer
	reports        api.ReportServiceClient
	runs           api.RunServiceClient
	deleteWorkflow func(context.Context, target) error
	mu             sync.Mutex
	targets        map[string]*evidence
	order          []string
}

func newProxy(namespace, raw string) (*proxy, error) {
	if namespace != fixtureNamespace || len(raw) > 16384 {
		return nil, errors.New("fixture namespace or target limit invalid")
	}
	var targets []target
	if err := json.Unmarshal([]byte(raw), &targets); err != nil || len(targets) == 0 || len(targets) > 10 {
		return nil, errors.New("fixture requires one to ten explicit targets")
	}
	p := &proxy{targets: make(map[string]*evidence)}
	names := make(map[string]bool)
	for _, t := range targets {
		if len(t.RunID) == 0 || len(t.RunID) > 253 || len(validation.IsDNS1123Subdomain(t.RunID)) != 0 ||
			len(t.WorkflowUID) == 0 || len(t.WorkflowUID) > 253 || len(validation.IsDNS1123Subdomain(t.WorkflowUID)) != 0 ||
			len(validation.IsDNS1123Subdomain(t.WorkflowName)) != 0 || names[t.WorkflowName] || p.targets[t.RunID] != nil {
			return nil, errors.New("fixture target identity invalid or duplicated")
		}
		names[t.WorkflowName] = true
		p.targets[t.RunID] = &evidence{target: t}
		p.order = append(p.order, t.RunID)
	}
	return p, nil
}

func outgoing(ctx context.Context) (context.Context, context.CancelFunc) {
	md, _ := metadata.FromIncomingContext(ctx)
	return context.WithTimeout(metadata.NewOutgoingContext(ctx, md.Copy()), operationTimeout)
}

func (p *proxy) ReportWorkflowV1(ctx context.Context, request *api.ReportWorkflowRequest) (*emptypb.Empty, error) {
	ctx, cancel := outgoing(ctx)
	defer cancel()
	var report workflowReport
	// Unselected and malformed reports are passed unchanged to the real API.
	if json.Unmarshal([]byte(request.GetWorkflow()), &report) != nil ||
		report.Kind != "Workflow" || report.APIVersion != "argoproj.io/v1alpha1" ||
		report.Metadata.Namespace != fixtureNamespace ||
		(report.Status.Phase != "Succeeded" && report.Status.Phase != "Failed" && report.Status.Phase != "Error") {
		return p.reports.ReportWorkflowV1(ctx, request)
	}
	p.mu.Lock()
	entry := p.targets[report.Metadata.Labels["pipeline/runid"]]
	if entry == nil || entry.WorkflowName != report.Metadata.Name || entry.WorkflowUID != string(report.Metadata.UID) {
		p.mu.Unlock()
		return p.reports.ReportWorkflowV1(ctx, request)
	}
	// Keep deletion and forwarding ordered across duplicate worker reports.
	defer p.mu.Unlock()
	entry.Attempts++
	if !entry.Deleted {
		if err := p.deleteWorkflow(ctx, entry.target); err != nil {
			entry.DeletionFailed = true
			return nil, status.Error(codes.Unavailable, "fixture could not delete the selected Workflow")
		}
		entry.Deleted = true
	}
	response, err := p.reports.ReportWorkflowV1(ctx, request)
	entry.UpstreamCode = status.Code(err).String()
	return response, err
}

func (p *proxy) ReportScheduledWorkflowV1(ctx context.Context, request *api.ReportScheduledWorkflowRequest) (*emptypb.Empty, error) {
	ctx, cancel := outgoing(ctx)
	defer cancel()
	return p.reports.ReportScheduledWorkflowV1(ctx, request)
}

func (p *proxy) ReportRunMetricsV1(ctx context.Context, request *api.ReportRunMetricsRequest) (*api.ReportRunMetricsResponse, error) {
	ctx, cancel := outgoing(ctx)
	defer cancel()
	return p.runs.ReportRunMetricsV1(ctx, request)
}

func deleteSelected(ctx context.Context, client dynamic.Interface, selected target) error {
	resource := client.Resource(workflows).Namespace(fixtureNamespace)
	uid := types.UID(selected.WorkflowUID)
	zero := int64(0)
	background := metav1.DeletePropagationBackground
	// NotFound on the first delete is a failed experiment: this proxy did not
	// establish that deletion happened after the worker captured the report.
	if err := resource.Delete(ctx, selected.WorkflowName, metav1.DeleteOptions{
		Preconditions: &metav1.Preconditions{UID: &uid}, GracePeriodSeconds: &zero,
		PropagationPolicy: &background,
	}); err != nil {
		return err
	}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		current, err := resource.Get(ctx, selected.WorkflowName, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if current.GetUID() != uid {
			return errors.New("selected Workflow was replaced")
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (p *proxy) serveEvidence(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	entries := make([]evidence, 0, len(p.order))
	for _, id := range p.order {
		entries = append(entries, *p.targets[id])
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(struct {
		Namespace string     `json:"namespace"`
		Runs      []evidence `json:"runs"`
	}{fixtureNamespace, entries})
}

func health(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://ml-pipeline:8888/healthz", nil)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	response, err := client.Do(request)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	defer response.Body.Close()
	// Initialization reads the API version from the health response.
	w.WriteHeader(response.StatusCode)
	_, _ = io.Copy(w, io.LimitReader(response.Body, 65536))
}

func run() error {
	p, err := newProxy(os.Getenv("FIXTURE_NAMESPACE"), os.Getenv("FIXTURE_TARGETS_JSON"))
	if err != nil {
		return err
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		return errors.New("fixture requires in-cluster credentials")
	}
	config.Timeout = 10 * time.Second
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return errors.New("fixture Kubernetes client unavailable")
	}
	p.deleteWorkflow = func(ctx context.Context, selected target) error { return deleteSelected(ctx, client, selected) }
	connection, err := grpc.NewClient("ml-pipeline:8887", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return errors.New("fixture upstream unavailable")
	}
	defer connection.Close()
	p.reports = api.NewReportServiceClient(connection)
	p.runs = api.NewRunServiceClient(connection)
	listener, err := net.Listen("tcp", ":8887")
	if err != nil {
		return errors.New("fixture grpc listener unavailable")
	}
	server := grpc.NewServer(grpc.MaxRecvMsgSize(16 * 1024 * 1024))
	api.RegisterReportServiceServer(server, p)
	api.RegisterRunServiceServer(server, p)
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", health)
	mux.HandleFunc("/apis/v2beta1/reporting-fixture-evidence", p.serveEvidence)
	httpServer := &http.Server{Addr: ":8888", Handler: mux, ReadHeaderTimeout: 5 * time.Second, WriteTimeout: 40 * time.Second, IdleTimeout: 30 * time.Second}
	failures := make(chan error, 2)
	go func() { failures <- server.Serve(listener) }()
	go func() { failures <- httpServer.ListenAndServe() }()
	<-failures
	server.Stop()
	_ = httpServer.Close()
	return errors.New("fixture listener stopped")
}

func main() {
	if err := run(); err != nil {
		// Only fixed diagnostic text reaches logs. Never print RPC requests/tokens.
		log.Print(err.Error())
		os.Exit(1)
	}
}
