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
	"fmt"
	"os"
	"sync"

	"github.com/golang/glog"
	"golang.org/x/oauth2"
	"google.golang.org/grpc/credentials"
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
	tokenSource        oauth2.TokenSource
	tokenSourceInitErr error
	tokenSourceOnce    sync.Once
)

// initTokenSource initializes the token source that watches the token file.
// This is called once lazily when the first request needs authentication.
func initTokenSource() error {
	tokenSourceOnce.Do(func() {
		// Check if token file exists before creating the token source
		if _, err := os.Stat(KFPTokenPath); err != nil {
			if os.IsNotExist(err) {
				glog.Warningf("KFP token file not found at %s, proceeding without authentication", KFPTokenPath)
				return
			}
			tokenSourceInitErr = fmt.Errorf("failed to stat KFP token file %s: %w", KFPTokenPath, err)
			return
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
	return tokenSourceInitErr
}

// getToken retrieves the current KFP service account token.
// The token is automatically reloaded when kubelet rotates it (no manual cache TTL).
// Returns empty string if the token file doesn't exist (e.g., dev environments).
func getToken() (string, error) {
	if err := initTokenSource(); err != nil {
		return "", err
	}

	if tokenSource == nil {
		return "", nil
	}

	tok, err := tokenSource.Token()
	if err != nil {
		return "", fmt.Errorf("failed to get token from token source: %w", err)
	}

	// oauth2.Token.AccessToken contains the actual token string
	if tok == nil || tok.AccessToken == "" {
		return "", nil
	}

	return tok.AccessToken, nil
}

type tokenPerRPCCredentials struct {
	requireTransportSecurity bool
}

var _ credentials.PerRPCCredentials = (*tokenPerRPCCredentials)(nil)

func newTokenPerRPCCredentials(requireTransportSecurity bool) credentials.PerRPCCredentials {
	return &tokenPerRPCCredentials{requireTransportSecurity: requireTransportSecurity}
}

func (c *tokenPerRPCCredentials) GetRequestMetadata(
	ctx context.Context,
	uri ...string,
) (map[string]string, error) {
	token, err := getToken()
	if err != nil {
		return nil, err
	}
	if token != "" {
		return map[string]string{"authorization": "Bearer " + token}, nil
	}
	return map[string]string{}, nil
}

func (c *tokenPerRPCCredentials) RequireTransportSecurity() bool {
	return c.requireTransportSecurity
}
