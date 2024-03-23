// Copyright 2020 The Kubeflow Authors
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

package server

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPodSucceeded(t *testing.T) {
	runningPod := *fakePod.DeepCopy()
	runningPod.Status.Phase = "Running"
	podIncomplete := isPodSucceeded(fakePod)
	assert.False(t, podIncomplete)
	// Assign pod as succeeded.
	completedPod := *fakePod.DeepCopy()
	completedPod.Status.Phase = "Succeeded"
	podComplete := isPodSucceeded(&completedPod)
	assert.True(t, podComplete)
}

func TestIsCacheWritten(t *testing.T) {
	cacheNotWritten := isCacheWriten(fakePod.ObjectMeta.Labels)
	assert.False(t, cacheNotWritten)
	// Mutate pod with a cache.
	mutatedPod := *fakePod.DeepCopy()
	mutatedPod.ObjectMeta.Labels[CacheIDLabelKey] = "1234"
	cacheWritten := isCacheWriten(mutatedPod.ObjectMeta.Labels)
	assert.True(t, cacheWritten)
}
