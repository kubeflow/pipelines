package apiclient

import (
	"context"
	"os"
	"path/filepath"
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
	creds := newTokenPerRPCCredentials()
	if !creds.RequireTransportSecurity() {
		t.Fatalf("RequireTransportSecurity() = false, want true")
	}
}

func TestTokenPerRPCCredentialsGetRequestMetadata(t *testing.T) {
	t.Cleanup(func() {
		tokenSource = nil
		tokenSourceOnce = sync.Once{}
		t.Setenv(KFPTokenPathEnvVar, "")
	})

	t.Run("adds bearer token when available", func(t *testing.T) {
		setTokenSourceForTest("test-token")

		metadata, err := newTokenPerRPCCredentials().GetRequestMetadata(context.Background())
		if err != nil {
			t.Fatalf("GetRequestMetadata() error = %v", err)
		}
		if metadata["authorization"] != "Bearer test-token" {
			t.Fatalf("authorization = %q, want %q", metadata["authorization"], "Bearer test-token")
		}
	})

	t.Run("returns empty metadata when token missing", func(t *testing.T) {
		setTokenSourceForTest("")

		metadata, err := newTokenPerRPCCredentials().GetRequestMetadata(context.Background())
		if err != nil {
			t.Fatalf("GetRequestMetadata() error = %v", err)
		}
		if len(metadata) != 0 {
			t.Fatalf("GetRequestMetadata() = %#v, want empty metadata", metadata)
		}
	})
}

func TestGetToken_UsesStagedTokenPathAndDeletesFile(t *testing.T) {
	t.Cleanup(func() {
		tokenSource = nil
		tokenSourceOnce = sync.Once{}
		t.Setenv(KFPTokenPathEnvVar, "")
	})

	tokenFilePath := filepath.Join(t.TempDir(), "token")
	if err := os.WriteFile(tokenFilePath, []byte("staged-token"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Setenv(KFPTokenPathEnvVar, tokenFilePath)

	if got := getToken(); got != "staged-token" {
		t.Fatalf("getToken() = %q, want %q", got, "staged-token")
	}
	if _, err := os.Stat(tokenFilePath); !os.IsNotExist(err) {
		t.Fatalf("expected staged token file to be removed, stat error = %v", err)
	}
}
