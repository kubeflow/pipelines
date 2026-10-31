// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package executorplugin serves the authenticated Argo executor-plugin driver API.
package executorplugin

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/golang/glog"
	"github.com/kubeflow/pipelines/backend/src/driver/driverapi"
	"github.com/kubeflow/pipelines/backend/src/v2/driver"
	"google.golang.org/protobuf/encoding/protojson"
)

// DriverExecutor executes one validated driver request, including its resource lifecycle.
// An execution returned with an error may still contain outputs needed by Argo.
type DriverExecutor func(context.Context, driverapi.DriverPluginArgs) (*driver.Execution, error)

const invocationTimeout = 30 * time.Minute

// Handler serves authenticated template.execute requests. Driver work outlives
// the HTTP request, so Drain lets a shutting-down server wait for it.
type Handler struct {
	authenticated http.Handler
	service       *driverService
}

// driverService runs at most one driver execution per taskset attempt and
// replays its result to Argo's repeated polls.
type driverService struct {
	execute DriverExecutor
	resolve attemptResolver
	ctx     context.Context
	mu      sync.Mutex
	calls   map[string]*driverCall
}

type driverCall struct {
	done     chan struct{}
	response driverapi.DriverResponse
}

// NewHandler loads the Argo bearer token and binds driver work to ctx, the
// service lifetime. Results are retained for this agent process; restarting it
// loses replay protection. A nil executor uses the in-cluster KFP driver runtime.
func NewHandler(ctx context.Context, tokenPath string, execute DriverExecutor) (*Handler, error) {
	if ctx == nil {
		return nil, fmt.Errorf("driver service context is nil; provide a context tied to the service lifetime")
	}
	agent := cachedAgent(inClusterAgent)
	if execute == nil {
		execute = newDriverRunner(agent).drive
	}
	return newHandlerWithResolver(ctx, tokenPath, execute, tasksetAttemptResolver(agent, watchedTaskset(ctx)))
}

func newHandlerWithResolver(ctx context.Context, tokenPath string, execute DriverExecutor, resolve attemptResolver) (*Handler, error) {
	service := &driverService{execute: execute, resolve: resolve, ctx: ctx, calls: make(map[string]*driverCall)}
	authenticated, err := authenticatedPluginHandler(tokenPath, http.HandlerFunc(service.executePlugin))
	if err != nil {
		return nil, err
	}
	return &Handler{authenticated: authenticated, service: service}, nil
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.authenticated.ServeHTTP(w, r)
}

// Drain waits for detached driver executions to finish or ctx to expire.
func (h *Handler) Drain(ctx context.Context) error {
	h.service.mu.Lock()
	done := make([]<-chan struct{}, 0, len(h.service.calls))
	for _, call := range h.service.calls {
		done = append(done, call.done)
	}
	h.service.mu.Unlock()
	for _, finished := range done {
		select {
		case <-finished:
		case <-ctx.Done():
			return fmt.Errorf("driver execution cleanup did not finish before shutdown deadline: %w", ctx.Err())
		}
	}
	return nil
}

func (s *driverService) executePlugin(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if err := r.Body.Close(); err != nil {
			glog.Errorf("Error closing request body: %v", err)
		}
	}()

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	glog.Info("Received request to execute driver plugin")
	args, err := parseDriverRequestArgs(r)
	if err != nil {
		glog.Errorf("Failed to parse driver request args: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if args == nil {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, "{}")
		return
	}
	// Resolve on every poll: an intentional Argo retry has a new taskset node ID.
	lookupCtx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	attempt, err := s.resolve(lookupCtx, *args)
	cancel()
	if err != nil || s.ctx.Err() != nil {
		glog.Warningf("Cannot resolve active driver attempt: %v (service: %v)", err, s.ctx.Err())
		http.Error(w, "cannot resolve active driver attempt; retry after taskset reconciliation", http.StatusServiceUnavailable)
		return
	}
	key := fmt.Sprintf("%s/%x", attempt, argsDigest(*args))
	s.mu.Lock()
	if s.ctx.Err() != nil {
		s.mu.Unlock()
		http.Error(w, "driver service is shutting down", http.StatusServiceUnavailable)
		return
	}
	call := s.calls[key]
	if call == nil {
		call = &driverCall{done: make(chan struct{})}
		s.calls[key] = call
		go s.run(call, *args)
	}
	s.mu.Unlock()
	timer := time.NewTimer(50 * time.Millisecond)
	defer timer.Stop()
	select {
	case <-call.done:
		writeJSONResponse(w, call.response)
	case <-timer.C:
		writeJSONResponse(w, driverapi.DriverResponse{Node: driverapi.Node{Phase: "Running"}, Requeue: "1s"})
	case <-r.Context().Done():
	}
}

