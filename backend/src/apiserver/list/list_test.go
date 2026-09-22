// Copyright 2018 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package list

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	api "github.com/kubeflow/pipelines/backend/api/v2beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/testing/protocmp"
)

type fakeListable struct {
	PrimaryKey       string
	FakeName         string
	CreatedTimestamp int64
}

func (f *fakeListable) PrimaryKeyColumnName() string {
	return "PrimaryKey"
}

func (f *fakeListable) DefaultSortField() string {
	return "CreatedTimestamp"
}

func testQuote(s string) string { return `"` + s + `"` }

var fakeAPIToModelMap = map[string]string{
	"timestamp": "CreatedTimestamp",
	"name":      "FakeName",
	"id":        "PrimaryKey",
}

func (f *fakeListable) APIToModelFieldMap() map[string]string {
	return fakeAPIToModelMap
}

func (f *fakeListable) GetModelName() string {
	return ""
}

func (f *fakeListable) GetField(name string) (string, string, bool) {
	if field, ok := fakeAPIToModelMap[name]; ok {
		return field, field, true
	}

	return "", "", false
}

func (f *fakeListable) GetFieldValue(name string) interface{} {
	switch name {
	case "CreatedTimestamp":
		return f.CreatedTimestamp
	case "FakeName":
		return f.FakeName
	case "PrimaryKey":
		return f.PrimaryKey
	}

	return nil
}

func (f *fakeListable) GetSortByFieldPrefix(name string) string {
	return ""
}

func (f *fakeListable) GetKeyFieldPrefix() string {
	return ""
}

func (f *fakeListable) CaseInsensitiveFields() map[string]struct{} {
	return map[string]struct{}{"name": {}}
}

func TestNextPageToken_ValidTokens(t *testing.T) {
	l := &fakeListable{PrimaryKey: "uuid123", FakeName: "Fake", CreatedTimestamp: 1234}

	protoFilter := &api.Filter{Predicates: []*api.Predicate{
		{
			Key:       "name",
			Operation: api.Predicate_EQUALS,
			Value:     &api.Predicate_StringValue{StringValue: "SomeName"},
		},
	}}
	testFilter, err := filter.New(protoFilter)
	if err != nil {
		t.Fatalf("failed to parse filter proto %+v: %v", protoFilter, err)
	}

	tests := []struct {
		inOpts *Options
		want   *token
	}{
		{
			inOpts: &Options{
				PageSize: 10, token: &token{SortByFieldName: "CreatedTimestamp", IsDesc: true},
			},
			want: &token{
				SortByFieldName:   "CreatedTimestamp",
				SortByFieldValue:  int64(1234),
				SortByFieldPrefix: "",
				KeyFieldName:      "PrimaryKey",
				KeyFieldValue:     "uuid123",
				KeyFieldPrefix:    "",
				IsDesc:            true,
			},
		},
		{
			inOpts: &Options{
				PageSize: 10, token: &token{SortByFieldName: "PrimaryKey", IsDesc: true},
			},
			want: &token{
				SortByFieldName:   "PrimaryKey",
				SortByFieldValue:  "uuid123",
				SortByFieldPrefix: "",
				KeyFieldName:      "PrimaryKey",
				KeyFieldValue:     "uuid123",
				KeyFieldPrefix:    "",
				IsDesc:            true,
			},
		},
		{
			inOpts: &Options{
				PageSize: 10, token: &token{SortByFieldName: "FakeName", IsDesc: false},
			},
			want: &token{
				SortByFieldName:   "FakeName",
				SortByFieldValue:  "Fake",
				SortByFieldPrefix: "",
				KeyFieldName:      "PrimaryKey",
				KeyFieldValue:     "uuid123",
				KeyFieldPrefix:    "",
				IsDesc:            false,
			},
		},
		{
			inOpts: &Options{
				PageSize: 10,
				token: &token{
					SortByFieldName: "FakeName", IsDesc: false,
					Filter: testFilter,
				},
			},
			want: &token{
				SortByFieldName:   "FakeName",
				SortByFieldValue:  "Fake",
				SortByFieldPrefix: "",
				KeyFieldName:      "PrimaryKey",
				KeyFieldValue:     "uuid123",
				KeyFieldPrefix:    "",
				IsDesc:            false,
				Filter:            testFilter,
			},
		},
	}

	for _, test := range tests {
		got, err := test.inOpts.nextPageToken(l)

		if !cmp.Equal(got, test.want, cmpopts.EquateEmpty(), protocmp.Transform(), cmp.AllowUnexported(filter.Filter{})) || err != nil {
			t.Errorf("nextPageToken(%+v, %+v) =\nGot: %+v, %+v\nWant: %+v, <nil>\nDiff:\n%s",
				test.inOpts, l, got, err, test.want, cmp.Diff(test.want, got))
		}
	}
}

func TestNextPageToken_InvalidSortByField(t *testing.T) {
	l := &fakeListable{PrimaryKey: "uuid123", FakeName: "Fake", CreatedTimestamp: 1234}

	inOpts := &Options{
		PageSize: 10, token: &token{SortByFieldName: "Timestamp", IsDesc: true},
	}
	want := util.NewInvalidInputError(`cannot sort by field "Timestamp" on type "fakeListable"`)

	got, err := inOpts.nextPageToken(l)

	if !cmp.Equal(err, want, cmpopts.IgnoreUnexported(util.UserError{})) {
		t.Errorf("nextPageToken(%+v, %+v) =\nGot: %+v, %v\nWant: _, %v",
			inOpts, l, got, err, want)
	}
}

