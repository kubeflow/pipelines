// Copyright 2025 The Kubeflow Authors
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

// Package apiclient provides API client functionality for KFP v2.
package apiclient

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/golang/glog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

const (
	// KFPTokenPath is the path where the projected service account token is mounted
	KFPTokenPath = "/var/run/secrets/kfp/token"
	// TokenCacheTTL is how long we cache the token before re-reading from disk
	TokenCacheTTL = 5 * time.Minute
)

// tokenCache holds a cached token and its expiry time
type tokenCache struct {
	mu        sync.RWMutex
	token     string
	expiresAt time.Time
}

var cache = &tokenCache{}

// getToken reads the KFP service account token from the projected volume.
// It caches the token for TokenCacheTTL to avoid reading from disk on every request.
// Returns empty string if the token file doesn't exist (e.g., dev environments).
func getToken() string {
	cache.mu.RLock()
	// Check if we have a valid cached token
	if cache.token != "" && time.Now().Before(cache.expiresAt) {
		token := cache.token
		cache.mu.RUnlock()
		return token
	}
	cache.mu.RUnlock()

	// Need to refresh the token
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// Double-check after acquiring write lock
	if cache.token != "" && time.Now().Before(cache.expiresAt) {
		return cache.token
	}

	// Read token from file
	tokenBytes, err := os.ReadFile(KFPTokenPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Token file doesn't exist - likely dev/test environment
			// Log once per cache refresh cycle to avoid spam
			if cache.token == "" {
				glog.V(2).Infof("KFP token file not found at %s, proceeding without authentication", KFPTokenPath)
			}
			cache.token = ""
			cache.expiresAt = time.Now().Add(TokenCacheTTL)
			return ""
		}
		// Other errors (permissions, I/O) - log warning but don't fail
		glog.Warningf("Failed to read KFP token from %s: %v, proceeding without authentication", KFPTokenPath, err)
		cache.token = ""
		cache.expiresAt = time.Now().Add(1 * time.Minute) // Retry sooner on errors
		return ""
	}

	token := string(tokenBytes)
	cache.token = token
	cache.expiresAt = time.Now().Add(TokenCacheTTL)

	glog.V(3).Infof("Successfully loaded KFP token from %s", KFPTokenPath)
	return token
}

// authUnaryInterceptor is a gRPC unary interceptor that adds the Authorization header
// to all outgoing requests using the KFP service account token.
func authUnaryInterceptor(
	ctx context.Context,
	method string,
	req interface{},
	reply interface{},
	cc *grpc.ClientConn,
	invoker grpc.UnaryInvoker,
	opts ...grpc.CallOption,
) error {
	token := getToken()
	if token != "" {
		// Add Authorization header with Bearer token
		ctx = metadata.AppendToOutgoingContext(ctx, "authorization", "Bearer "+token)
	}
	return invoker(ctx, method, req, reply, cc, opts...)
}
