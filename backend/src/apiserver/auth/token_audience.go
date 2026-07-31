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

package auth

import (
	"context"

	"google.golang.org/grpc/metadata"
)

type expectedTokenAudiencesKey struct{}

// BoundRunIDMetadataKey is the outgoing/incoming gRPC metadata key used by
// runtime clients to declare the in-flight run ID for namespace-scoped RPCs
// such as FindCachedTask.
const BoundRunIDMetadataKey = "x-kfp-bound-run-id"

// WithExpectedTokenAudiences returns a child context that tells
// TokenReviewAuthenticator which audiences to validate against first.
func WithExpectedTokenAudiences(ctx context.Context, audiences []string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(audiences) == 0 {
		return ctx
	}
	copied := append([]string(nil), audiences...)
	return context.WithValue(ctx, expectedTokenAudiencesKey{}, copied)
}

// ExpectedTokenAudiences returns audiences previously attached with
// WithExpectedTokenAudiences.
func ExpectedTokenAudiences(ctx context.Context) ([]string, bool) {
	if ctx == nil {
		return nil, false
	}
	audiences, ok := ctx.Value(expectedTokenAudiencesKey{}).([]string)
	if !ok || len(audiences) == 0 {
		return nil, false
	}
	return audiences, true
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
