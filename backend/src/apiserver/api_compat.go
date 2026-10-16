// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0.

package main

import (
	"context"
	"net/http"
	"strings"

	// Register the frozen legacy descriptors for reflection, without loading a
	// second generated message or client implementation.
	_ "github.com/kubeflow/pipelines/backend/api/v2beta1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
)

const (
	canonicalAPIPath    = "/apis/v2"
	legacyAPIPath       = "/apis/v2beta1"
	canonicalRPCPackage = "kubeflow.pipelines.backend.api.v2."
	legacyRPCPackage    = "kubeflow.pipelines.backend.api.v2beta1."
)

var legacyAPIRequests = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "kfp_api_legacy_requests_total",
	Help: "Requests received on legacy v2beta1 entrypoints, including unsuccessful requests.",
}, []string{"protocol"})

// compatibleServiceRegistrar exposes the same v2 implementation under both
// service names. The v2 handler retains its interceptors and wire schema.
// Backward compatibility is checked against the frozen v2beta1 descriptors.
type compatibleServiceRegistrar struct {
	grpc.ServiceRegistrar
}

func (r compatibleServiceRegistrar) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	r.ServiceRegistrar.RegisterService(desc, impl)
	if !strings.HasPrefix(desc.ServiceName, canonicalRPCPackage) {
		return
	}
	legacy := *desc
	legacy.ServiceName = legacyRPCPackage + strings.TrimPrefix(desc.ServiceName, canonicalRPCPackage)
	legacy.Methods = append([]grpc.MethodDesc(nil), desc.Methods...)
	for i, method := range legacy.Methods {
		handler := method.Handler
		legacy.Methods[i].Handler = func(srv any, ctx context.Context, decode func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
			legacyAPIRequests.WithLabelValues("grpc").Inc()
			return handler(srv, ctx, decode, interceptor)
		}
	}
	legacy.Streams = append([]grpc.StreamDesc(nil), desc.Streams...)
	for i, stream := range legacy.Streams {
		handler := stream.Handler
		legacy.Streams[i].Handler = func(srv any, stream grpc.ServerStream) error {
			legacyAPIRequests.WithLabelValues("grpc").Inc()
			return handler(srv, stream)
		}
	}
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
		legacyAPIRequests.WithLabelValues("http").Inc()
		forwarded := r.Clone(r.Context())
		forwarded.URL.Path = canonicalAPIPath + strings.TrimPrefix(r.URL.Path, legacyAPIPath)
		if r.URL.RawPath != "" {
			forwarded.URL.RawPath = canonicalAPIPath + strings.TrimPrefix(r.URL.RawPath, legacyAPIPath)
		}
		canonical.ServeHTTP(w, forwarded)
	})
}
