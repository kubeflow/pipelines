// Copyright 2026 The Kubeflow Authors
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

package executorplugin

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"
)

type drainingHandler struct {
	http.Handler
	execution *handler
}

func (h *drainingHandler) drain(ctx context.Context) error {
	h.execution.mu.Lock()
	done := make([]<-chan struct{}, 0, len(h.execution.calls))
	for _, call := range h.execution.calls {
		done = append(done, call.done)
	}
	h.execution.mu.Unlock()
	for _, finished := range done {
		select {
		case <-finished:
		case <-ctx.Done():
			return fmt.Errorf("driver execution cleanup did not finish before shutdown deadline: %w", ctx.Err())
		}
	}
	return nil
}

// Serve runs the authenticated driver endpoint until ctx is canceled or serving
// fails, then waits up to ten seconds for HTTP and detached execution cleanup.
func Serve(ctx context.Context, address, tokenPath string) error {
	if ctx == nil {
		return fmt.Errorf("driver service context is nil; provide a service lifetime context")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	handler, err := NewHandlerWithContext(ctx, tokenPath, nil)
	if err != nil {
		return err
	}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("cannot listen on %q; choose an available driver port: %w", address, err)
	}
	return serveHTTP(ctx, cancel, listener, handler, 10*time.Second)
}

func serveHTTP(ctx context.Context, cancel context.CancelFunc, listener net.Listener, handler http.Handler, grace time.Duration) error {
	mux := http.NewServeMux()
	mux.Handle("/api/v1/template.execute", handler)
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	var serveErr error
	serveFinished := false
	select {
	case <-ctx.Done():
	case serveErr = <-served:
		serveFinished = true
	}
	cancel()
	shutdownCtx, finish := context.WithTimeout(context.Background(), grace)
	defer finish()
	shutdownErr := server.Shutdown(shutdownCtx)
	if shutdownErr != nil {
		_ = server.Close()
	}
	if !serveFinished {
		serveErr = <-served
	}
	var drainErr error
	if drainer, ok := handler.(interface{ drain(context.Context) error }); ok {
		drainErr = drainer.drain(shutdownCtx)
	}
	if errors.Is(serveErr, http.ErrServerClosed) {
		serveErr = nil
	}
	return errors.Join(serveErr, shutdownErr, drainErr)
}