// A nullable column is exposed as a pointer. A nil pointer is a NULL cursor, and
// a set pointer carries its plain value.
func TestNextPageToken_NullablePointerField(t *testing.T) {
	parentTaskID := "parent-task"
	numberValue := 0.5

	tests := []struct {
		name      string
		row       Listable
		sortBy    string
		wantValue interface{}
		wantNull  bool
	}{
		{"task without parent", &model.Task{UUID: "row-1"}, "parent_task_id", nil, true},
		{"task with parent", &model.Task{UUID: "row-1", ParentTaskUUID: &parentTaskID}, "parent_task_id", "parent-task", false},
		{"artifact without number", &model.Artifact{UUID: "row-1"}, "number_value", nil, true},
		{"artifact with number", &model.Artifact{UUID: "row-1", NumberValue: &numberValue}, "number_value", 0.5, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := NewOptions(tt.row, 2, tt.sortBy, nil)
			if err != nil {
				t.Fatalf("NewOptions() unexpected error: %v", err)
			}
			got, err := opts.nextPageToken(tt.row)
			if err != nil {
				t.Fatalf("nextPageToken() unexpected error: %v", err)
			}
			assert.Equal(t, tt.wantNull, got.SortByFieldIsNull)
			assert.Equal(t, tt.wantValue, got.SortByFieldValue)
			assert.Equal(t, "row-1", got.KeyFieldValue)
		})
	}
}

// The cursor built from a nullable string column has to survive the page token.
// A NULL cursor then continues inside the NULL block from the saved key.
func TestNextPageToken_NullableStringFieldRoundTrip(t *testing.T) {
	parentTaskID := "parent-task"

	tests := []struct {
		name     string
		sortBy   string
		row      *model.Task
		wantNull bool
		wantSQL  string
	}{
		{
			name:     "no parent ascending",
			sortBy:   "parent_task_id",
			row:      &model.Task{UUID: "row-1"},
			wantNull: true,
			wantSQL:  `("tasks"."ParentTaskUUID" IS NULL AND "tasks"."UUID" >= ?)`,
		},
		{
			name:     "no parent descending",
			sortBy:   "parent_task_id desc",
			row:      &model.Task{UUID: "row-1"},
			wantNull: true,
			wantSQL:  `("tasks"."ParentTaskUUID" IS NULL AND "tasks"."UUID" <= ?)`,
		},
		{
			name:    "with parent ascending",
			sortBy:  "parent_task_id",
			row:     &model.Task{UUID: "row-1", ParentTaskUUID: &parentTaskID},
			wantSQL: `LOWER("tasks"."ParentTaskUUID") > LOWER(?)`,
		},
		{
			name:    "with parent descending",
			sortBy:  "parent_task_id desc",
			row:     &model.Task{UUID: "row-1", ParentTaskUUID: &parentTaskID},
			wantSQL: `LOWER("tasks"."ParentTaskUUID") < LOWER(?)`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts, err := NewOptions(&model.Task{}, 2, tt.sortBy, nil)
			if err != nil {
				t.Fatalf("NewOptions() unexpected error: %v", err)
			}
			// Every page, including the first, has to order the column the same way.
			assert.True(t, opts.SortByFieldIsString)

			pageToken, err := opts.NextPageToken(tt.row)
			if err != nil {
				t.Fatalf("NextPageToken() unexpected error: %v", err)
			}
			next, err := NewOptionsFromToken(pageToken, 2)
			if err != nil {
				t.Fatalf("NewOptionsFromToken() unexpected error: %v", err)
			}
			assert.Equal(t, tt.wantNull, next.SortByFieldIsNull)
			assert.True(t, next.SortByFieldIsString)
			assert.Equal(t, "row-1", next.KeyFieldValue)

			sql, args, err := next.AddSortingToSelect(sq.Select("*").From("tasks"), testQuote, "").ToSql()
			if err != nil {
				t.Fatalf("AddSortingToSelect() unexpected error: %v", err)
			}
			assert.Contains(t, sql, tt.wantSQL)
			assert.Contains(t, sql, `ORDER BY ("tasks"."ParentTaskUUID" IS NULL) ASC, LOWER("tasks"."ParentTaskUUID")`)
			assert.Contains(t, args, "row-1")
		})
	}
}

func TestValidatePageSize(t *testing.T) {
	tests := []struct {
		in   int
		want int
	}{
		{0, defaultPageSize},
		{100, 100},
		{200, 200},
		{300, maxPageSize},
	}

	for _, test := range tests {
		got, err := validatePageSize(test.in)

		if got != test.want || err != nil {
			t.Errorf("validatePageSize(%d) = %d, %v\nWant: %d, <nil>", test.in, got, err, test.want)
		}
	}

	got, err := validatePageSize(-1)
	if err == nil {
		t.Errorf("validatePageSize(-1) = %d, <nil>\nWant: _, error", got)
	}
}

func TestNewOptions_FromValidSerializedToken(t *testing.T) {
	tok := &token{
		SortByFieldName:   "SortField",
		SortByFieldValue:  "string_field_value",
		SortByFieldPrefix: "",
		KeyFieldName:      "KeyField",
		KeyFieldValue:     "string_key_value",
		KeyFieldPrefix:    "",
		IsDesc:            true,
	}

	s, err := tok.marshal()
	if err != nil {
		t.Fatalf("failed to marshal token %+v: %v", tok, err)
	}

	want := &Options{PageSize: 123, token: tok}
	got, err := NewOptionsFromToken(s, 123)

	opt := cmp.AllowUnexported(Options{})
	if !cmp.Equal(got, want, opt) || err != nil {
		t.Errorf("NewOptionsFromToken(%q, 123) =\nGot: %+v, %v\nWant: %+v, nil\nDiff:\n%s",
			s, got, err, want, cmp.Diff(want, got, opt))
	}
}