func argsDigest(args driverapi.DriverPluginArgs) [32]byte {
	encoded, _ := json.Marshal(args) // validated DTO contains no unsupported JSON types
	return sha256.Sum256(encoded)
}

func (s *driverService) run(call *driverCall, args driverapi.DriverPluginArgs) {
	ctx, cancel := context.WithTimeout(s.ctx, invocationTimeout)
	defer cancel()
	defer close(call.done)
	defer func() {
		if recovered := recover(); recovered != nil {
			call.response = driverapi.DriverResponse{Node: driverapi.Node{Phase: "Failed", Message: fmt.Sprintf("unable to drive execution: panic: %v", recovered)}}
		}
	}()
	execution, err := s.execute(ctx, args)
	if err == nil {
		err = ctx.Err()
	}
	if err == nil && execution == nil {
		// Argo would otherwise run the task with empty outputs and no pod patch.
		err = errors.New("driver returned no execution result; report this driver bug with the run ID")
	}
	outputs := extractOutputParameters(execution, args.DriverType)
	if err == nil && execution.ExecutorInput != nil {
		if _, marshalErr := protojson.Marshal(execution.ExecutorInput); marshalErr != nil {
			err = fmt.Errorf("failed to marshal ExecutorInput to JSON: %w", marshalErr)
		}
	}
	call.response = driverapi.DriverResponse{Node: driverapi.Node{Phase: "Succeeded", Outputs: driverapi.Outputs{Parameters: outputs}}}
	if err != nil {
		call.response.Node.Phase = "Failed"
		call.response.Node.Message = fmt.Sprintf("unable to drive execution: %v", err)
	}
}

// rawDriverRequest is the part of Argo's ExecuteTemplateArgs the plugin reads.
type rawDriverRequest struct {
	Template *struct {
		Plugin json.RawMessage `json:"plugin"`
	} `json:"template"`
}

// parseDriverRequestArgs returns nil args for templates owned by another plugin.
func parseDriverRequestArgs(r *http.Request) (*driverapi.DriverPluginArgs, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read driver request body; resend the request: %w", err)
	}
	var request rawDriverRequest
	if err := json.Unmarshal(body, &request); err != nil {
		return nil, fmt.Errorf("driver request body is not valid JSON; send Argo ExecuteTemplateArgs: %w", err)
	}
	if request.Template == nil {
		return nil, fmt.Errorf("driver request has no template; send Argo ExecuteTemplateArgs with template.plugin")
	}
	if len(request.Template.Plugin) == 0 || string(request.Template.Plugin) == "null" {
		return nil, fmt.Errorf("driver request template has no plugin; send Argo ExecuteTemplateArgs with template.plugin")
	}
	args, owned, err := driverapi.ParseDriverPluginArgs(request.Template.Plugin)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, nil
	}
	return args, nil
}

func extractOutputParameters(execution *driver.Execution, driverType driverapi.DriverType) []driverapi.Parameter {
	if execution == nil {
		return []driverapi.Parameter{}
	}
	var outputs []driverapi.Parameter
	if execution.TaskID != "" {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "task-id",
			Value: execution.TaskID,
		})
	}
	if execution.IterationCount != nil {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "iteration-count",
			Value: fmt.Sprint(*execution.IterationCount),
		})
	} else if driverType.IsDAG() {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "iteration-count",
			Value: "0",
		})
	}
	if execution.Cached != nil {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "cached-decision",
			Value: strconv.FormatBool(*execution.Cached),
		})
	}
	condition := "nil" // nil is a valid value for Condition
	if execution.Condition != nil {
		condition = strconv.FormatBool(*execution.Condition)
	}
	outputs = append(outputs, driverapi.Parameter{Name: "condition", Value: condition})
	outputs = append(outputs, driverapi.Parameter{Name: "pod-spec-patch", Value: execution.PodSpecPatch})
	return outputs
}

func writeJSONResponse(w http.ResponseWriter, payload driverapi.DriverResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
