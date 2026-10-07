// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"net/http"
	"strings"

	// Retain the legacy descriptors for gRPC reflection. Requests are decoded
	// directly into the wire-compatible v2 messages, not legacy server types.
	_ "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"google.golang.org/grpc"
)

const (
	canonicalAPIPath    = "/apis/v2"
	legacyAPIPath       = "/apis/v2beta1"
	canonicalRPCPackage = "kubeflow.pipelines.backend.api.v2."
	legacyRPCPackage    = "kubeflow.pipelines.backend.api.v2beta1."
)

// compatibleServiceRegistrar exposes the same v2 implementation under both
// service names. The v2 handler retains its interceptors and wire schema.
// Schema parity is checked against the frozen v2beta1 descriptors in tests.
type compatibleServiceRegistrar struct {
	grpc.ServiceRegistrar
}

func (r compatibleServiceRegistrar) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	r.ServiceRegistrar.RegisterService(desc, impl)
	legacy := *desc
	legacy.ServiceName = strings.Replace(desc.ServiceName, canonicalRPCPackage, legacyRPCPackage, 1)
	if filename, ok := desc.Metadata.(string); ok {
		legacy.Metadata = strings.Replace(filename, "backend/api/v2/", "backend/api/v2beta1/", 1)
	}
	r.ServiceRegistrar.RegisterService(&legacy, impl)
}

// legacyAPIHandler rewrites only the version prefix before normal routing.
// No redirect, body buffering, or second HTTP request is involved, so uploads,
// streaming, credentials, cancellation, and gateway middleware stay intact.
func legacyAPIHandler(canonical http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded := r.Clone(r.Context())
		forwarded.URL.Path = canonicalAPIPath + strings.TrimPrefix(r.URL.Path, legacyAPIPath)
		if r.URL.RawPath != "" {
			forwarded.URL.RawPath = canonicalAPIPath + strings.TrimPrefix(r.URL.RawPath, legacyAPIPath)
		}
		canonical.ServeHTTP(w, forwarded)
	})
}
