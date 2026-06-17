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
	"testing"
)

func TestParseTLSVersion(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected uint16
	}{
		{"TLS 1.2", "VersionTLS12", tls.VersionTLS12},
		{"TLS 1.3", "VersionTLS13", tls.VersionTLS13},
		{"empty defaults to TLS 1.2", "", tls.VersionTLS12},
		{"unrecognized defaults to TLS 1.2", "SomeBogusVersion", tls.VersionTLS12},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTLSVersion(tt.input)
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
	}{
		{"empty returns nil", "", 0},
		{"single valid cipher", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", 1},
		{"multiple valid ciphers", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384", 2},
		{"unknown cipher skipped", "BOGUS_CIPHER", 0},
		{"mixed valid and invalid", "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,BOGUS", 1},
		{"whitespace trimmed", " TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 , TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384 ", 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTLSCipherSuites(tt.input)
			if len(got) != tt.wantCount {
				t.Errorf("parseTLSCipherSuites(%q) returned %d ciphers, want %d", tt.input, len(got), tt.wantCount)
			}
		})
	}
}

func TestBuildTLSConfig(t *testing.T) {
	origMin := tlsMinVersion
	origCiphers := tlsCipherSuites
	t.Cleanup(func() {
		tlsMinVersion = origMin
		tlsCipherSuites = origCiphers
	})
	defaultMin := "VersionTLS12"
	defaultCiphers := ""
	tlsMinVersion = &defaultMin
	tlsCipherSuites = &defaultCiphers

	cfg := buildTLSConfig()

	if cfg.MinVersion != tls.VersionTLS12 {
		t.Errorf("MinVersion = %d, want %d", cfg.MinVersion, tls.VersionTLS12)
	}
	if len(cfg.NextProtos) != 2 || cfg.NextProtos[0] != "h2" || cfg.NextProtos[1] != "http/1.1" {
		t.Errorf("NextProtos = %v, want [h2 http/1.1]", cfg.NextProtos)
	}
	if cfg.CipherSuites != nil {
		t.Errorf("CipherSuites should be nil when flag is empty, got %v", cfg.CipherSuites)
	}
}
