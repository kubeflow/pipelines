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

package testutil

import (
	"fmt"

	"github.com/google/uuid"
)

// NewTestResourceNameSuffix isolates resources created by each parallel test spec.
// Keep the full suffix when adding resource prefixes so no UUID bits are lost.
func NewTestResourceNameSuffix(worker int) string {
	return testResourceNameSuffix(worker, uuid.New())
}

func testResourceNameSuffix(worker int, id uuid.UUID) string {
	return fmt.Sprintf("%d-%s", worker, id)
}
