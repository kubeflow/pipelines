// Copyright 2026 The Kubeflow Authors
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

package util

import (
	"context"
	"net/http"
	"net/url"
)

type proxyContextKey struct{}

// WithHTTPProxy returns a context with an invocation-local HTTP proxy resolver.
// A nil resolver explicitly selects direct connections instead of process settings.
func WithHTTPProxy(ctx context.Context, resolver func(*http.Request) (*url.URL, error)) context.Context {
	return context.WithValue(ctx, proxyContextKey{}, resolver)
}

// HTTPProxyFrom returns the invocation's resolver and whether it overrides process settings.
func HTTPProxyFrom(ctx context.Context) (func(*http.Request) (*url.URL, error), bool) {
	resolver, ok := ctx.Value(proxyContextKey{}).(func(*http.Request) (*url.URL, error))
	return resolver, ok
}
