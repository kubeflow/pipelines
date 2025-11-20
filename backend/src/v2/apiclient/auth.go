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

	"github.com/golang/glog"
	"golang.org/x/oauth2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"k8s.io/client-go/transport"
)

const (
	// KFPTokenPath is the path where the projected service account token is mounted
	KFPTokenPath = "/var/run/secrets/kfp/token"
)

var (
	// tokenSource is initialized once and reused for the lifetime of the process.
	// It automatically watches the token file and reloads when kubelet rotates it.
	// Uses client-go's NewCachedFileTokenSource which periodically re-reads the file
	// (every minute by default) to pick up token rotations.
	tokenSource     oauth2.TokenSource
	tokenSourceOnce sync.Once
)

// initTokenSource initializes the token source that watches the token file.
// This is called once lazily when the first request needs authentication.
func initTokenSource() {
	tokenSourceOnce.Do(func() {
		// Check if token file exists before creating the token source
		if _, err := os.Stat(KFPTokenPath); err != nil {
			if os.IsNotExist(err) {
				glog.V(2).Infof("KFP token file not found at %s, proceeding without authentication", KFPTokenPath)
				tokenSource = &emptyTokenSource{}
				return
			}
			// Other errors - log but continue
			glog.Warningf("Error checking KFP token file at %s: %v", KFPTokenPath, err)
		}

		// Create a cached file token source that automatically reloads when the file changes.
		// This uses client-go's implementation which:
		// - Periodically re-reads the token file (every minute by default)
		// - Handles token rotation seamlessly
		// - Caches the token to avoid excessive disk I/O
		// This is the same approach used by client-go for in-cluster authentication.
		tokenSource = transport.NewCachedFileTokenSource(KFPTokenPath)
		glog.V(2).Infof("Initialized KFP token source from %s with automatic reload", KFPTokenPath)
	})
}

// emptyTokenSource returns an empty token (for dev/test environments without token files)
type emptyTokenSource struct{}

func (e *emptyTokenSource) Token() (*oauth2.Token, error) {
	return &oauth2.Token{}, nil
}

// getToken retrieves the current KFP service account token.
// The token is automatically reloaded when kubelet rotates it (no manual cache TTL).
// Returns empty string if the token file doesn't exist (e.g., dev environments).
func getToken() string {
	initTokenSource()

	if tokenSource == nil {
		return ""
	}

	tok, err := tokenSource.Token()
	if err != nil {
		glog.V(4).Infof("Failed to get token from token source: %v", err)
		return ""
	}

	// oauth2.Token.AccessToken contains the actual token string
	if tok == nil || tok.AccessToken == "" {
		return ""
	}

	return tok.AccessToken
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
