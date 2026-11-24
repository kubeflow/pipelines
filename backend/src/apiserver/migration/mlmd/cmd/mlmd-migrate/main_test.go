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

package main

import (
	"os"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/migration/mlmd"
	"github.com/stretchr/testify/require"
)

func TestProductionCLIRefusesCutover(t *testing.T) {
	if mlmd.CutoverEnabled {
		t.Skip("only the production build is held")
	}
	original := os.Args
	t.Cleanup(func() { os.Args = original })
	for _, args := range [][]string{
		{"mlmd-migrate"},
		{"mlmd-migrate", "--check-release-eligibility"},
	} {
		os.Args = args
		require.ErrorContains(t, run(), mlmd.CutoverHoldMessage)
	}
}