func TestNewOptionsFromToken_FromInValidSerializedToken(t *testing.T) {
	tests := []struct{ in string }{{"random nonsense"}, {""}}

	for _, test := range tests {
		got, err := NewOptionsFromToken(test.in, 123)
		if err == nil {
			t.Errorf("NewOptionsFromToken(%q, 123) =\nGot: %+v, <nil>\nWant: _, error",
				test.in, got)
		}
	}
}

func TestNewOptionsFromToken_MaliciousFilterKey(t *testing.T) {
	// Simulate a forged pageToken with a malicious filter key containing SQL injection.
	// The filter key bypasses NewWithKeyMap's allowlist and reaches SQL construction directly.
	tests := []struct {
		name      string
		filterKey string
	}{
		{"sql injection in EQ key", `pipelines.Name) OR 1=1 --`},
		{"semicolon injection", `Name; DROP TABLE pipelines--`},
		{"unqualified injection", `Name) OR 1=1`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			raw := fmt.Sprintf(`{"KeyFieldName":"ID","SortByFieldName":"Name","Filter":{"EQ":{%q:[]}}}`, test.filterKey)
			token := base64.StdEncoding.EncodeToString([]byte(raw))
			got, err := NewOptionsFromToken(token, 10)
			if err == nil {
				t.Errorf("NewOptionsFromToken with malicious filter key %q =\nGot: %+v, <nil>\nWant: _, error",
					test.filterKey, got)
			}
		})
	}
}

func TestNewOptionsFromToken_FromInValidPageSize(t *testing.T) {
	tok := &token{
		SortByFieldName:   "SortField",
		SortByFieldValue:  "string_field_value",
		SortByFieldPrefix: "",
		KeyFieldName:      "KeyField",
		KeyFieldValue:     "string_key_value",
		KeyFieldPrefix:    "",
		IsDesc:            true,
	}

	s, err := tok.marshal()
	if err != nil {
		t.Fatalf("failed to marshal token %+v: %v", tok, err)
	}
	got, err := NewOptionsFromToken(s, -1)

	if err == nil {
		t.Errorf("NewOptionsFromToken(%q, 123) =\nGot: %+v, <nil>\nWant: _, error",
			s, got)
	}
}

func TestNewOptions_ValidSortOptions(t *testing.T) {
	pageSize := 10
	tests := []struct {
		sortBy string
		want   *Options
	}{
		{
			sortBy: "", // default sorting
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:        "PrimaryKey",
					KeyFieldPrefix:      "",
					SortByFieldName:     "CreatedTimestamp",
					SortBySQLColumn:     "CreatedTimestamp",
					SortByFieldPrefix:   "",
					SortByFieldIsString: false,
					IsDesc:              false,
				},
			},
		},
		{
			sortBy: "timestamp",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:        "PrimaryKey",
					KeyFieldPrefix:      "",
					SortByFieldName:     "CreatedTimestamp",
					SortBySQLColumn:     "CreatedTimestamp",
					SortByFieldPrefix:   "",
					SortByFieldIsString: false,
					IsDesc:              false,
				},
			},
		},
		{
			sortBy: "name",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:        "PrimaryKey",
					KeyFieldPrefix:      "",
					SortByFieldName:     "FakeName",
					SortBySQLColumn:     "FakeName",
					SortByFieldPrefix:   "",
					SortByFieldIsString: true,
					IsDesc:              false,
				},
			},
		},
		{
			sortBy: "name asc",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:        "PrimaryKey",
					KeyFieldPrefix:      "",
					SortByFieldName:     "FakeName",
					SortBySQLColumn:     "FakeName",
					SortByFieldPrefix:   "",
					SortByFieldIsString: true,
					IsDesc:              false,
				},
			},
		},
		{
			sortBy: "name desc",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:        "PrimaryKey",
					KeyFieldPrefix:      "",
					SortByFieldName:     "FakeName",
					SortBySQLColumn:     "FakeName",
					SortByFieldPrefix:   "",
					SortByFieldIsString: true,
					IsDesc:              true,
				},
			},
		},
		{
			sortBy: "id desc",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:        "PrimaryKey",
					KeyFieldPrefix:      "",
					SortByFieldName:     "PrimaryKey",
					SortBySQLColumn:     "PrimaryKey",
					SortByFieldPrefix:   "",
					SortByFieldIsString: true,
					IsDesc:              true,
				},
			},
		},
	}

	for _, test := range tests {
		got, err := NewOptions(&fakeListable{}, pageSize, test.sortBy, nil)

		opt := cmp.AllowUnexported(Options{})
		if !cmp.Equal(got, test.want, opt) || err != nil {
			t.Errorf("NewOptions(sortBy=%q) =\nGot: %+v, %v\nWant: %+v, nil\nDiff:\n%s",
				test.sortBy, got, err, test.want, cmp.Diff(got, test.want, opt))
		}
	}
}

func TestNewOptions_InvalidSortOptions(t *testing.T) {
	pageSize := 10
	tests := []struct {
		sortBy string
	}{
		{"unknownfield"},
		{"timestamp descending"},
		{"timestamp asc hello"},
	}

	for _, test := range tests {
		got, err := NewOptions(&fakeListable{}, pageSize, test.sortBy, nil)
		if err == nil {
			t.Errorf("NewOptions(sortBy=%q) =\nGot: %+v, <nil>\nWant error", test.sortBy, got)
		}
	}
}

