package apiclient

import (
	"context"
	"sync"
	"testing"

	"golang.org/x/oauth2"
)

func setTokenSourceForTest(token string) {
	tokenSourceOnce = sync.Once{}
	tokenSourceOnce.Do(func() {})
	tokenSource = oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
}

func TestTokenPerRPCCredentialsRequireTransportSecurity(t *testing.T) {
	t.Run("requires transport security when configured", func(t *testing.T) {
		creds := newTokenPerRPCCredentials(true)
		if !creds.RequireTransportSecurity() {
			t.Fatalf("RequireTransportSecurity() = false, want true")
		}
	})

	t.Run("allows insecure transport when configured", func(t *testing.T) {
		creds := newTokenPerRPCCredentials(false)
		if creds.RequireTransportSecurity() {
			t.Fatalf("RequireTransportSecurity() = true, want false")
		}
	})
}

func TestTokenPerRPCCredentialsGetRequestMetadata(t *testing.T) {
	t.Cleanup(func() {
		tokenSource = nil
		tokenSourceOnce = sync.Once{}
	})

	t.Run("adds bearer token when available", func(t *testing.T) {
		setTokenSourceForTest("test-token")

		metadata, err := newTokenPerRPCCredentials(true).GetRequestMetadata(context.Background())
		if err != nil {
			t.Fatalf("GetRequestMetadata() error = %v", err)
		}
		if metadata["authorization"] != "Bearer test-token" {
			t.Fatalf("authorization = %q, want %q", metadata["authorization"], "Bearer test-token")
		}
	})

	t.Run("returns empty metadata when token missing", func(t *testing.T) {
		setTokenSourceForTest("")

		metadata, err := newTokenPerRPCCredentials(false).GetRequestMetadata(context.Background())
		if err != nil {
			t.Fatalf("GetRequestMetadata() error = %v", err)
		}
		if len(metadata) != 0 {
			t.Fatalf("GetRequestMetadata() = %#v, want empty metadata", metadata)
		}
	})
}
