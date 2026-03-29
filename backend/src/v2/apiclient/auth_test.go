package apiclient

import (
	"context"
	"errors"
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

func setTokenSourceForTest(token string) {
	tokenSourceOnce = sync.Once{}
	tokenSourceOnce.Do(func() {})
	tokenSource = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tokenSourceInitErr = nil
}

func TestTokenPerRPCCredentialsRequireTransportSecurity(t *testing.T) {
	t.Run("requires transport security when configured", func(t *testing.T) {
		creds := newTokenPerRPCCredentials(true, nil)
		if !creds.RequireTransportSecurity() {
			t.Fatalf("RequireTransportSecurity() = false, want true")
		}
	})

	t.Run("allows insecure transport when configured", func(t *testing.T) {
		creds := newTokenPerRPCCredentials(false, nil)
		if creds.RequireTransportSecurity() {
			t.Fatalf("RequireTransportSecurity() = true, want false")
		}
	})
}

func TestTokenPerRPCCredentialsGetRequestMetadata(t *testing.T) {
	t.Cleanup(func() {
		tokenSource = nil
		tokenSourceInitErr = nil
		tokenSourceOnce = sync.Once{}
	})

	t.Run("adds bearer token when available", func(t *testing.T) {
		setTokenSourceForTest("test-token")

		metadata, err := newTokenPerRPCCredentials(true, nil).GetRequestMetadata(context.Background())
		if err != nil {
			t.Fatalf("GetRequestMetadata() error = %v", err)
		}
		if metadata["authorization"] != "Bearer test-token" {
			t.Fatalf("authorization = %q, want %q", metadata["authorization"], "Bearer test-token")
		}
	})

	t.Run("returns empty metadata when token missing", func(t *testing.T) {
		setTokenSourceForTest("")

		metadata, err := newTokenPerRPCCredentials(false, nil).GetRequestMetadata(context.Background())
		if err != nil {
			t.Fatalf("GetRequestMetadata() error = %v", err)
		}
		if len(metadata) != 0 {
			t.Fatalf("GetRequestMetadata() = %#v, want empty metadata", metadata)
		}
	})

	t.Run("propagates token source errors", func(t *testing.T) {
		tokenSourceInitErr = assertiveError("boom")

		_, err := newTokenPerRPCCredentials(true, nil).GetRequestMetadata(context.Background())
		if err == nil || err.Error() != "boom" {
			t.Fatalf("GetRequestMetadata() error = %v, want boom", err)
		}
	})
}

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

type assertiveError string

func (e assertiveError) Error() string {
	return string(e)
}
