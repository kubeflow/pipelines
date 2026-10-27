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

package apiclient

import (
	"context"
	"errors"
	"sync"
	"testing"
)

type tokenSourceFunc func(context.Context) (string, error)

func (source tokenSourceFunc) Token(ctx context.Context) (string, error) {
	return source(ctx)
}

func TestTokenPerRPCCredentialsUsesInjectedSource(t *testing.T) {
	tokenSourceInitErr = errors.New("file source must not be used")
	tokenSourceOnce = sync.Once{}
	tokenSourceOnce.Do(func() {})
	t.Cleanup(func() {
		tokenSourceInitErr = nil
		tokenSourceOnce = sync.Once{}
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sourceErr := errors.New("token request denied")
	tests := []struct {
		name      string
		token     string
		sourceErr error
		wantErr   bool
	}{
		{name: "first client", token: "first-token"},
		{name: "second client", token: "second-token"},
		{name: "empty source fails closed", wantErr: true},
		{name: "source error", sourceErr: sourceErr, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			creds := newTokenPerRPCCredentials(false, tokenSourceFunc(func(received context.Context) (string, error) {
				if received != ctx {
					t.Fatal("token source did not receive RPC context")
				}
				return test.token, test.sourceErr
			}))
			metadata, err := creds.GetRequestMetadata(ctx)
			if test.wantErr {
				if err == nil || metadata != nil {
					t.Fatalf("expected error without metadata, got %v, %v", metadata, err)
				}
				if test.sourceErr != nil && !errors.Is(err, test.sourceErr) {
					t.Fatalf("expected source error, got %v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if metadata["authorization"] != "Bearer "+test.token {
				t.Fatal("credentials used a different client's token")
			}
		})
	}
}