func TestNewOptions_InvalidPageSize(t *testing.T) {
	got, err := NewOptions(&fakeListable{}, -1, "", nil)
	if err == nil {
		t.Errorf("NewOptions(pageSize=-1) =\nGot: %+v, <nil>\nWant error", got)
	}
}

func TestNewOptions_ValidFilter(t *testing.T) {
	protoFilter := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:       "name",
				Operation: api.Predicate_EQUALS,
				Value:     &api.Predicate_StringValue{StringValue: "SomeName"},
			},
		},
	}
	newFilter, _ := filter.New(protoFilter)

	got, err := NewOptions(&fakeListable{}, 10, "timestamp", newFilter)
	if err != nil {
		t.Fatalf("NewOptions: %v", err)
	}

	assert.Equal(t, 10, got.PageSize)
	assert.Equal(t, "PrimaryKey", got.KeyFieldName)
	assert.Equal(t, "CreatedTimestamp", got.SortByFieldName)
	assert.Equal(t, "CreatedTimestamp", got.SortBySQLColumn)
	assert.False(t, got.IsDesc)
	assert.NotNil(t, got.Filter)
}

func TestNewOptions_InvalidFilter(t *testing.T) {
	protoFilter := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:       "unknownfield",
				Operation: api.Predicate_EQUALS,
				Value:     &api.Predicate_StringValue{StringValue: "SomeName"},
			},
		},
	}
	newFilter, _ := filter.New(protoFilter)

	got, err := NewOptions(&fakeListable{}, 10, "timestamp", newFilter)
	if err == nil {
		t.Errorf("NewOptions(protoFilter=%+v) =\nGot: %+v, <nil>\nWant error", protoFilter, got)
	}
}

func TestNewOptions_ModelFilter(t *testing.T) {
	protoFilter := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:       "finished_at",
				Operation: api.Predicate_GREATER_THAN,
				Value:     &api.Predicate_StringValue{StringValue: "SomeTime"},
			},
		},
	}
	newFilter, _ := filter.New(protoFilter)

	got, err := NewOptions(&model.Run{}, 10, "name", newFilter)
	if err != nil {
		t.Fatalf("NewOptions: %v", err)
	}

	assert.Equal(t, 10, got.PageSize)
	assert.Equal(t, "UUID", got.KeyFieldName)
	assert.Equal(t, "DisplayName", got.SortByFieldName)
	assert.Equal(t, "DisplayName", got.SortBySQLColumn)
	assert.True(t, got.SortByFieldIsString)
	assert.False(t, got.IsDesc)
	assert.NotNil(t, got.Filter)
}

