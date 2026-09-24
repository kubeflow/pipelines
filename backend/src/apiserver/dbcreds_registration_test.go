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

package main

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/dbcreds"
	"github.com/stretchr/testify/assert"
)

// TestCredentialProvidersAreRegistered guards the blank import of
// dbcreds/all. Without it the binary builds and starts, and only fails once an
// operator sets DB_CREDENTIAL_PROVIDER, so a compile-time check is not enough.
func TestCredentialProvidersAreRegistered(t *testing.T) {
	// The provider name is spelled out rather than imported from its package:
	// importing it here would run its init and register it, so the test would
	// pass even with the blank import gone.
	registered := dbcreds.RegisteredNames()
	assert.Contains(t, registered, dbcreds.StaticProviderName)
	assert.Contains(t, registered, "aws-iam")
}
