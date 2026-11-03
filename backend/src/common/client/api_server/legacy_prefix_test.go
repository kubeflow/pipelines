// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

//nolint:staticcheck // Tests belong to the existing api_server package.
package api_server

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/kubeflow/pipelines/backend/test/config"
	"github.com/stretchr/testify/require"
)

type inspectingTransport struct {
	inspect func(*http.Request)
}

func (t *inspectingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	t.inspect(request)
	return &http.Response{StatusCode: http.StatusAccepted, Body: io.NopCloser(strings.NewReader("response"))}, nil
}

func TestLegacyPrefixTransportPreservesRequest(t *testing.T) {
	previous := *config.UseLegacyAPIPrefix
	t.Cleanup(func() { *config.UseLegacyAPIPrefix = previous })
	*config.UseLegacyAPIPrefix = true
	for _, tc := range []struct{ name, base, path, want string }{
		{"health", "", "/apis/v2/healthz", "/apis/v2beta1/healthz"},
		{"upload", "", "/apis/v2/pipelines/upload", "/apis/v2beta1/pipelines/upload"},
		{"escaped ID", "", "/apis/v2/runs/a%2Fb", "/apis/v2beta1/runs/a%2Fb"},
		{"lookalike", "", "/apis/v20/runs", "/apis/v20/runs"},
		{"already legacy", "", "/apis/v2beta1/runs", "/apis/v2beta1/runs"},
		{"unrelated", "", "/artifacts/apis/v2/file", "/artifacts/apis/v2/file"},
		{"Kubernetes proxy", "/api/v1/namespaces/kubeflow/services/ml-pipeline:8888/proxy/", "/apis/v2/runs", "/apis/v2beta1/runs"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			basePath := strings.TrimRight(tc.base, "/")
			request, err := http.NewRequest(http.MethodPost, "https://server"+basePath+tc.path+"?filter=a%2Bb&token=v2", strings.NewReader("unchanged body"))
			require.NoError(t, err)
			ctx, cancel := context.WithCancel(request.Context())
			cancel()
			request = request.WithContext(ctx)
			request.Header.Set("Authorization", "Bearer credential")
			originalURL := request.URL.String()
			transport := &inspectingTransport{inspect: func(forwarded *http.Request) {
				require.Equal(t, basePath+tc.want, forwarded.URL.EscapedPath())
				require.Equal(t, request.URL.RawQuery, forwarded.URL.RawQuery)
				require.Equal(t, request.Header, forwarded.Header)
				require.Equal(t, request.Method, forwarded.Method)
				require.True(t, request.Body == forwarded.Body)
				require.ErrorIs(t, forwarded.Context().Err(), context.Canceled)
			}}
			client := &http.Client{Transport: transport, Timeout: time.Minute}
			legacy := withLegacyAPIPrefix(client, tc.base)
			require.NotSame(t, client, legacy)
			require.Equal(t, client.Timeout, legacy.Timeout)
			response, err := legacy.Transport.RoundTrip(request)
			require.NoError(t, err)
			require.NoError(t, response.Body.Close())
			require.Equal(t, http.StatusAccepted, response.StatusCode)
			require.Same(t, transport, client.Transport)
			require.Equal(t, originalURL, request.URL.String())
		})
	}
}

func TestCanonicalPrefixDoesNotWrapClient(t *testing.T) {
	previous := *config.UseLegacyAPIPrefix
	t.Cleanup(func() { *config.UseLegacyAPIPrefix = previous })
	*config.UseLegacyAPIPrefix = false
	require.Same(t, http.DefaultClient, withLegacyAPIPrefix(http.DefaultClient, ""))
	*config.UseLegacyAPIPrefix = true
	client := withLegacyAPIPrefix(http.DefaultClient, "")
	require.NotSame(t, http.DefaultClient, client)
	require.NotNil(t, client.Transport)
}
