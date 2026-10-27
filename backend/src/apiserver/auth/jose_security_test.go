// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auth_test

import (
	"encoding/base64"
	"encoding/json"
	"math/big"
	"strings"
	"testing"

	"github.com/go-jose/go-jose/v4"
)

// Guard the transitive JOSE parser against GHSA-mgcc-cww5-fp2q without
// depending on timing or performing expensive RSA private-key validation.
func TestJOSERejectsUnsafeEmbeddedRSAKeys(t *testing.T) {
	for _, tc := range []struct {
		name      string
		bits      uint
		private   bool
		wantError string
	}{
		{name: "public key", bits: 2048},
		{name: "oversized modulus", bits: 4097, wantError: "modulus too large"},
		{name: "private parameters", bits: 2048, private: true, wantError: "JWK contains private parameters"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			n := new(big.Int).Lsh(big.NewInt(1), tc.bits-1)
			n.SetBit(n, 0, 1)
			key := map[string]string{
				"kty": "RSA",
				"n":   base64.RawURLEncoding.EncodeToString(n.Bytes()),
				"e":   "AQAB",
			}
			if tc.private {
				key["d"] = "Aw"
				key["p"] = "Aw"
				key["q"] = "Aw"
			}
			header, err := json.Marshal(map[string]interface{}{"alg": "RS256", "jwk": key})
			if err != nil {
				t.Fatal(err)
			}
			// Parsing a public header is distinct from trusting its key or
			// verifying a signature; this fixture intentionally has no valid signature.
			token := base64.RawURLEncoding.EncodeToString(header) + ".e30.c2lnbmF0dXJl"
			_, err = jose.ParseSigned(token, []jose.SignatureAlgorithm{jose.RS256})
			if tc.wantError == "" {
				if err != nil {
					t.Fatalf("parse bounded public key: %v", err)
				}
			} else if err == nil || !strings.Contains(err.Error(), tc.wantError) {
				t.Fatalf("want early rejection %q, got %v", tc.wantError, err)
			}
		})
	}
}