func TestAddPaginationAndFilterToSelect(t *testing.T) {
	protoFilter := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:       "Name",
				Operation: api.Predicate_EQUALS,
				Value:     &api.Predicate_StringValue{StringValue: "SomeName"},
			},
		},
	}
	f, err := filter.New(protoFilter)
	if err != nil {
		t.Fatalf("failed to parse filter proto %+v: %v", protoFilter, err)
	}

	tests := []struct {
		in       *Options
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:   "SortField",
					SortBySQLColumn:   "SortField",
					SortByFieldValue:  "value",
					SortByFieldPrefix: "",
					KeyFieldName:      "KeyField",
					KeyFieldValue:     1111,
					KeyFieldPrefix:    "",
					IsDesc:            true,
				},
			},
			wantSQL:  `SELECT * FROM MyTable WHERE (LOWER("SortField") < LOWER(?) OR (LOWER("SortField") = LOWER(?) AND "KeyField" <= ?) OR "SortField" IS NULL) ORDER BY ("SortField" IS NULL) ASC, LOWER("SortField") DESC, "KeyField" DESC LIMIT 124`,
			wantArgs: []interface{}{"value", "value", 1111},
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:   "SortField",
					SortBySQLColumn:   "SortField",
					SortByFieldValue:  "value",
					SortByFieldPrefix: "",
					KeyFieldName:      "KeyField",
					KeyFieldValue:     1111,
					KeyFieldPrefix:    "",
					IsDesc:            false,
				},
			},
			wantSQL:  `SELECT * FROM MyTable WHERE (LOWER("SortField") > LOWER(?) OR (LOWER("SortField") = LOWER(?) AND "KeyField" >= ?) OR "SortField" IS NULL) ORDER BY ("SortField" IS NULL) ASC, LOWER("SortField") ASC, "KeyField" ASC LIMIT 124`,
			wantArgs: []interface{}{"value", "value", 1111},
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:   "SortField",
					SortBySQLColumn:   "SortField",
					SortByFieldValue:  "value",
					SortByFieldPrefix: "",
					KeyFieldName:      "KeyField",
					KeyFieldValue:     1111,
					KeyFieldPrefix:    "",
					IsDesc:            false,
					Filter:            f,
				},
			},
			wantSQL:  `SELECT * FROM MyTable WHERE (LOWER("SortField") > LOWER(?) OR (LOWER("SortField") = LOWER(?) AND "KeyField" >= ?) OR "SortField" IS NULL) AND ("Name" = ?) ORDER BY ("SortField" IS NULL) ASC, LOWER("SortField") ASC, "KeyField" ASC LIMIT 124`,
			wantArgs: []interface{}{"value", "value", 1111, "SomeName"},
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:     "SortField",
					SortBySQLColumn:     "SortField",
					SortByFieldIsString: true,
					SortByFieldPrefix:   "",
					KeyFieldName:        "KeyField",
					KeyFieldPrefix:      "",
					KeyFieldValue:       1111,
					IsDesc:              true,
				},
			},
			wantSQL:  `SELECT * FROM MyTable ORDER BY ("SortField" IS NULL) ASC, LOWER("SortField") DESC, "KeyField" DESC LIMIT 124`,
			wantArgs: nil,
		},
		{
			in:       EmptyOptions(),
			wantSQL:  fmt.Sprintf("SELECT * FROM MyTable LIMIT %d", math.MaxInt32+1),
			wantArgs: nil,
		},
		// Numeric field, first page (SortByFieldValue == nil): should NOT use LOWER().
		// This is the regression test for PostgreSQL "function lower(bigint) does not exist".
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:     "CreatedAtInSec",
					SortBySQLColumn:     "CreatedAtInSec",
					SortByFieldIsString: false,
					SortByFieldPrefix:   "",
					KeyFieldName:        "KeyField",
					KeyFieldPrefix:      "",
					IsDesc:              false,
				},
			},
			wantSQL:  `SELECT * FROM MyTable ORDER BY ("CreatedAtInSec" IS NULL) ASC, "CreatedAtInSec" ASC, "KeyField" ASC LIMIT 124`,
			wantArgs: nil,
		},
		// Numeric field, second page (SortByFieldValue is float64, e.g. CreatedAtInSec):
		// WHERE clause should NOT use LOWER().
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:     "CreatedAtInSec",
					SortBySQLColumn:     "CreatedAtInSec",
					SortByFieldIsString: false,
					SortByFieldValue:    float64(1234567890),
					SortByFieldPrefix:   "",
					KeyFieldName:        "KeyField",
					KeyFieldValue:       "uuid-2",
					KeyFieldPrefix:      "",
					IsDesc:              false,
				},
			},
			wantSQL:  `SELECT * FROM MyTable WHERE ("CreatedAtInSec" > ? OR ("CreatedAtInSec" = ? AND "KeyField" >= ?) OR "CreatedAtInSec" IS NULL) ORDER BY ("CreatedAtInSec" IS NULL) ASC, "CreatedAtInSec" ASC, "KeyField" ASC LIMIT 124`,
			wantArgs: []interface{}{float64(1234567890), float64(1234567890), "uuid-2"},
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:   "SortField",
					SortBySQLColumn:   "SortField",
					SortByFieldValue:  "value",
					SortByFieldPrefix: "",
					KeyFieldName:      "KeyField",
					KeyFieldPrefix:    "",
					IsDesc:            false,
				},
			},
			wantSQL:  `SELECT * FROM MyTable ORDER BY ("SortField" IS NULL) ASC, LOWER("SortField") ASC, "KeyField" ASC LIMIT 124`,
			wantArgs: nil,
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:   "SortField",
					SortBySQLColumn:   "SortField",
					SortByFieldValue:  "value",
					SortByFieldPrefix: "",
					KeyFieldName:      "KeyField",
					KeyFieldPrefix:    "",
					IsDesc:            false,
					Filter:            f,
				},
			},
			wantSQL:  `SELECT * FROM MyTable WHERE ("Name" = ?) ORDER BY ("SortField" IS NULL) ASC, LOWER("SortField") ASC, "KeyField" ASC LIMIT 124`,
			wantArgs: []interface{}{"SomeName"},
		},
		// Numeric field, second page (SortByFieldValue is float64): bind parameter preserves full precision.
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:     "MetricValue",
					SortBySQLColumn:     "MetricValue",
					SortByFieldIsString: false,
					SortByFieldValue:    float64(0.123456789),
					SortByFieldPrefix:   "",
					KeyFieldName:        "KeyField",
					KeyFieldValue:       "uuid-1",
					KeyFieldPrefix:      "",
					IsDesc:              false,
				},
			},
			wantSQL:  `SELECT * FROM MyTable WHERE ("MetricValue" > ? OR ("MetricValue" = ? AND "KeyField" >= ?) OR "MetricValue" IS NULL) ORDER BY ("MetricValue" IS NULL) ASC, "MetricValue" ASC, "KeyField" ASC LIMIT 124`,
			wantArgs: []interface{}{float64(0.123456789), float64(0.123456789), "uuid-1"},
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:     "MetricValue",
					SortBySQLColumn:     "MetricValue",
					SortByFieldIsString: false,
					SortByFieldValue:    float64(0.123456789),
					SortByFieldPrefix:   "",
					KeyFieldName:        "KeyField",
					KeyFieldValue:       "uuid-1",
					KeyFieldPrefix:      "",
					IsDesc:              true,
				},
			},
			wantSQL:  `SELECT * FROM MyTable WHERE ("MetricValue" < ? OR ("MetricValue" = ? AND "KeyField" <= ?) OR "MetricValue" IS NULL) ORDER BY ("MetricValue" IS NULL) ASC, "MetricValue" DESC, "KeyField" DESC LIMIT 124`,
			wantArgs: []interface{}{float64(0.123456789), float64(0.123456789), "uuid-1"},
		},
		// Non-metric nullable string field, DESC with cursor: NULL handling in both
		// WHERE and ORDER BY, ensuring cross-dialect consistency for regular fields.
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:     "FakeName",
					SortBySQLColumn:     "FakeName",
					SortByFieldIsString: true,
					SortByFieldValue:    "some_value",
					SortByFieldPrefix:   "",
					KeyFieldName:        "KeyField",
					KeyFieldValue:       "uuid-3",
					KeyFieldPrefix:      "",
					IsDesc:              true,
				},
			},
			wantSQL:  `SELECT * FROM MyTable WHERE (LOWER("FakeName") < LOWER(?) OR (LOWER("FakeName") = LOWER(?) AND "KeyField" <= ?) OR "FakeName" IS NULL) ORDER BY ("FakeName" IS NULL) ASC, LOWER("FakeName") DESC, "KeyField" DESC LIMIT 124`,
			wantArgs: []interface{}{"some_value", "some_value", "uuid-3"},
		},
		// Non-metric nullable string field, ASC with cursor.
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:     "FakeName",
					SortBySQLColumn:     "FakeName",
					SortByFieldIsString: true,
					SortByFieldValue:    "some_value",
					SortByFieldPrefix:   "",
					KeyFieldName:        "KeyField",
					KeyFieldValue:       "uuid-3",
					KeyFieldPrefix:      "",
					IsDesc:              false,
				},
			},
			wantSQL:  `SELECT * FROM MyTable WHERE (LOWER("FakeName") > LOWER(?) OR (LOWER("FakeName") = LOWER(?) AND "KeyField" >= ?) OR "FakeName" IS NULL) ORDER BY ("FakeName" IS NULL) ASC, LOWER("FakeName") ASC, "KeyField" ASC LIMIT 124`,
			wantArgs: []interface{}{"some_value", "some_value", "uuid-3"},
		},
		// Non-metric nullable numeric field, DESC with cursor.
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:     "CreatedAtInSec",
					SortBySQLColumn:     "CreatedAtInSec",
					SortByFieldIsString: false,
					SortByFieldValue:    float64(1000),
					SortByFieldPrefix:   "",
					KeyFieldName:        "KeyField",
					KeyFieldValue:       "uuid-4",
					KeyFieldPrefix:      "",
					IsDesc:              true,
				},
			},
			wantSQL:  `SELECT * FROM MyTable WHERE ("CreatedAtInSec" < ? OR ("CreatedAtInSec" = ? AND "KeyField" <= ?) OR "CreatedAtInSec" IS NULL) ORDER BY ("CreatedAtInSec" IS NULL) ASC, "CreatedAtInSec" DESC, "KeyField" DESC LIMIT 124`,
			wantArgs: []interface{}{float64(1000), float64(1000), "uuid-4"},
		},
		// Non-metric nullable numeric field, ASC with cursor.
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:     "CreatedAtInSec",
					SortBySQLColumn:     "CreatedAtInSec",
					SortByFieldIsString: false,
					SortByFieldValue:    float64(1000),
					SortByFieldPrefix:   "",
					KeyFieldName:        "KeyField",
					KeyFieldValue:       "uuid-4",
					KeyFieldPrefix:      "",
					IsDesc:              false,
				},
			},
			wantSQL:  `SELECT * FROM MyTable WHERE ("CreatedAtInSec" > ? OR ("CreatedAtInSec" = ? AND "KeyField" >= ?) OR "CreatedAtInSec" IS NULL) ORDER BY ("CreatedAtInSec" IS NULL) ASC, "CreatedAtInSec" ASC, "KeyField" ASC LIMIT 124`,
			wantArgs: []interface{}{float64(1000), float64(1000), "uuid-4"},
		},
		// Metric sort, non-NULL cursor, ASC (case A): NULL rows sort last, so the
		// cursor also pulls in the trailing NULL block via "sort_metric_value IS NULL".
		// The ORDER BY gains a leading "(col IS NULL) ASC" key for deterministic NULL-last.

		// Metric sort, non-NULL cursor, DESC (case A).

		// Metric sort, NULL cursor, ASC (case B): all non-NULL rows are already paged
		// through; advance within the trailing NULL block using the key alone.

		// Metric sort, NULL cursor, DESC (case B): key tie-break flips to <=.

	}

	for _, test := range tests {
		sql := sq.Select("*").From("MyTable")
		gotSQL, gotArgs, err := test.in.AddFilterToSelect(test.in.AddPaginationToSelect(sql, testQuote, ""), testQuote).ToSql()

		if gotSQL != test.wantSQL || !reflect.DeepEqual(gotArgs, test.wantArgs) || err != nil {
			t.Errorf("BuildListSQLQuery(%+v) =\nGot: %q, %v, %v\nWant: %q, %v, nil",
				test.in, gotSQL, gotArgs, err, test.wantSQL, test.wantArgs)
		}
	}
}

