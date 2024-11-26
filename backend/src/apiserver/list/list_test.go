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

func (f *fakeListable) GetField(name string) (string, bool) {
	if field, ok := fakeAPIToModelMap[name]; ok {
		return field, true
	}
	if strings.HasPrefix(name, "metric:") {
		return name[7:], true
	}
	return "", false
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
					KeyFieldName:      "PrimaryKey",
					KeyFieldPrefix:    "",
					SortByFieldName:   "CreatedTimestamp",
					SortByFieldPrefix: "",
					IsDesc:            false,
				},
			},
		},
		{
			sortBy: "timestamp",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:      "PrimaryKey",
					KeyFieldPrefix:    "",
					SortByFieldName:   "CreatedTimestamp",
					SortByFieldPrefix: "",
					IsDesc:            false,
				},
			},
		},
		{
			sortBy: "name",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:      "PrimaryKey",
					KeyFieldPrefix:    "",
					SortByFieldName:   "FakeName",
					SortByFieldPrefix: "",
					IsDesc:            false,
				},
			},
		},
		{
			sortBy: "name asc",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:      "PrimaryKey",
					KeyFieldPrefix:    "",
					SortByFieldName:   "FakeName",
					SortByFieldPrefix: "",
					IsDesc:            false,
				},
			},
		},
		{
			sortBy: "name desc",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:      "PrimaryKey",
					KeyFieldPrefix:    "",
					SortByFieldName:   "FakeName",
					SortByFieldPrefix: "",
					IsDesc:            true,
				},
			},
		},
		{
			sortBy: "id desc",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:      "PrimaryKey",
					KeyFieldPrefix:    "",
					SortByFieldName:   "PrimaryKey",
					SortByFieldPrefix: "",
					IsDesc:            true,
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
				Key:   "name",
				Op:    api.Predicate_EQUALS,
				Value: &api.Predicate_StringValue{StringValue: "SomeName"},
			},
		},
	}
	newFilter, _ := filter.New(protoFilter)

	protoFilterWithRightKeyNames := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:   "FakeName",
				Op:    api.Predicate_EQUALS,
				Value: &api.Predicate_StringValue{StringValue: "SomeName"},
			},
		},
	}

	f, err := filter.New(protoFilterWithRightKeyNames)
	if err != nil {
		t.Fatalf("failed to parse filter proto %+v: %v", protoFilter, err)
	}

	got, err := NewOptions(&fakeListable{}, 10, "timestamp", newFilter)
	want := &Options{
		PageSize: 10,
		token: &token{
			KeyFieldName:      "PrimaryKey",
			KeyFieldPrefix:    "",
			SortByFieldName:   "CreatedTimestamp",
			SortByFieldPrefix: "",
			IsDesc:            false,
			Filter:            f,
		},
	}

	opts := []cmp.Option{
		cmpopts.EquateEmpty(), protocmp.Transform(),
		cmp.AllowUnexported(Options{}),
		cmp.AllowUnexported(filter.Filter{}),
	}

	if !cmp.Equal(got, want, opts...) || err != nil {
		t.Errorf("NewOptions(protoFilter=%+v) =\nGot: %+v, %v\nWant: %+v, nil\nDiff:\n%s",
			protoFilter, got, err, want, cmp.Diff(got, want, opts...))
	}
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

	protoFilterWithRightKeyNames := &api.Filter{
		Predicates: []*api.Predicate{
			{
				Key:   "FinishedAtInSec",
				Op:    api.Predicate_GREATER_THAN,
				Value: &api.Predicate_StringValue{StringValue: "SomeTime"},
			},
		},
	}

	f, err := filter.New(protoFilterWithRightKeyNames)
	if err != nil {
		t.Fatalf("failed to parse filter proto %+v: %v", protoFilter, err)
	}

	got, err := NewOptions(&model.Run{}, 10, "name", newFilter)
	want := &Options{
		PageSize: 10,
		token: &token{
			KeyFieldName:    "UUID",
			SortByFieldName: "DisplayName",
			IsDesc:          false,
			Filter:          f,
		},
	}

	opts := []cmp.Option{
		cmpopts.EquateEmpty(), protocmp.Transform(),
		cmp.AllowUnexported(Options{}),
		cmp.AllowUnexported(filter.Filter{}),
	}

	if !cmp.Equal(got, want, opts...) || err != nil {
		t.Errorf("NewOptions(protoFilter=%+v) =\nGot: %+v, %v\nWant: %+v, nil\nDiff:\n%s",
			protoFilter, got, err, want, cmp.Diff(got, want, opts...))
	}
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
					SortByFieldValue:  "value",
					SortByFieldPrefix: "",
					KeyFieldName:      "KeyField",
					KeyFieldValue:     1111,
					KeyFieldPrefix:    "",
					IsDesc:            true,
				},
			},
			wantSQL:  "SELECT * FROM MyTable WHERE (SortField < ? OR (SortField = ? AND KeyField <= ?)) ORDER BY SortField DESC, KeyField DESC LIMIT 124",
			wantArgs: []interface{}{"value", "value", 1111},
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:   "SortField",
					SortByFieldValue:  "value",
					SortByFieldPrefix: "",
					KeyFieldName:      "KeyField",
					KeyFieldValue:     1111,
					KeyFieldPrefix:    "",
					IsDesc:            false,
				},
			},
			wantSQL:  "SELECT * FROM MyTable WHERE (SortField > ? OR (SortField = ? AND KeyField >= ?)) ORDER BY SortField ASC, KeyField ASC LIMIT 124",
			wantArgs: []interface{}{"value", "value", 1111},
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:   "SortField",
					SortByFieldValue:  "value",
					SortByFieldPrefix: "",
					KeyFieldName:      "KeyField",
					KeyFieldValue:     1111,
					KeyFieldPrefix:    "",
					IsDesc:            false,
					Filter:            f,
				},
			},
			wantSQL:  "SELECT * FROM MyTable WHERE (SortField > ? OR (SortField = ? AND KeyField >= ?)) AND Name = ? ORDER BY SortField ASC, KeyField ASC LIMIT 124",
			wantArgs: []interface{}{"value", "value", 1111, "SomeName"},
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:   "SortField",
					SortByFieldPrefix: "",
					KeyFieldName:      "KeyField",
					KeyFieldPrefix:    "",
					KeyFieldValue:     1111,
					IsDesc:            true,
				},
			},
			wantSQL:  "SELECT * FROM MyTable ORDER BY SortField DESC, KeyField DESC LIMIT 124",
			wantArgs: nil,
		},
		{
			in:       EmptyOptions(),
			wantSQL:  fmt.Sprintf("SELECT * FROM MyTable LIMIT %d", math.MaxInt32+1),
			wantArgs: nil,
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:   "SortField",
					SortByFieldValue:  "value",
					SortByFieldPrefix: "",
					KeyFieldName:      "KeyField",
					KeyFieldPrefix:    "",
					IsDesc:            false,
				},
			},
			wantSQL:  "SELECT * FROM MyTable ORDER BY SortField ASC, KeyField ASC LIMIT 124",
			wantArgs: nil,
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:   "SortField",
					SortByFieldValue:  "value",
					SortByFieldPrefix: "",
					KeyFieldName:      "KeyField",
					KeyFieldPrefix:    "",
					IsDesc:            false,
					Filter:            f,
				},
			},
			wantSQL:  "SELECT * FROM MyTable WHERE Name = ? ORDER BY SortField ASC, KeyField ASC LIMIT 124",
			wantArgs: []interface{}{"SomeName"},
		},
	}

	for _, test := range tests {
		sql := sq.Select("*").From("MyTable")
		gotSQL, gotArgs, err := test.in.AddFilterToSelect(test.in.AddPaginationToSelect(sql)).ToSql()

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
	}

	for _, test := range tests {
		got := test.o1.Matches(test.o2)

		if got != test.want {
			t.Errorf("Matches(%+v, %+v) = %v, Want nil %v", test.o1, test.o2, got, test.want)
			continue
		}
	}
}

