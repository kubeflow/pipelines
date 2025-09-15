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
	"fmt"
	"math"
	"reflect"
	"strings"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	api "github.com/kubeflow/pipelines/backend/api/v1beta1/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/testing/protocmp"
)

type fakeMetric struct {
	Name  string
	Value float64
}

type fakeListable struct {
	PrimaryKey       string
	FakeName         string
	CreatedTimestamp int64
	Metrics          []*fakeMetric
}

func (f *fakeListable) PrimaryKeyColumnName() string {
	return "PrimaryKey"
}

func (f *fakeListable) DefaultSortField() string {
	return "CreatedTimestamp"
}

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
	if strings.HasPrefix(name, "metric:") {
		return name[7:], model.MetricSortSQLAlias, true
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
	for _, metric := range f.Metrics {
		if metric.Name == name {
			return metric.Value
		}
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
	l := &fakeListable{PrimaryKey: "uuid123", FakeName: "Fake", CreatedTimestamp: 1234, Metrics: []*fakeMetric{
		{
			Name:  "m1",
			Value: 1.0,
		},
		{
			Name:  "m2",
			Value: 2.0,
		},
	}}

	protoFilter := &api.Filter{Predicates: []*api.Predicate{
		{
			Key:   "name",
			Op:    api.Predicate_EQUALS,
			Value: &api.Predicate_StringValue{StringValue: "SomeName"},
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
		{
			inOpts: &Options{
				PageSize: 10,
				token: &token{
					SortByFieldName: "m1", IsDesc: false,
				},
			},
			want: &token{
				SortByFieldName:   "m1",
				SortByFieldValue:  1.0,
				SortByFieldPrefix: "",
				KeyFieldName:      "PrimaryKey",
				KeyFieldValue:     "uuid123",
				KeyFieldPrefix:    "",
				IsDesc:            false,
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

// TestNextPageToken_MetricValueNull covers the regression where a lookahead row
// without the selected metric produced a nil sort value and made an otherwise
// valid ListRuns request fail with "cannot sort by field". For metric sorts a
// missing metric is a legitimate SQL NULL: the token must be generated with
// SortByFieldIsNull=true rather than returning an error.
func TestNextPageToken_MetricValueNull(t *testing.T) {
	// Row has some metrics, but not the one we are sorting by ("accuracy").
	l := &fakeListable{
		PrimaryKey: "uuid123", FakeName: "Fake", CreatedTimestamp: 1234,
		Metrics: []*fakeMetric{{Name: "loss", Value: 0.1}},
	}

	inOpts := &Options{
		PageSize: 10,
		token: &token{
			SortByFieldName: "accuracy",
			SortBySQLColumn: model.MetricSortSQLAlias,
			KeyFieldName:    "PrimaryKey",
			IsDesc:          true,
		},
	}

	got, err := inOpts.nextPageToken(l)
	if err != nil {
		t.Fatalf("nextPageToken() unexpected error for missing metric: %v", err)
	}
	if !got.SortByFieldIsNull {
		t.Errorf("nextPageToken() SortByFieldIsNull = false, want true for missing metric")
	}
	if got.SortByFieldValue != nil {
		t.Errorf("nextPageToken() SortByFieldValue = %v, want nil for missing metric", got.SortByFieldValue)
	}
	if got.KeyFieldValue != "uuid123" {
		t.Errorf("nextPageToken() KeyFieldValue = %v, want %q", got.KeyFieldValue, "uuid123")
	}
}

// TestNextPageToken_MetricValuePresent is the complementary case: when the row
// does have the selected metric, SortByFieldIsNull must stay false and the value
// is carried in the token as usual.
func TestNextPageToken_MetricValuePresent(t *testing.T) {
	l := &fakeListable{
		PrimaryKey: "uuid123", FakeName: "Fake", CreatedTimestamp: 1234,
		Metrics: []*fakeMetric{{Name: "accuracy", Value: 0.95}},
	}

	inOpts := &Options{
		PageSize: 10,
		token: &token{
			SortByFieldName: "accuracy",
			SortBySQLColumn: model.MetricSortSQLAlias,
			KeyFieldName:    "PrimaryKey",
			IsDesc:          true,
		},
	}

	got, err := inOpts.nextPageToken(l)
	if err != nil {
		t.Fatalf("nextPageToken() unexpected error: %v", err)
	}
	if got.SortByFieldIsNull {
		t.Errorf("nextPageToken() SortByFieldIsNull = true, want false when metric present")
	}
	if got.SortByFieldValue != 0.95 {
		t.Errorf("nextPageToken() SortByFieldValue = %v, want 0.95", got.SortByFieldValue)
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

func TestNewOptions_ValidMetricSort(t *testing.T) {
	pageSize := 10
	tests := []struct {
		sortBy         string
		wantMetricName string
	}{
		{"metric:accuracy", "accuracy"},
		{"metric:log-loss", "log-loss"},
		{"metric:val_accuracy", "val_accuracy"},
	}

	for _, test := range tests {
		got, err := NewOptions(&fakeListable{}, pageSize, test.sortBy, nil)
		if err != nil {
			t.Errorf("NewOptions(sortBy=%q) returned unexpected error: %v", test.sortBy, err)
			continue
		}
		// Metric sorts carry the raw metric name in SortByFieldName and the fixed
		// SQL alias in SortBySQLColumn.
		if got.SortByFieldName != test.wantMetricName {
			t.Errorf("NewOptions(sortBy=%q) SortByFieldName = %q, want %q", test.sortBy, got.SortByFieldName, test.wantMetricName)
		}
		if got.SortBySQLColumn != model.MetricSortSQLAlias {
			t.Errorf("NewOptions(sortBy=%q) SortBySQLColumn = %q, want %q", test.sortBy, got.SortBySQLColumn, model.MetricSortSQLAlias)
		}
		if !got.IsMetricSort() {
			t.Errorf("NewOptions(sortBy=%q) IsMetricSort() = false, want true", test.sortBy)
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
				Key:   "name",
				Op:    api.Predicate_EQUALS,
				Value: &api.Predicate_StringValue{StringValue: "SomeName"},
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
				Key:   "unknownfield",
				Op:    api.Predicate_EQUALS,
				Value: &api.Predicate_StringValue{StringValue: "SomeName"},
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
				Key:   "finished_at",
				Op:    api.Predicate_GREATER_THAN,
				Value: &api.Predicate_StringValue{StringValue: "SomeTime"},
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
				Key:   "Name",
				Op:    api.Predicate_EQUALS,
				Value: &api.Predicate_StringValue{StringValue: "SomeName"},
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
			wantSQL:  "SELECT * FROM MyTable WHERE (LOWER(SortField) < LOWER(?) OR (LOWER(SortField) = LOWER(?) AND KeyField <= ?) OR SortField IS NULL) ORDER BY (SortField IS NULL) ASC, LOWER(SortField) DESC, KeyField DESC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable WHERE (LOWER(SortField) > LOWER(?) OR (LOWER(SortField) = LOWER(?) AND KeyField >= ?) OR SortField IS NULL) ORDER BY (SortField IS NULL) ASC, LOWER(SortField) ASC, KeyField ASC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable WHERE (LOWER(SortField) > LOWER(?) OR (LOWER(SortField) = LOWER(?) AND KeyField >= ?) OR SortField IS NULL) AND (Name = ?) ORDER BY (SortField IS NULL) ASC, LOWER(SortField) ASC, KeyField ASC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable ORDER BY (SortField IS NULL) ASC, LOWER(SortField) DESC, KeyField DESC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable ORDER BY (CreatedAtInSec IS NULL) ASC, CreatedAtInSec ASC, KeyField ASC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable WHERE (CreatedAtInSec > ? OR (CreatedAtInSec = ? AND KeyField >= ?) OR CreatedAtInSec IS NULL) ORDER BY (CreatedAtInSec IS NULL) ASC, CreatedAtInSec ASC, KeyField ASC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable ORDER BY (SortField IS NULL) ASC, LOWER(SortField) ASC, KeyField ASC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable WHERE (Name = ?) ORDER BY (SortField IS NULL) ASC, LOWER(SortField) ASC, KeyField ASC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable WHERE (MetricValue > ? OR (MetricValue = ? AND KeyField >= ?) OR MetricValue IS NULL) ORDER BY (MetricValue IS NULL) ASC, MetricValue ASC, KeyField ASC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable WHERE (MetricValue < ? OR (MetricValue = ? AND KeyField <= ?) OR MetricValue IS NULL) ORDER BY (MetricValue IS NULL) ASC, MetricValue DESC, KeyField DESC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable WHERE (LOWER(FakeName) < LOWER(?) OR (LOWER(FakeName) = LOWER(?) AND KeyField <= ?) OR FakeName IS NULL) ORDER BY (FakeName IS NULL) ASC, LOWER(FakeName) DESC, KeyField DESC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable WHERE (LOWER(FakeName) > LOWER(?) OR (LOWER(FakeName) = LOWER(?) AND KeyField >= ?) OR FakeName IS NULL) ORDER BY (FakeName IS NULL) ASC, LOWER(FakeName) ASC, KeyField ASC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable WHERE (CreatedAtInSec < ? OR (CreatedAtInSec = ? AND KeyField <= ?) OR CreatedAtInSec IS NULL) ORDER BY (CreatedAtInSec IS NULL) ASC, CreatedAtInSec DESC, KeyField DESC LIMIT 124",
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
			wantSQL:  "SELECT * FROM MyTable WHERE (CreatedAtInSec > ? OR (CreatedAtInSec = ? AND KeyField >= ?) OR CreatedAtInSec IS NULL) ORDER BY (CreatedAtInSec IS NULL) ASC, CreatedAtInSec ASC, KeyField ASC LIMIT 124",
			wantArgs: []interface{}{float64(1000), float64(1000), "uuid-4"},
		},
		// Metric sort, non-NULL cursor, ASC (case A): NULL rows sort last, so the
		// cursor also pulls in the trailing NULL block via "sort_metric_value IS NULL".
		// The ORDER BY gains a leading "(col IS NULL) ASC" key for deterministic NULL-last.
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:     "accuracy",
					SortBySQLColumn:     model.MetricSortSQLAlias,
					SortByFieldIsString: false,
					SortByFieldValue:    float64(0.5),
					KeyFieldName:        "KeyField",
					KeyFieldValue:       "uuid-1",
					IsDesc:              false,
				},
			},
			wantSQL:  "SELECT * FROM MyTable WHERE (sort_metric_value > ? OR (sort_metric_value = ? AND KeyField >= ?) OR sort_metric_value IS NULL) ORDER BY (sort_metric_value IS NULL) ASC, sort_metric_value ASC, KeyField ASC LIMIT 124",
			wantArgs: []interface{}{float64(0.5), float64(0.5), "uuid-1"},
		},
		// Metric sort, non-NULL cursor, DESC (case A).
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:     "accuracy",
					SortBySQLColumn:     model.MetricSortSQLAlias,
					SortByFieldIsString: false,
					SortByFieldValue:    float64(0.5),
					KeyFieldName:        "KeyField",
					KeyFieldValue:       "uuid-1",
					IsDesc:              true,
				},
			},
			wantSQL:  "SELECT * FROM MyTable WHERE (sort_metric_value < ? OR (sort_metric_value = ? AND KeyField <= ?) OR sort_metric_value IS NULL) ORDER BY (sort_metric_value IS NULL) ASC, sort_metric_value DESC, KeyField DESC LIMIT 124",
			wantArgs: []interface{}{float64(0.5), float64(0.5), "uuid-1"},
		},
		// Metric sort, NULL cursor, ASC (case B): all non-NULL rows are already paged
		// through; advance within the trailing NULL block using the key alone.
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:   "accuracy",
					SortBySQLColumn:   model.MetricSortSQLAlias,
					SortByFieldIsNull: true,
					KeyFieldName:      "KeyField",
					KeyFieldValue:     "uuid-9",
					IsDesc:            false,
				},
			},
			wantSQL:  "SELECT * FROM MyTable WHERE (sort_metric_value IS NULL AND KeyField >= ?) ORDER BY (sort_metric_value IS NULL) ASC, sort_metric_value ASC, KeyField ASC LIMIT 124",
			wantArgs: []interface{}{"uuid-9"},
		},
		// Metric sort, NULL cursor, DESC (case B): key tie-break flips to <=.
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:   "accuracy",
					SortBySQLColumn:   model.MetricSortSQLAlias,
					SortByFieldIsNull: true,
					KeyFieldName:      "KeyField",
					KeyFieldValue:     "uuid-9",
					IsDesc:            true,
				},
			},
			wantSQL:  "SELECT * FROM MyTable WHERE (sort_metric_value IS NULL AND KeyField <= ?) ORDER BY (sort_metric_value IS NULL) ASC, sort_metric_value DESC, KeyField DESC LIMIT 124",
			wantArgs: []interface{}{"uuid-9"},
		},
	}

	for _, test := range tests {
		sql := sq.Select("*").From("MyTable")
		gotSQL, gotArgs, err := test.in.AddFilterToSelect(test.in.AddPaginationToSelect(sql, nil, ""), nil).ToSql()

		if gotSQL != test.wantSQL || !reflect.DeepEqual(gotArgs, test.wantArgs) || err != nil {
			t.Errorf("BuildListSQLQuery(%+v) =\nGot: %q, %v, %v\nWant: %q, %v, nil",
				test.in, gotSQL, gotArgs, err, test.wantSQL, test.wantArgs)
		}
	}
}