func TestTokenSerialization(t *testing.T) {
	protoFilter := &api.Filter{Predicates: []*api.Predicate{
		{
			Key:       "name",
			Operation: api.Predicate_EQUALS,
			Value:     &api.Predicate_StringValue{StringValue: "SomeName"},
		},
	}}
	testFilter, err := filter.New(protoFilter)
	if err != nil {
		t.Fatalf("failed to parse filter proto %+v: %v", protoFilter, err)
	}

	tests := []struct {
		in   *token
		want *token
	}{
		// string values in sort by fields
		{
			in: &token{
				SortByFieldName:   "SortField",
				SortByFieldValue:  "string_field_value",
				SortByFieldPrefix: "",
				KeyFieldName:      "KeyField",
				KeyFieldValue:     "string_key_value",
				KeyFieldPrefix:    "",
				IsDesc:            true,
			},
			want: &token{
				SortByFieldName:   "SortField",
				SortByFieldValue:  "string_field_value",
				SortByFieldPrefix: "",
				KeyFieldName:      "KeyField",
				KeyFieldValue:     "string_key_value",
				KeyFieldPrefix:    "",
				IsDesc:            true,
			},
		},
		// int values get deserialized as floats by JSON unmarshal.
		{
			in: &token{
				SortByFieldName:   "SortField",
				SortByFieldValue:  100,
				SortByFieldPrefix: "",
				KeyFieldName:      "KeyField",
				KeyFieldValue:     200,
				KeyFieldPrefix:    "",
				IsDesc:            true,
			},
			want: &token{
				SortByFieldName:   "SortField",
				SortByFieldValue:  float64(100),
				SortByFieldPrefix: "",
				KeyFieldName:      "KeyField",
				KeyFieldValue:     float64(200),
				KeyFieldPrefix:    "",
				IsDesc:            true,
			},
		},
		// has a filter.
		{
			in: &token{
				SortByFieldName:   "SortField",
				SortByFieldValue:  100,
				SortByFieldPrefix: "",
				KeyFieldName:      "KeyField",
				KeyFieldValue:     200,
				KeyFieldPrefix:    "",
				IsDesc:            true,
				Filter:            testFilter,
			},
			want: &token{
				SortByFieldName:   "SortField",
				SortByFieldValue:  float64(100),
				SortByFieldPrefix: "",
				KeyFieldName:      "KeyField",
				KeyFieldValue:     float64(200),
				KeyFieldPrefix:    "",
				IsDesc:            true,
				Filter:            testFilter,
			},
		},
	}

	for _, test := range tests {
		s, err := test.in.marshal()
		if err != nil {
			t.Errorf("Token.Marshal(%+v) = _, %v\nWant nil error", test.in, err)
			continue
		}

		got := &token{}
		got.unmarshal(s)
		if !cmp.Equal(got, test.want, cmpopts.EquateEmpty(), protocmp.Transform(), cmp.AllowUnexported(filter.Filter{})) {
			t.Errorf("token.unmarshal(%q) =\nGot: %+v\nWant: %+v\nDiff:\n%s",
				s, got, test.want, cmp.Diff(test.want, got, cmp.AllowUnexported(filter.Filter{})))
		}
	}
}