func TestFilterOnResourceReference(t *testing.T) {
	type testIn struct {
		table        string
		resourceType model.ResourceType
		count        bool
		filter       *model.FilterContext
	}
	tests := []struct {
		in      *testIn
		wantSql string
		wantErr error
	}{
		{
			in: &testIn{
				table:        "testTable",
				resourceType: model.RunResourceType,
				count:        false,
				filter:       &model.FilterContext{},
			},
			wantSql: "SELECT * FROM testTable",
			wantErr: nil,
		},
		{
			in: &testIn{
				table:        "testTable",
				resourceType: model.RunResourceType,
				count:        true,
				filter:       &model.FilterContext{},
			},
			wantSql: "SELECT count(*) FROM testTable",
			wantErr: nil,
		},
		{
			in: &testIn{
				table:        "testTable",
				resourceType: model.RunResourceType,
				count:        false,
				filter:       &model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: "test3"}},
			},
			wantSql: "SELECT * FROM testTable WHERE UUID in (SELECT ResourceUUID FROM resource_references as rf WHERE (rf.ResourceType = ? AND rf.ReferenceUUID = ? AND rf.ReferenceType = ?))",
			wantErr: nil,
		},
		{
			in: &testIn{
				table:        "testTable",
				resourceType: model.RunResourceType,
				count:        true,
				filter:       &model.FilterContext{ReferenceKey: &model.ReferenceKey{Type: model.RunResourceType, ID: "test4"}},
			},
			wantSql: "SELECT count(*) FROM testTable WHERE UUID in (SELECT ResourceUUID FROM resource_references as rf WHERE (rf.ResourceType = ? AND rf.ReferenceUUID = ? AND rf.ReferenceType = ?))",
			wantErr: nil,
		},
	}

	for _, test := range tests {
		sqlBuilder, gotErr := FilterOnResourceReference(test.in.table, []string{"*"}, test.in.resourceType, test.in.count, test.in.filter)
		gotSql, _, err := sqlBuilder.ToSql()
		assert.Nil(t, err)

		if gotSql != test.wantSql || gotErr != test.wantErr {
			t.Errorf("FilterOnResourceReference(%+v) =\nGot: %q, %v\nWant: %q, %v",
				test.in, gotSql, gotErr, test.wantSql, test.wantErr)
		}
	}
}

