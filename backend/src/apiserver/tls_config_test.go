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

package main

import (
	"crypto/tls"
	"strings"
	"testing"
)

func TestParseTLSVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint16
		wantErr  bool
	}{
		{"empty defaults to TLS 1.2", "", tls.VersionTLS12, false},
		{"TLS 1.2", "VersionTLS12", tls.VersionTLS12, false},
		{"TLS 1.3", "VersionTLS13", tls.VersionTLS13, false},
		{"whitespace trimmed TLS 1.2", " VersionTLS12 ", tls.VersionTLS12, false},
		{"whitespace trimmed TLS 1.3", " VersionTLS13 ", tls.VersionTLS13, false},
		{"bogus version is error", "bogus", 0, true},
		{"VersionTLS11 is error", "VersionTLS11", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTLSVersion(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseTLSVersion(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseTLSVersion(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.expected {
				t.Errorf("parseTLSVersion(%q) = %d, want %d", tt.input, got, tt.expected)
			}
		})
	}
}

func TestParseTLSCipherSuites(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantCount int
		wantErr   bool
	}{
		{"empty returns nil", "", 0, false},
		{"single valid cipher", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", 1, false},
		{"multiple valid ciphers", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", 2, false},
		{"whitespace trimmed", " TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 , TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 ", 2, false},
		{"insecure cipher accepted", "TLS_RSA_WITH_RC4_128_SHA", 1, false},
		{"unknown cipher is error", "BOGUS_CIPHER", 0, true},
		{"mixed valid and invalid is error", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,BOGUS", 0, true},
		{"only commas is error", ",,", 0, true},
		{"whitespace only returns nil", "", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTLSCipherSuites(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("parseTLSCipherSuites(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("parseTLSCipherSuites(%q) unexpected error: %v", tt.input, err)
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("parseTLSCipherSuites(%q) returned %d ciphers, want %d", tt.input, len(got), tt.wantCount)
			}
		})
	}
}

func TestBuildTLSConfig(t *testing.T) {
	tests := []struct {
		name           string
		minVersion     string
		cipherSuites   string
		wantErr        bool
		errContains    string
		wantMinVersion uint16
		wantCiphers    int
	}{
		{
			name:           "defaults: TLS 1.2, no ciphers",
			minVersion:     "",
			cipherSuites:   "",
			wantMinVersion: tls.VersionTLS12,
			wantCiphers:    0,
		},
		{
			name:           "explicit TLS 1.2, no ciphers",
			minVersion:     "VersionTLS12",
			cipherSuites:   "",
			wantMinVersion: tls.VersionTLS12,
			wantCiphers:    0,
		},
		{
			name:           "TLS 1.2 with one cipher",
			minVersion:     "VersionTLS12",
			cipherSuites:   "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			wantMinVersion: tls.VersionTLS12,
			wantCiphers:    1,
		},
		{
			name:           "TLS 1.2 with two ciphers",
			minVersion:     "VersionTLS12",
			cipherSuites:   "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
			wantMinVersion: tls.VersionTLS12,
			wantCiphers:    2,
		},
		{
			name:           "TLS 1.2 with insecure cipher accepted",
			minVersion:     "VersionTLS12",
			cipherSuites:   "TLS_RSA_WITH_RC4_128_SHA",
			wantMinVersion: tls.VersionTLS12,
			wantCiphers:    1,
		},
		{
			name:           "TLS 1.3, no ciphers",
			minVersion:     "VersionTLS13",
			cipherSuites:   "",
			wantMinVersion: tls.VersionTLS13,
			wantCiphers:    0,
		},
		{
			name:           "unset version with one cipher",
			minVersion:     "",
			cipherSuites:   "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			wantMinVersion: tls.VersionTLS12,
			wantCiphers:    1,
		},
		{
			name:           "whitespace-only ciphers treated as empty",
			minVersion:     "VersionTLS12",
			cipherSuites:   "   ",
			wantMinVersion: tls.VersionTLS12,
			wantCiphers:    0,
		},
		{
			name:           "whitespace version trimmed",
			minVersion:     " VersionTLS12 ",
			cipherSuites:   "",
			wantMinVersion: tls.VersionTLS12,
			wantCiphers:    0,
		},
		// Error cases
		{
			name:         "bogus version is error",
			minVersion:   "bogus",
			cipherSuites: "",
			wantErr:      true,
			errContains:  "invalid --tlsMinVersion",
		},
		{
			name:         "bogus version with valid ciphers is error",
			minVersion:   "bogus",
			cipherSuites: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			wantErr:      true,
			errContains:  "invalid --tlsMinVersion",
		},
		{
			name:         "all bogus ciphers is error",
			minVersion:   "VersionTLS12",
			cipherSuites: "BOGUS_CIPHER",
			wantErr:      true,
			errContains:  "unknown TLS cipher suite",
		},
		{
			name:         "mixed valid and invalid ciphers is error",
			minVersion:   "VersionTLS12",
			cipherSuites: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,BOGUS",
			wantErr:      true,
			errContains:  "unknown TLS cipher suite",
		},
		{
			name:         "unset version with bogus cipher is error",
			minVersion:   "",
			cipherSuites: "BOGUS_CIPHER",
			wantErr:      true,
			errContains:  "unknown TLS cipher suite",
		},
		{
			name:         "only commas is error",
			minVersion:   "VersionTLS12",
			cipherSuites: ",,",
			wantErr:      true,
			errContains:  "no cipher suites were specified",
		},
		{
			name:         "TLS 1.3 with ciphers is error",
			minVersion:   "VersionTLS13",
			cipherSuites: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
			wantErr:      true,
			errContains:  "cannot be used with TLS 1.3",
		},
		{
			name:         "TLS 1.3 with bogus cipher is error",
			minVersion:   "VersionTLS13",
			cipherSuites: "BOGUS_CIPHER",
			wantErr:      true,
			errContains:  "cannot be used with TLS 1.3",
		},
		{
			name:         "TLS 1.3 with insecure cipher is error",
			minVersion:   "VersionTLS13",
			cipherSuites: "TLS_RSA_WITH_RC4_128_SHA",
			wantErr:      true,
			errContains:  "cannot be used with TLS 1.3",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			origMin := tlsMinVersion
			origCiphers := tlsCipherSuites
			t.Cleanup(func() {
				tlsMinVersion = origMin
				tlsCipherSuites = origCiphers
			})
			minVer := tt.minVersion
			ciphers := tt.cipherSuites
			tlsMinVersion = &minVer
			tlsCipherSuites = &ciphers

			cfg, err := buildTLSConfig()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("buildTLSConfig() expected error containing %q, got nil", tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("buildTLSConfig() error = %q, want it to contain %q", err.Error(), tt.errContains)
				}
				return
			}
			if err != nil {
				t.Fatalf("buildTLSConfig() unexpected error: %v", err)
			}
			if cfg.MinVersion != tt.wantMinVersion {
				t.Errorf("MinVersion = %d, want %d", cfg.MinVersion, tt.wantMinVersion)
			}
			if len(cfg.CipherSuites) != tt.wantCiphers {
				t.Errorf("CipherSuites count = %d, want %d", len(cfg.CipherSuites), tt.wantCiphers)
			}
			if len(cfg.NextProtos) != 2 || cfg.NextProtos[0] != "h2" || cfg.NextProtos[1] != "http/1.1" {
				t.Errorf("NextProtos = %v, want [h2 http/1.1]", cfg.NextProtos)
			}
		})
	}
}
