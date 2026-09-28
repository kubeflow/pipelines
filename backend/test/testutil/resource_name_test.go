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
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/validation"
)

func TestResourceNameSuffixSeparatesWorkers(t *testing.T) {
	// Even identical per-spec IDs cannot collide across parallel workers.
	id := uuid.MustParse("11111111-2222-4333-8444-555555555555")
	seen := make(map[string]bool)
	for worker := 1; worker <= 5; worker++ {
		name := "apitest-" + testResourceNameSuffix(worker, id)
		require.False(t, seen[name], "worker %d reused another worker's pipeline name", worker)
		seen[name] = true
	}
}

func TestResourceNameSuffixPreservesSpecIdentity(t *testing.T) {
	// These IDs differ only at the end: truncating to the old 17-character
	// suffix would discard the distinction even within a single worker.
	first := testResourceNameSuffix(1, uuid.MustParse("11111111-2222-4333-8444-555555555551"))
	second := testResourceNameSuffix(1, uuid.MustParse("11111111-2222-4333-8444-555555555552"))
	for _, prefix := range []string{"apitest-", "filter-test-", "ut-", "dt-"} {
		require.NotEqual(t, prefix+first, prefix+second, "prefix %q lost the per-spec identity", prefix)
	}
}

func TestNewTestResourceNameSuffixParallelSpecs(t *testing.T) {
	const workers, specsPerWorker = 5, 100
	names := make(chan string, workers*specsPerWorker)
	var wg sync.WaitGroup
	for worker := 1; worker <= workers; worker++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for spec := 0; spec < specsPerWorker; spec++ {
				names <- NewTestResourceNameSuffix(worker)
			}
		}(worker)
	}
	wg.Wait()
	close(names)

	seen := make(map[string]bool)
	for suffix := range names {
		require.False(t, seen[suffix], "a spec reused a resource name")
		seen[suffix] = true
		// Cover API suite resource names and label values using the complete
		// suffix, including the longest resource-name prefix and postfix.
		for _, name := range []string{
			"apitest-" + suffix,
			"apitest-" + suffix + "-display-desc-1",
			"filter-test-" + suffix,
			"pgx-ns-test-" + suffix + "-run",
			"pgx-test-" + suffix + "-run",
		} {
			require.Empty(t, validation.IsDNS1123Label(name), "invalid resource name: %s", name)
		}
		for _, prefix := range []string{"ut-", "dt-"} {
			value := prefix + suffix
			require.Empty(t, validation.IsValidLabelValue(value), "invalid label value: %s", value)
		}
	}
	require.Len(t, seen, workers*specsPerWorker)
}
