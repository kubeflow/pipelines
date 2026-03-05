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

package client

import (
	"testing"
)

func TestJoinHostPort(t *testing.T) {
	tests := []struct {
		name string
		host string
		port string
		want string
	}{
		{
			name: "host and port",
			host: "minio-service",
			port: "9000",
			want: "minio-service:9000",
		},
		{
			name: "host with empty port",
			host: "minio-service.default.svc",
			port: "",
			want: "minio-service.default.svc",
		},
		{
			name: "ip address and port",
			host: "10.0.0.1",
			port: "443",
			want: "10.0.0.1:443",
		},
		{
			name: "empty host with port",
			host: "",
			port: "9000",
			want: ":9000",
		},
		{
			name: "both empty",
			host: "",
			port: "",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinHostPort(tt.host, tt.port)
			if got != tt.want {
				t.Errorf("joinHostPort(%q, %q) = %q, want %q", tt.host, tt.port, got, tt.want)
			}
		})
	}
}

func TestCreateMinioClient(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		port      string
		accessKey string
		secretKey string
		secure    bool
		region    string
		wantErr   bool
	}{
		{
			name:      "creates client with static credentials",
			host:      "minio-service",
			port:      "9000",
			accessKey: "access-key",
			secretKey: "secret-key",
			secure:    false,
			region:    "us-east-1",
			wantErr:   false,
		},
		{
			name:      "creates client without credentials (chained provider)",
			host:      "minio-service",
			port:      "9000",
			accessKey: "",
			secretKey: "",
			secure:    false,
			region:    "",
			wantErr:   false,
		},
		{
			name:      "creates client with secure connection",
			host:      "minio-service",
			port:      "443",
			accessKey: "key",
			secretKey: "secret",
			secure:    true,
			region:    "us-west-2",
			wantErr:   false,
		},
		{
			name:      "creates client with empty port",
			host:      "minio.example.com",
			port:      "",
			accessKey: "key",
			secretKey: "secret",
			secure:    true,
			region:    "",
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := CreateMinioClient(tt.host, tt.port, tt.accessKey, tt.secretKey, tt.secure, tt.region)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateMinioClient() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && client == nil {
				t.Error("CreateMinioClient() returned nil client without error")
			}
		})
	}
}

func TestCreateCredentialProvidersChain(t *testing.T) {
	t.Run("static credentials when both keys provided", func(t *testing.T) {
		cred := createCredentialProvidersChain("minio:9000", "mykey", "mysecret")
		if cred == nil {
			t.Fatal("createCredentialProvidersChain() returned nil")
		}
		value, err := cred.Get()
		if err != nil {
			t.Fatalf("cred.Get() unexpected error: %v", err)
		}
		if value.AccessKeyID != "mykey" {
			t.Errorf("AccessKeyID = %q, want %q", value.AccessKeyID, "mykey")
		}
		if value.SecretAccessKey != "mysecret" {
			t.Errorf("SecretAccessKey = %q, want %q", value.SecretAccessKey, "mysecret")
		}
	})

	t.Run("chained providers when access key is empty", func(t *testing.T) {
		cred := createCredentialProvidersChain("minio:9000", "", "mysecret")
		if cred == nil {
			t.Fatal("createCredentialProvidersChain() returned nil")
		}
	})

	t.Run("chained providers when secret key is empty", func(t *testing.T) {
		cred := createCredentialProvidersChain("minio:9000", "mykey", "")
		if cred == nil {
			t.Fatal("createCredentialProvidersChain() returned nil")
		}
	})

	t.Run("chained providers when both keys are empty", func(t *testing.T) {
		cred := createCredentialProvidersChain("minio:9000", "", "")
		if cred == nil {
			t.Fatal("createCredentialProvidersChain() returned nil")
		}
	})
}