func TestMatches(t *testing.T) {
	protoFilter1 := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:       "Name",
				Operation: api.Predicate_EQUALS,
				Value:     &api.Predicate_StringValue{StringValue: "SomeName"},
			},
		},
	}
	f1, err := filter.New(protoFilter1)
	if err != nil {
		t.Fatalf("failed to parse filter proto %+v: %v", protoFilter1, err)
	}

	protoFilter2 := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:       "Name",
				Operation: api.Predicate_NOT_EQUALS, // Not equals as opposed to equals above.
				Value:     &api.Predicate_StringValue{StringValue: "SomeName"},
			},
		},
	}
	f2, err := filter.New(protoFilter2)
	if err != nil {
		t.Fatalf("failed to parse filter proto %+v: %v", protoFilter2, err)
	}

	tests := []struct {
		o1   *Options
		o2   *Options
		want bool
	}{
		{
			o1:   &Options{token: &token{SortByFieldName: "SortField1", IsDesc: true}},
			o2:   &Options{token: &token{SortByFieldName: "SortField2", IsDesc: true}},
			want: false,
		},
		{
			o1:   &Options{token: &token{SortByFieldName: "SortField1", IsDesc: true}},
			o2:   &Options{token: &token{SortByFieldName: "SortField1", IsDesc: true}},
			want: true,
		},
		{
			o1:   &Options{token: &token{SortByFieldName: "SortField1", IsDesc: true}},
			o2:   &Options{token: &token{SortByFieldName: "SortField1", IsDesc: false}},
			want: false,
		},
		{
			o1:   &Options{token: &token{Filter: f1}},
			o2:   &Options{token: &token{Filter: f1}},
			want: true,
		},
		{
			o1:   &Options{token: &token{Filter: f1}},
			o2:   &Options{token: &token{Filter: f2}},
			want: false,
		},
		// Metric sort: SortByFieldName holds the raw metric name, so tokens for
		// different metrics are distinct queries even though they share the same
		// SQL alias in SortBySQLColumn.

		// Metric sort: same metric name is the same query.

	}

	for _, test := range tests {
		got := test.o1.Matches(test.o2)

		if got != test.want {
			t.Errorf("Matches(%+v, %+v) = %v, Want nil %v", test.o1, test.o2, got, test.want)
			continue
		}
	}
}

func TestAddSortingToSelectWithPipelineVersionModel(t *testing.T) {
	listable := &model.PipelineVersion{
		UUID:           "version_id_1",
		CreatedAtInSec: 1,
		Name:           "version_name_1",
		Parameters:     "",
		PipelineId:     "pipeline_id_1",
		Status:         model.PipelineVersionReady,
		CodeSourceUrl:  "",
	}
	protoFilter := &api.Filter{}
	newFilter, _ := filter.New(protoFilter)
	listableOptions, err := NewOptions(listable, 10, "name", newFilter)
	assert.Nil(t, err)
	sqlBuilder := sq.Select("*").From("pipeline_versions")
	sql, _, err := listableOptions.AddSortingToSelect(sqlBuilder, testQuote, "").ToSql()
	assert.Nil(t, err)

	assert.Contains(t, sql, `"pipeline_versions"."Name"`) // sorting field
	assert.Contains(t, sql, `"pipeline_versions"."UUID"`) // primary key field
}

