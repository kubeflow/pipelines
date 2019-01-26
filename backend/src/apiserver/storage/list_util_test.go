// Copyright 2019 Google LLC
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

package storage

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/stretchr/testify/assert"
)

func TestFilterOnResourceReference(t *testing.T) {

	type testIn struct {
		table        string
		resourceType common.ResourceType
		count        bool
		filter       *common.FilterContext
	}
	tests := []struct {
		in      *testIn
		wantSql string
		wantErr error
	}{
		{
			in: &testIn{
				table:        "testTable",
				resourceType: common.Run,
				count:        false,
				filter:       &common.FilterContext{},
			},
			wantSql: "SELECT * FROM testTable",
			wantErr: nil,
		},
		{
			in: &testIn{
				table:        "testTable",
				resourceType: common.Run,
				count:        true,
				filter:       &common.FilterContext{},
			},
			wantSql: "SELECT count(*) FROM testTable",
			wantErr: nil,
		},
		{
			in: &testIn{
				table:        "testTable",
				resourceType: common.Run,
				count:        false,
				filter:       &common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Run}},
			},
			wantSql: "SELECT * FROM testTable WHERE UUID in (SELECT ResourceUUID FROM resource_references as rf WHERE (rf.ResourceType = ? AND rf.ReferenceUUID = ? AND rf.ReferenceType = ?))",
			wantErr: nil,
		},
		{
			in: &testIn{
				table:        "testTable",
				resourceType: common.Run,
				count:        true,
				filter:       &common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Run}},
			},
			wantSql: "SELECT count(*) FROM testTable WHERE UUID in (SELECT ResourceUUID FROM resource_references as rf WHERE (rf.ResourceType = ? AND rf.ReferenceUUID = ? AND rf.ReferenceType = ?))",
			wantErr: nil,
		},
	}

	for _, test := range tests {
		sqlBuilder, gotErr := FilterOnResourceReference(test.in.table, test.in.resourceType, test.in.count, test.in.filter)
		gotSql, _, err := sqlBuilder.ToSql()
		assert.Nil(t, err)

		if gotSql != test.wantSql || gotErr != test.wantErr {
			t.Errorf("FilterOnResourceReference(%+v) =\nGot: %q, %v\nWant: %q, %v",
				test.in, gotSql, gotErr, test.wantSql, test.wantErr)
		}
	}
}
