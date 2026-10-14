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
	"strings"
	"testing"
)

func TestQueueEvidenceUsesCounterDeltas(t *testing.T) {
	after := []byte(`apiserver_flowcontrol_current_inqueue_requests{priority_level="kfp-apf-experiment-jail"} 0
apiserver_flowcontrol_rejected_requests_total{priority_level="kfp-apf-experiment-jail",reason="time-out"} 2
apiserver_flowcontrol_request_wait_duration_seconds_sum{priority_level="kfp-apf-experiment-jail",execute="true"} 2.5
apiserver_flowcontrol_request_wait_duration_seconds_count{priority_level="kfp-apf-experiment-jail",execute="true"} 2
`)
	if !queuedDuring(nil, after) {
		t.Fatal("completed queue waits must remain observable after instantaneous queue drains")
	}
	if queuedDuring(after, after) {
		t.Fatal("stale counters must not prove queueing in a new interval")
	}
	if queuedDuring(after, nil) {
		t.Fatal("counter reset must not prove queueing")
	}
	if queuedDuring(nil, []byte(strings.ReplaceAll(string(after), "-jail", "-unrelated"))) {
		t.Fatal("other priority metrics must not prove experiment queueing")
	}
	if queuedDuring(nil, []byte(strings.ReplaceAll(string(after), `reason="time-out"`, `reason="queue-full"`))) {
		t.Fatal("queue-full rejections alone must not prove queued requests waited")
	}
}