func TestAddStatusFilterToSelectWithRunModel(t *testing.T) {
	listable := &model.Run{
		UUID:        "run_id_1",
		DisplayName: "run_name_1",
		RunDetails: model.RunDetails{
			CreatedAtInSec: 1,
			Conditions:     "Succeeded",
			State:          model.RuntimeStateSucceededV1,
		},
	}
	protoFilter := &api.Filter{}
	protoFilter.Predicates = []*api.Predicate{
		{
			Key:       "status",
			Operation: api.Predicate_EQUALS,
			Value:     &api.Predicate_StringValue{StringValue: "Succeeded"},
		},
	}
	newFilter, _ := filter.New(protoFilter)
	listableOptions, err := NewOptions(listable, 10, "name", newFilter)
	assert.Nil(t, err)
	sqlBuilder := sq.Select("*").From("run_details")
	sql, args, err := listableOptions.AddFilterToSelect(sqlBuilder, testQuote).ToSql()
	assert.Nil(t, err)
	assert.Contains(t, sql, `WHERE ("Conditions" = ?)`) // status is not case-insensitive; exact comparison
	assert.Contains(t, args, "Succeeded")

	notEqualProtoFilter := &api.Filter{}
	notEqualProtoFilter.Predicates = []*api.Predicate{
		{
			Key:       "status",
			Operation: api.Predicate_NOT_EQUALS,
			Value:     &api.Predicate_StringValue{StringValue: "somevalue"},
		},
	}
	newNotEqualFilter, _ := filter.New(notEqualProtoFilter)
	listableOptions, err = NewOptions(listable, 10, "name", newNotEqualFilter)
	assert.Nil(t, err)
	sqlBuilder = sq.Select("*").From("run_details")
	sql, args, err = listableOptions.AddFilterToSelect(sqlBuilder, testQuote).ToSql()
	assert.Nil(t, err)
	assert.Contains(t, sql, `WHERE ("Conditions" <> ?)`) // status is not case-insensitive; exact comparison
	assert.Contains(t, args, "somevalue")
}

// Sorting by a mapped field that GetFieldValue cannot resolve returns a first
// page, then fails the whole call once NextPageToken has to build a token.
func TestGetFieldValue_ResolvesEveryMappedField(t *testing.T) {
	parentTaskID := "parent-task"
	uri := "s3://bucket/artifact"
	numberValue := 0.5

	// Optional fields are set, so a nil value below means a missing getter and
	// not just an empty field.
	listables := []Listable{
		&model.Run{UUID: "run"},
		&model.Job{UUID: "job"},
		&model.Experiment{UUID: "experiment"},
		&model.Pipeline{UUID: "pipeline"},
		&model.PipelineVersion{UUID: "pipeline-version"},
		&model.Task{
			UUID:             "task",
			ParentTaskUUID:   &parentTaskID,
			StatusMetadata:   model.JSONData{"message": "done"},
			StateHistory:     model.JSONSlice{"SUCCEEDED"},
			InputParameters:  model.JSONSlice{"input"},
			OutputParameters: model.JSONSlice{"output"},
			TypeAttrs:        model.JSONData{"iteration_count": 1},
		},
		&model.Artifact{UUID: "artifact", URI: &uri, NumberValue: &numberValue, Metadata: model.JSONData{"key": "value"}},
		&model.ArtifactTask{UUID: "artifact-task", Producer: model.JSONData{"task_name": "producer"}},
	}

	// JSON columns with no getter. They could not carry a page cursor anyway.
	noGetter := map[string]bool{
		"Run.StateHistory": true,
		"Task.pods":        true,
	}

	for _, listable := range listables {
		modelName := reflect.TypeOf(listable).Elem().Name()
		for apiField, modelField := range listable.APIToModelFieldMap() {
			t.Run(modelName+"/"+apiField, func(t *testing.T) {
				value := listable.GetFieldValue(modelField)
				if noGetter[modelName+"."+modelField] {
					assert.Nil(t, value, "%s.GetFieldValue(%q) now resolves, so remove it from noGetter", modelName, modelField)
					return
				}
				require.NotNil(t, value, "%s.GetFieldValue(%q) returns nil, so sorting by %q cannot build a page token", modelName, modelField, apiField)

				// Only scalar values can be a sort cursor, so JSON values stop here.
				switch reflect.Indirect(reflect.ValueOf(value)).Kind() {
				case reflect.Map, reflect.Slice:
					return
				}

				opts, err := NewOptions(listable, 1, apiField, nil)
				require.NoError(t, err)
				pageToken, err := opts.NextPageToken(listable)
				require.NoError(t, err)
				next, err := NewOptionsFromToken(pageToken, 1)
				require.NoError(t, err)
				assert.Equal(t, decodedTokenValue(t, value), next.SortByFieldValue)
				assert.Equal(t, decodedTokenValue(t, reflect.ValueOf(listable).Elem().FieldByName("UUID").Interface()), next.KeyFieldValue)
			})
		}
	}
}

// decodedTokenValue returns v the way a page token decodes it, so integers come
// back as float64 and pointers as the value they point to.
func decodedTokenValue(t *testing.T, v interface{}) interface{} {
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var decoded interface{}
	require.NoError(t, json.Unmarshal(b, &decoded))
	return decoded
}

func TestNewOptionsFromToken_RejectsRetiredMetricSort(t *testing.T) {
	oldToken := &token{KeyFieldName: "UUID", SortByFieldName: "accuracy", SortBySQLColumn: "sort_metric_value"}
	encoded, err := oldToken.marshal()
	require.NoError(t, err)
	_, err = NewOptionsFromToken(encoded, 10)
	require.Error(t, err)
	require.Contains(t, err.Error(), "Metric sorting is no longer supported")
}
