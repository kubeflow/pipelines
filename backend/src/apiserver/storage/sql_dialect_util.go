// Copyright 2025 The Kubeflow Authors
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

package storage

import (
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
)

// quoteAll applies the dialect's QuoteIdentifier (q) to each column name and returns a new slice.
// Use this when passing columns into squirrel.Select(...), so each identifier is properly quoted
// for the current SQL dialect (e.g., Postgres requires double quotes to preserve case).
func quoteAll(q func(string) string, cols []string) []string {
	out := make([]string, len(cols))
	for i, c := range cols {
		out[i] = q(c)
	}
	return out
}

// qualifyIdentifier is an alias for filter.QualifyIdentifier for internal use.
var qualifyIdentifier = filter.QualifyIdentifier
