// Copyright 2026 The Kubeflow Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package credentialrotation

import (
	"net/http"
	"sync/atomic"
	"time"
)

// Health describes recent successful reconciliation, not migration completion.
type Health struct{ lastSuccess atomic.Int64 }

func (h *Health) Observe(err error) {
	if err == nil || IsWaiting(err) {
		h.lastSuccess.Store(time.Now().UnixNano())
	} else {
		h.lastSuccess.Store(0)
	}
}

func (h *Health) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	last := h.lastSuccess.Load()
	if last == 0 || time.Since(time.Unix(0, last)) > 2*time.Minute {
		http.Error(w, "reconciliation unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}
