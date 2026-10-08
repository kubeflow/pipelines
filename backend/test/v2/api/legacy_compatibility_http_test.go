// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	api "github.com/kubeflow/pipelines/backend/api/v2/go_client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompatibilityHTTPClientPreservesLiteralPrefixes(t *testing.T) {
	paths := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths <- r.URL.EscapedPath()
		assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "namespace=team%2Fone", r.URL.RawQuery)
		assert.Equal(t, http.MethodPost, r.Method)
		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"display_name":"wire-test"}`, string(body))
		_, _ = w.Write([]byte(`{"experiment_id":"id","display_name":"wire-test"}`))
	}))
	t.Cleanup(server.Close)
	client := newCompatibilityHTTPClient(server.URL+"/pipeline/", "test-token", nil)
	t.Cleanup(client.client.CloseIdleConnections)
	for _, version := range []string{"v2beta1", "v2"} {
		result := new(api.Experiment)
		err := client.message(t.Context(), http.MethodPost, version, "/experiments/a%2Fb?namespace=team%2Fone", &api.Experiment{DisplayName: "wire-test"}, result)
		require.NoError(t, err)
		require.Equal(t, "id", result.ExperimentId)
		require.Equal(t, "/pipeline/apis/"+version+"/experiments/a%2Fb", <-paths)
	}
}

func TestCompatibilityHTTPClientRejectsRedirectsAndMissingAliases(t *testing.T) {
	for _, status := range []int{http.StatusTemporaryRedirect, http.StatusNotFound} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var canonicalCalls atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/apis/v2beta1/healthz" {
					w.Header().Set("Location", "/apis/v2/healthz")
					w.WriteHeader(status)
					return
				}
				canonicalCalls.Add(1)
				_, _ = w.Write([]byte(`{}`))
			}))
			t.Cleanup(server.Close)
			client := newCompatibilityHTTPClient(server.URL, "", nil)
			t.Cleanup(client.client.CloseIdleConnections)
			_, err := client.request(t.Context(), http.MethodGet, "v2beta1", "/healthz", nil, "application/json", http.StatusOK)
			require.ErrorContains(t, err, "/apis/v2beta1/healthz")
			require.Zero(t, canonicalCalls.Load(), "must not hide a missing legacy route by following a redirect or falling back")
		})
	}
}

func TestCompatibilityHTTPClientPreservesCancellation(t *testing.T) {
	client := newCompatibilityHTTPClient("http://127.0.0.1:1", "", nil)
	t.Cleanup(client.client.CloseIdleConnections)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := client.request(ctx, http.MethodGet, "v2beta1", "/healthz", nil, "application/json", http.StatusOK)
	require.ErrorIs(t, err, context.Canceled)
}