func TestFilterOnExperiment(t *testing.T) {
	type testIn struct {
		table  string
		count  bool
		filter *model.FilterContext
	}
	tests := []struct {
		in      *testIn
		wantSql string
		wantErr error
	}{
		{
			in: &testIn{
				table:  "testTable",
				count:  false,
				filter: &model.FilterContext{},
			},
			wantSql: "SELECT * FROM testTable WHERE ExperimentUUID = ?",
			wantErr: nil,
		},
		{
			in: &testIn{
				table:  "testTable",
				count:  true,
				filter: &model.FilterContext{},
			},
			wantSql: "SELECT count(*) FROM testTable WHERE ExperimentUUID = ?",
			wantErr: nil,
		},
	}

	for _, test := range tests {
		sqlBuilder, gotErr := FilterOnExperiment(test.in.table, []string{"*"}, test.in.count, "123")
		gotSql, _, err := sqlBuilder.ToSql()
		assert.Nil(t, err)

		if gotSql != test.wantSql || gotErr != test.wantErr {
			t.Errorf("FilterOnExperiment(%+v) =\nGot: %q, %v\nWant: %q, %v",
				test.in, gotSql, gotErr, test.wantSql, test.wantErr)
		}
	}
}

func TestFilterOnNamesapce(t *testing.T) {
	type testIn struct {
		table  string
		count  bool
		filter *model.FilterContext
	}
	tests := []struct {
		in      *testIn
		wantSql string
		wantErr error
	}{
		{
			in: &testIn{
				table:  "testTable",
				count:  false,
				filter: &model.FilterContext{},
			},
			wantSql: "SELECT * FROM testTable WHERE Namespace = ?",
			wantErr: nil,
		},
		{
			in: &testIn{
				table:  "testTable",
				count:  true,
				filter: &model.FilterContext{},
			},
			wantSql: "SELECT count(*) FROM testTable WHERE Namespace = ?",
			wantErr: nil,
		},
	}

	for _, test := range tests {
		sqlBuilder, gotErr := FilterOnNamespace(test.in.table, []string{"*"}, test.in.count, "ns")
		gotSql, _, err := sqlBuilder.ToSql()
		assert.Nil(t, err)

		if gotSql != test.wantSql || gotErr != test.wantErr {
			t.Errorf("FilterOnNamespace(%+v) =\nGot: %q, %v\nWant: %q, %v",
				test.in, gotSql, gotErr, test.wantSql, test.wantErr)
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
	sql, _, err := listableOptions.AddSortingToSelect(sqlBuilder).ToSql()
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
	sql, args, err := listableOptions.AddFilterToSelect(sqlBuilder).ToSql()
	assert.Nil(t, err)
	assert.Contains(t, sql, "WHERE Conditions = ?") // filtering on status, aka Conditions in db
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
	sql, args, err = listableOptions.AddFilterToSelect(sqlBuilder).ToSql()
	assert.Nil(t, err)
	assert.Contains(t, sql, "WHERE Conditions <> ?") // filtering on status, aka Conditions in db
	assert.Contains(t, args, "somevalue")
}
