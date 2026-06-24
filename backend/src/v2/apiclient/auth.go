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
	"strings"
	"sync"

	"github.com/golang/glog"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials"
	"k8s.io/client-go/transport"
)

const (
	// KFPTokenPath is the path where the projected service account token is mounted
	KFPTokenPath = "/var/run/secrets/kfp/token"
	// KFPTokenPathEnvVar overrides the token path for executor launchers that stage
	// the token into a private scratch volume before user code starts.
	KFPTokenPathEnvVar = "KFP_TOKEN_PATH"
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
		tokenPath := resolveTokenPath()
		// Check if token file exists before creating the token source
		if _, err := os.Stat(tokenPath); err != nil {
			if os.IsNotExist(err) {
				glog.Warningf("KFP token file not found at %s, proceeding without authentication", tokenPath)
				tokenSource = &emptyTokenSource{}
				return
			}
			// Other errors - log but continue
			glog.Warningf("Error checking KFP token file at %s: %v", tokenPath, err)
		}

		if tokenPath != KFPTokenPath {
			tokenBytes, err := os.ReadFile(tokenPath)
			if err != nil {
				glog.Warningf("Failed to read staged KFP token from %s: %v", tokenPath, err)
				tokenSource = &emptyTokenSource{}
				return
			}
			if err := os.Remove(tokenPath); err != nil {
				glog.Warningf("Failed to remove staged KFP token at %s: %v", tokenPath, err)
			}
			token := strings.TrimSpace(string(tokenBytes))
			tokenSource = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
			return
		}

		// Create a cached file token source that automatically reloads when the file changes.
		// This uses client-go's implementation which:
		// - Periodically re-reads the token file (every minute by default)
		// - Handles token rotation seamlessly
		// - Caches the token to avoid excessive disk I/O
		// This is the same approach used by client-go for in-cluster authentication.
		tokenSource = transport.NewCachedFileTokenSource(tokenPath)
		glog.V(2).Infof("Initialized KFP token source from %s with automatic reload", tokenPath)
	})
}

func resolveTokenPath() string {
	if tokenPath := os.Getenv(KFPTokenPathEnvVar); tokenPath != "" {
		return tokenPath
	}
	return KFPTokenPath
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

type tokenPerRPCCredentials struct{}

var _ credentials.PerRPCCredentials = (*tokenPerRPCCredentials)(nil)

func newTokenPerRPCCredentials() credentials.PerRPCCredentials {
	return &tokenPerRPCCredentials{}
}

func (c *tokenPerRPCCredentials) GetRequestMetadata(
	ctx context.Context,
	uri ...string,
) (map[string]string, error) {
	token := getToken()
	if token != "" {
		return map[string]string{"authorization": "Bearer " + token}, nil
	}
	return map[string]string{}, nil
}

func (c *tokenPerRPCCredentials) RequireTransportSecurity() bool {
	return true
}
