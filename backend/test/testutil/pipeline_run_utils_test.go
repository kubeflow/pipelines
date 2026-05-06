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

package testutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestResolveRunWaitDurations_DefaultTimeout(t *testing.T) {
	waitDuration, pollDuration := resolveRunWaitDurations(nil)

	assert.Equal(t, 300*time.Second, waitDuration)
	assert.Equal(t, 5*time.Second, pollDuration)
}

func TestResolveRunWaitDurations_ExplicitTimeout(t *testing.T) {
	timeout := 720 * time.Second

	waitDuration, pollDuration := resolveRunWaitDurations(&timeout)

	assert.Equal(t, 720*time.Second, waitDuration)
	assert.Equal(t, 5*time.Second, pollDuration)
}
