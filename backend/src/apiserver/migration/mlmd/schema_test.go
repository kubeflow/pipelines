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

package mlmd_test

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestArtifactTaskProposalSQLDoesNotIndexMissingTimestamp(t *testing.T) {
	data, err := os.ReadFile("../../../../../proposals/12147-mlmd-removal/schema_changes.sql")
	require.NoError(t, err)
	shipped, err := os.ReadFile("schema.sql")
	require.NoError(t, err)
	require.Equal(t, string(data), string(shipped), "CLI schema must match the reviewed proposal SQL")
	statement := strings.SplitN(string(data), "CREATE TABLE `artifact_tasks`", 2)
	require.Len(t, statement, 2)
	require.NotContains(t, statement[1], "`CreatedAtInSec`",
		"artifact_tasks has no creation timestamp column to index")
}
