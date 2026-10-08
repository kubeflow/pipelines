// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

// Package api_server constructs HTTP transports for API clients and integration tests.
//
//nolint:staticcheck // Preserve the existing package name for importing clients.
package api_server

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/kubeflow/pipelines/backend/test/config"
)

// withLegacyAPIPrefix opts test clients into the old server's wire routes.
// It is explicit rather than a 404 fallback, and never changes shared clients.
func withLegacyAPIPrefix(client *http.Client, basePath string) *http.Client {
	if !*config.UseLegacyAPIPrefix {
		return client
	}
	copy := *client
	base := client.Transport
	if base == nil {
		base = http.DefaultTransport
	}
	copy.Transport = legacyPrefixTransport{
		base:   base,
		prefix: strings.TrimRight(basePath, "/") + "/apis/v2",
	}
	return &copy
}

type legacyPrefixTransport struct {
	base   http.RoundTripper
	prefix string
}

func (t legacyPrefixTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	if request.URL.Path != t.prefix && !strings.HasPrefix(request.URL.Path, t.prefix+"/") {
		return t.base.RoundTrip(request)
	}
	forwarded := request.Clone(request.Context())
	forwarded.URL.Path = t.prefix + "beta1" + strings.TrimPrefix(request.URL.Path, t.prefix)
	if request.URL.RawPath != "" {
		escapedPrefix := (&url.URL{Path: t.prefix}).EscapedPath()
		if strings.HasPrefix(request.URL.RawPath, escapedPrefix) {
			forwarded.URL.RawPath = escapedPrefix + "beta1" + strings.TrimPrefix(request.URL.RawPath, escapedPrefix)
		} else {
			forwarded.URL.RawPath = ""
		}
	}
	return t.base.RoundTrip(forwarded)
}