func TestTokenSerialization(t *testing.T) {
	protoFilter := &api.Filter{Predicates: []*api.Predicate{
		{
			Key:   "name",
			Op:    api.Predicate_EQUALS,
			Value: &api.Predicate_StringValue{StringValue: "SomeName"},
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

func TestUnmarshalInvalidMetricNameRoundTrip(t *testing.T) {
	// A tampered token may carry a metric name that fails the metric-name
	// pattern (e.g. contains a slash). unmarshal must reject it before the
	// value is used anywhere.
	badToken := token{
		SortByFieldName: "val/loss",
		SortBySQLColumn: model.MetricSortSQLAlias,
		KeyFieldName:    "UUID",
	}
	s, err := badToken.marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	got := &token{}
	if err := got.unmarshal(s); err == nil {
		t.Errorf("unmarshal token with metric name %q: expected error, got nil", "val/loss")
	}
}

func TestUnmarshalMetricSortToken(t *testing.T) {
	// Metric-sort tokens carry the raw metric name in SortByFieldName and the
	// fixed alias in SortBySQLColumn. unmarshal must accept them as-is — both
	// identifier-like names ("accuracy") and hyphenated ones ("log-loss") —
	// with no migration or mutation of any field.
	for _, metricName := range []string{"accuracy", "log-loss"} {
		t.Run(metricName, func(t *testing.T) {
			tok := token{
				SortByFieldName:   metricName,
				SortBySQLColumn:   model.MetricSortSQLAlias,
				SortByFieldValue:  0.5,
				SortByFieldPrefix: "",
				KeyFieldName:      "UUID",
				KeyFieldValue:     "abc",
				KeyFieldPrefix:    "pipeline_runs.",
				IsDesc:            false,
			}
			s, err := tok.marshal()
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			got := &token{}
			if err := got.unmarshal(s); err != nil {
				t.Fatalf("unmarshal metric sort token: %v", err)
			}

			if got.SortByFieldName != metricName {
				t.Errorf("SortByFieldName = %q, want %q", got.SortByFieldName, metricName)
			}
			if got.SortBySQLColumn != model.MetricSortSQLAlias {
				t.Errorf("SortBySQLColumn = %q, want %q", got.SortBySQLColumn, model.MetricSortSQLAlias)
			}
			// Prefixes are stored with their trailing dot and left untouched;
			// the dot is stripped only at SQL build time (see qualifyColumn).
			if got.KeyFieldPrefix != "pipeline_runs." {
				t.Errorf("KeyFieldPrefix = %q, want %q", got.KeyFieldPrefix, "pipeline_runs.")
			}
		})
	}
}

func TestMatches(t *testing.T) {
	protoFilter1 := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:   "Name",
				Op:    api.Predicate_EQUALS,
				Value: &api.Predicate_StringValue{StringValue: "SomeName"},
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
				Key:   "Name",
				Op:    api.Predicate_NOT_EQUALS, // Not equals as opposed to equals above.
				Value: &api.Predicate_StringValue{StringValue: "SomeName"},
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
		{
			o1:   &Options{token: &token{SortByFieldName: "accuracy", SortBySQLColumn: model.MetricSortSQLAlias}},
			o2:   &Options{token: &token{SortByFieldName: "log-loss", SortBySQLColumn: model.MetricSortSQLAlias}},
			want: false,
		},
		// Metric sort: same metric name is the same query.
		{
			o1:   &Options{token: &token{SortByFieldName: "accuracy", SortBySQLColumn: model.MetricSortSQLAlias}},
			o2:   &Options{token: &token{SortByFieldName: "accuracy", SortBySQLColumn: model.MetricSortSQLAlias}},
			want: true,
		},
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
	sql, _, err := listableOptions.AddSortingToSelect(sqlBuilder, nil, "").ToSql()
	assert.Nil(t, err)

	assert.Contains(t, sql, "pipeline_versions.Name") // sorting field
	assert.Contains(t, sql, "pipeline_versions.UUID") // primary key field
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
			Key:   "status",
			Op:    api.Predicate_EQUALS,
			Value: &api.Predicate_StringValue{StringValue: "Succeeded"},
		},
	}
	newFilter, _ := filter.New(protoFilter)
	listableOptions, err := NewOptions(listable, 10, "name", newFilter)
	assert.Nil(t, err)
	sqlBuilder := sq.Select("*").From("run_details")
	sql, args, err := listableOptions.AddFilterToSelect(sqlBuilder, nil).ToSql()
	assert.Nil(t, err)
	assert.Contains(t, sql, "WHERE (Conditions = ?)") // status is not case-insensitive; exact comparison
	assert.Contains(t, args, "Succeeded")

	notEqualProtoFilter := &api.Filter{}
	notEqualProtoFilter.Predicates = []*api.Predicate{
		{
			Key:   "status",
			Op:    api.Predicate_NOT_EQUALS,
			Value: &api.Predicate_StringValue{StringValue: "somevalue"},
		},
	}
	newNotEqualFilter, _ := filter.New(notEqualProtoFilter)
	listableOptions, err = NewOptions(listable, 10, "name", newNotEqualFilter)
	assert.Nil(t, err)
	sqlBuilder = sq.Select("*").From("run_details")
	sql, args, err = listableOptions.AddFilterToSelect(sqlBuilder, nil).ToSql()
	assert.Nil(t, err)
	assert.Contains(t, sql, "WHERE (Conditions <> ?)") // status is not case-insensitive; exact comparison
	assert.Contains(t, args, "somevalue")
}
