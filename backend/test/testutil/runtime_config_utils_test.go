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

package testutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaybeWithCoordinatorRuntimeSelector_Disabled(t *testing.T) {
	t.Setenv(useCoordinatorRuntimeForTestsEnvVar, "false")

	parameters := map[string]interface{}{"message": "hello"}

	selectedParameters := MaybeWithCoordinatorRuntimeSelector(parameters)

	selectedParameters["extra"] = "shared"

	assert.Equal(t, "shared", parameters["extra"])
}

func TestMaybeWithCoordinatorRuntimeSelector_Enabled(t *testing.T) {
	t.Setenv(useCoordinatorRuntimeForTestsEnvVar, "true")

	parameters := map[string]interface{}{"message": "hello"}

	selectedParameters := MaybeWithCoordinatorRuntimeSelector(parameters)

	selectedParameters["extra"] = "copy"

	assert.Equal(t, "copy", parameters["extra"])
	assert.Equal(t, "hello", selectedParameters["message"])
}

func TestMaybeWithCoordinatorRuntimeSelector_AlreadySelected(t *testing.T) {
	t.Setenv(useCoordinatorRuntimeForTestsEnvVar, "true")

	parameters := map[string]interface{}{
		"message": "hello",
	}

	selectedParameters := MaybeWithCoordinatorRuntimeSelector(parameters)

	selectedParameters["extra"] = "shared"

	assert.Equal(t, "shared", parameters["extra"])
}

func TestMaybeWithCoordinatorRuntimeSelector_NilEnabled(t *testing.T) {
	t.Setenv(useCoordinatorRuntimeForTestsEnvVar, "true")

	selectedParameters := MaybeWithCoordinatorRuntimeSelector(nil)

	assert.Nil(t, selectedParameters)
}
