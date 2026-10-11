// Copyright 2026 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package constants

import "testing"

func TestAPIPipelineShardPartition(t *testing.T) {
	// The workflow consumes these literal labels, so renaming a constant must
	// not silently send all generated cases to its complementary shard.
	if got := APIPipelineShardForIndex(0); got != "ApiPipelineShardA" {
		t.Fatalf("first pipeline assigned to %q, want ApiPipelineShardA", got)
	}

	counts := map[string]int{"ApiPipelineShardA": 0, "ApiPipelineShardB": 0}
	previous := ""
	// Every prefix is an inventory size, including odd sizes and newly added
	// files beyond the current inventory. Each must stay balanced and covered.
	for index := 0; index < 1000; index++ {
		shard := APIPipelineShardForIndex(index)
		if _, ok := counts[shard]; !ok {
			t.Fatalf("pipeline %d has unsupported label %q", index, shard)
		}
		if shard == previous {
			t.Fatalf("adjacent pipelines %d and %d both use %q", index-1, index, shard)
		}
		counts[shard]++
		difference := counts["ApiPipelineShardA"] - counts["ApiPipelineShardB"]
		if difference < 0 || difference > 1 {
			t.Fatalf("inventory size %d has unbalanced shards: %v", index+1, counts)
		}
		previous = shard
	}
}
