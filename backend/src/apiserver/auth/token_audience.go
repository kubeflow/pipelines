// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package auth provides API server authentication helpers, including
// TokenReview-based identity checks and run-bound token audience validation.
package auth

import (
	"context"
	"fmt"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"google.golang.org/grpc/metadata"
)

// BoundRunIDMetadataKey is the outgoing/incoming gRPC metadata key used by
// runtime clients to declare the in-flight run ID for namespace-scoped RPCs
// such as FindCachedTask.
const BoundRunIDMetadataKey = "x-kfp-bound-run-id"

// AuthMethodTokenReview identifies principals authenticated via TokenReview.
const AuthMethodTokenReview = "tokenreview"

// TokenScope describes how a TokenReview principal is bound.
type TokenScope int

const (
	// TokenScopeUnspecified means no TokenReview principal was recorded
	// (for example HTTP-header identity).
	TokenScopeUnspecified TokenScope = iota
	// TokenScopeBroad means the token matched the configured base audience
	// and authorizes through normal RBAC.
	TokenScopeBroad
	// TokenScopeRun means the token matched only a run-scoped audience and
	// may access that single run.
	TokenScopeRun
)

// AuthenticatedPrincipal is the TokenReview classification stored on the
// request context for authorization decisions.
type AuthenticatedPrincipal struct {
	Username   string
	AuthMethod string
	Scope      TokenScope
	RunID      string
}

type requestedRunIDKey struct{}
type principalCollectorKey struct{}

// principalCollector is a mutable slot attached to context so authenticators
// can record the classified principal without changing GetUserIdentity's
// return type.
type principalCollector struct {
	principal *AuthenticatedPrincipal
}

// WithRequestedRunID marks the request as targeting a specific run for
// TokenReview. The authenticator reviews the run audience together with the
// configured base audience in a single call and records the resulting
// principal scope on this context.
func WithRequestedRunID(ctx context.Context, runID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if runID == "" {
		return ctx
	}
	ctx = context.WithValue(ctx, requestedRunIDKey{}, runID)
	if principalCollectorFromContext(ctx) == nil {
		ctx = context.WithValue(ctx, principalCollectorKey{}, &principalCollector{})
	}
	return ctx
}

// RequestedRunIDFromContext returns the run ID attached with WithRequestedRunID.
func RequestedRunIDFromContext(ctx context.Context) (string, bool) {
	if ctx == nil {
		return "", false
	}
	runID, ok := ctx.Value(requestedRunIDKey{}).(string)
	if !ok || runID == "" {
		return "", false
	}
	return runID, true
}

func principalCollectorFromContext(ctx context.Context) *principalCollector {
	if ctx == nil {
		return nil
	}
	collector, _ := ctx.Value(principalCollectorKey{}).(*principalCollector)
	return collector
}

func storeAuthenticatedPrincipal(ctx context.Context, principal AuthenticatedPrincipal) {
	collector := principalCollectorFromContext(ctx)
	if collector == nil {
		return
	}
	copied := principal
	collector.principal = &copied
}

// AuthenticatedPrincipalFromContext returns the TokenReview principal recorded
// during authentication, if any.
func AuthenticatedPrincipalFromContext(ctx context.Context) (*AuthenticatedPrincipal, bool) {
	collector := principalCollectorFromContext(ctx)
	if collector == nil || collector.principal == nil {
		return nil, false
	}
	copied := *collector.principal
	return &copied, true
}

// EnforceAuthenticatedRunScope rejects run-scoped TokenReview principals that
// are bound to a different run than requestedRunID. Broad / header identities
// are unaffected.
func EnforceAuthenticatedRunScope(ctx context.Context, requestedRunID string) error {
	if requestedRunID == "" {
		return nil
	}
	principal, ok := AuthenticatedPrincipalFromContext(ctx)
	if !ok || principal.Scope != TokenScopeRun {
		return nil
	}
	if principal.RunID == requestedRunID {
		return nil
	}
	return util.NewPermissionDeniedError(
		fmt.Errorf("run-scoped token for run %q cannot access run %q", principal.RunID, requestedRunID),
		"The runtime token is bound to a different pipeline run",
	)
}

// BoundRunIDFromIncomingContext reads the runtime-bound run ID from incoming
// gRPC metadata, if present.
func BoundRunIDFromIncomingContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return ""
	}
	values := md.Get(BoundRunIDMetadataKey)
	if len(values) == 0 {
		return ""
	}
	return values[0]
}
