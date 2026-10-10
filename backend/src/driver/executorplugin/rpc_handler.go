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

const (
	unsetProxyArgValue = "unset"
	rootDAG            = "ROOT_DAG"
	dagDriver          = "DAG"
	containerDriver    = "CONTAINER"
)

// DriverExecutor executes one validated driver request, including its resource lifecycle.
// An execution returned with an error may still contain outputs needed by Argo.
type DriverExecutor func(context.Context, driverapi.DriverPluginArgs) (*driver.Execution, error)

type handler struct {
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

const invocationTimeout = 30 * time.Minute

// NewHandler loads the Argo bearer token and constructs the template.execute handler.
// A nil executor uses the in-cluster KFP driver runtime.
func NewHandler(tokenPath string, execute DriverExecutor) (http.Handler, error) {
	return NewHandlerWithContext(context.Background(), tokenPath, execute)
}

// NewHandlerWithContext binds driver work to the service lifetime, not an RPC timeout.
// Results are retained for this agent process; restarting it loses replay protection.
func NewHandlerWithContext(ctx context.Context, tokenPath string, execute DriverExecutor) (http.Handler, error) {
	if ctx == nil {
		return nil, fmt.Errorf("driver service context is nil; provide a context tied to the service lifetime")
	}
	return newHandlerWithResolver(ctx, tokenPath, execute, inClusterAttemptResolver())
}

func newHandlerWithResolver(ctx context.Context, tokenPath string, execute DriverExecutor, resolve attemptResolver) (http.Handler, error) {
	if execute == nil {
		execute = newDriverRunner().drive
	}
	h := &handler{execute: execute, resolve: resolve, ctx: ctx, calls: make(map[string]*driverCall)}
	authenticated, err := authenticatedPluginHandler(tokenPath, http.HandlerFunc(h.executePlugin))
	if err != nil {
		return nil, err
	}
	return &drainingHandler{Handler: authenticated, execution: h}, nil
}

func (h *handler) executePlugin(w http.ResponseWriter, r *http.Request) {
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
	attempt, err := h.resolve(lookupCtx, *args)
	cancel()
	if err != nil || h.ctx.Err() != nil {
		glog.Warningf("Cannot resolve active driver attempt: %v (service: %v)", err, h.ctx.Err())
		http.Error(w, "cannot resolve active driver attempt; retry after taskset reconciliation", http.StatusServiceUnavailable)
		return
	}
	key := fmt.Sprintf("%s/%x", attempt, argsDigest(*args))
	h.mu.Lock()
	if h.ctx.Err() != nil {
		h.mu.Unlock()
		http.Error(w, "driver service is shutting down", http.StatusServiceUnavailable)
		return
	}
	call := h.calls[key]
	if call == nil {
		call = &driverCall{done: make(chan struct{})}
		h.calls[key] = call
		go h.run(call, *args)
	}
	h.mu.Unlock()
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

func (h *handler) run(call *driverCall, args driverapi.DriverPluginArgs) {
	ctx, cancel := context.WithTimeout(h.ctx, invocationTimeout)
	defer cancel()
	defer close(call.done)
	defer func() {
		if recovered := recover(); recovered != nil {
			call.response = driverapi.DriverResponse{Node: driverapi.Node{Phase: "Failed", Message: fmt.Sprintf("unable to drive execution: panic: %v", recovered)}}
		}
	}()
	execution, err := h.execute(ctx, args)
	if err == nil {
		err = ctx.Err()
	}
	outputs := extractOutputParameters(execution, args.Type)
	if err == nil && execution != nil && execution.ExecutorInput != nil {
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

func parseDriverRequestArgs(r *http.Request) (*driverapi.DriverPluginArgs, error) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read driver request body: %w", err)
	}
	var body rawDriverRequest
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		return nil, fmt.Errorf("failed to parse driver request body: %w", err)
	}
	switch {
	case body.Template == nil:
		return nil, fmt.Errorf("driver request body.Template is empty")
	case body.Template.Plugin == nil:
		return nil, fmt.Errorf("driver request body.Template.Plugin is empty")
	}
	config, owned := body.Template.Plugin["driver-plugin"]
	if !owned {
		return nil, nil
	}
	var container *rawDriverPluginContainer
	if err := json.Unmarshal(config, &container); err != nil || container == nil {
		return nil, fmt.Errorf("driver request body.Template.Plugin.DriverPlugin must be an object")
	}
	if len(container.Args) == 0 || string(container.Args) == "null" {
		return nil, fmt.Errorf("driver request body.Template.Plugin.Args is empty")
	}
	var args driverapi.DriverPluginArgs
	if err := json.Unmarshal(container.Args, &args); err != nil {
		return nil, fmt.Errorf("failed to parse driver request args: %w", err)
	}
	var argFields map[string]json.RawMessage
	if err := json.Unmarshal(container.Args, &argFields); err != nil {
		return nil, fmt.Errorf("failed to parse driver request args as object: %w", err)
	}
	if err := validate(args, argFields); err != nil {
		return nil, err
	}
	return &args, nil
}

type rawDriverRequest struct {
	Template *rawDriverTemplate `json:"template"`
}

type rawDriverTemplate struct {
	Plugin map[string]json.RawMessage `json:"plugin"`
}

type rawDriverPluginContainer struct {
	Args json.RawMessage `json:"args"`
}

var commonRequiredDriverArgFields = []string{
	"type", "pipeline_name", "run_id", "run_name", "run_display_name",
	"parent_task_id", "task_name", "namespace", "iteration_index",
	"ml_pipeline_server_address", "ml_pipeline_server_port", "log_level", "publish_logs",
	"cache_disabled", "ml_pipeline_tls_enabled", "http_proxy", "https_proxy", "no_proxy",
}

func requiredDriverArgFields(driverType string) ([]string, error) {
	required := append([]string{}, commonRequiredDriverArgFields...)
	switch driverType {
	case rootDAG:
		required = append(required, "runtime_config")
	case dagDriver:
	case containerDriver:
		required = append(required, "kubernetes_config")
	default:
		return nil, fmt.Errorf("unknown driver type %q, must be one of %s, %s, %s", driverType, rootDAG, dagDriver, containerDriver)
	}
	return required, nil
}

func validate(args driverapi.DriverPluginArgs, argFields map[string]json.RawMessage) error {
	switch {
	case args.Type == "":
		return fmt.Errorf("argument type must be specified")
	case args.HTTPProxy == unsetProxyArgValue:
		return fmt.Errorf("argument http_proxy is required but can be an empty value")
	case args.HTTPSProxy == unsetProxyArgValue:
		return fmt.Errorf("argument https_proxy is required but can be an empty value")
	case args.NoProxy == unsetProxyArgValue:
		return fmt.Errorf("argument no_proxy is required but can be an empty value")
	}
	required, err := requiredDriverArgFields(args.Type)
	if err != nil {
		return err
	}
	for _, name := range required {
		if _, ok := argFields[name]; !ok {
			return fmt.Errorf("--%s is required for %s but was not provided", name, args.Type)
		}
	}
	_, err = parseLogLevel(args.LogLevel)
	return err
}

func extractOutputParameters(execution *driver.Execution, driverType string) []driverapi.Parameter {
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
	} else if driverType == rootDAG || driverType == dagDriver {
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
	if execution.Condition != nil {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "condition",
			Value: strconv.FormatBool(*execution.Condition),
		})
	} else if driverType == dagDriver || driverType == rootDAG || driverType == containerDriver {
		// nil is a valid value for Condition
		outputs = append(outputs, driverapi.Parameter{
			Name:  "condition",
			Value: "nil",
		})
	}
	if execution.PodSpecPatch != "" {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "pod-spec-patch",
			Value: execution.PodSpecPatch,
		})
	} else {
		outputs = append(outputs, driverapi.Parameter{
			Name:  "pod-spec-patch",
			Value: "",
		})
	}
	return outputs
}

func writeJSONResponse(w http.ResponseWriter, payload driverapi.DriverResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, "failed to encode response", http.StatusInternalServerError)
	}
}
