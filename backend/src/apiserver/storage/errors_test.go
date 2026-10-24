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

package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
)

func TestStorageIsNotFound(t *testing.T) {
	for _, test := range []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"sql", sql.ErrNoRows, true},
		{"wrapped sql", fmt.Errorf("get runtime: %w", sql.ErrNoRows), true},
		{"resource", util.NewResourceNotFoundError("artifact", "id"), true},
		{"wrapped resource", fmt.Errorf("publish: %w", util.NewResourceNotFoundError("artifact", "id")), true},
		{"different user error", util.NewInvalidInputError("bad request"), false},
		{"other error", errors.New("database unavailable"), false},
	} {
		t.Run(test.name, func(t *testing.T) { assert.Equal(t, test.want, IsNotFound(test.err)) })
	}
}
