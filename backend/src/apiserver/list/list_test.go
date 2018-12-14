package list

import (
	"errors"
	"reflect"
	"testing"

	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"

	sq "github.com/Masterminds/squirrel"
	"github.com/google/go-cmp/cmp"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
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

var fakeAPIToModelMap = map[string]string{
	"timestamp": "CreatedTimestamp",
	"name":      "FakeName",
	"id":        "PrimaryKey",
}

func (f *fakeListable) APIToModelFieldMap() map[string]string {
	return fakeAPIToModelMap
}

func TestNextPageToken_ValidTokens(t *testing.T) {
	l := &fakeListable{PrimaryKey: "uuid123", FakeName: "Fake", CreatedTimestamp: 1234}

	tests := []struct {
		inOpts *Options
		want   *token
	}{
		{
			inOpts: &Options{
				PageSize: 10, token: &token{SortByFieldName: "CreatedTimestamp", IsDesc: true},
			},
			want: &token{
				SortByFieldName:  "CreatedTimestamp",
				SortByFieldValue: int64(1234),
				KeyFieldName:     "PrimaryKey",
				KeyFieldValue:    "uuid123",
				IsDesc:           true,
			},
		},
		{
			inOpts: &Options{
				PageSize: 10, token: &token{SortByFieldName: "PrimaryKey", IsDesc: true},
			},
			want: &token{
				SortByFieldName:  "PrimaryKey",
				SortByFieldValue: "uuid123",
				KeyFieldName:     "PrimaryKey",
				KeyFieldValue:    "uuid123",
				IsDesc:           true,
			},
		},
		{
			inOpts: &Options{
				PageSize: 10, token: &token{SortByFieldName: "FakeName", IsDesc: false},
			},
			want: &token{
				SortByFieldName:  "FakeName",
				SortByFieldValue: "Fake",
				KeyFieldName:     "PrimaryKey",
				KeyFieldValue:    "uuid123",
				IsDesc:           false,
			},
		},
	}

	for _, test := range tests {
		got, err := test.inOpts.nextPageToken(l)

		if !cmp.Equal(got, test.want) || err != nil {
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
	want := errors.New(`cannot sort by field "Timestamp" on type "fakeListable"`)

	got, err := inOpts.nextPageToken(l)

	if !reflect.DeepEqual(err, want) {
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
		SortByFieldName:  "SortField",
		SortByFieldValue: "string_field_value",
		KeyFieldName:     "KeyField",
		KeyFieldValue:    "string_key_value",
		IsDesc:           true,
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
		SortByFieldName:  "SortField",
		SortByFieldValue: "string_field_value",
		KeyFieldName:     "KeyField",
		KeyFieldValue:    "string_key_value",
		IsDesc:           true,
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
					KeyFieldName:    "PrimaryKey",
					SortByFieldName: "CreatedTimestamp",
					IsDesc:          false,
				},
			},
		},
		{
			sortBy: "timestamp",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:    "PrimaryKey",
					SortByFieldName: "CreatedTimestamp",
					IsDesc:          false,
				},
			},
		},
		{
			sortBy: "name",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:    "PrimaryKey",
					SortByFieldName: "FakeName",
					IsDesc:          false,
				},
			},
		},
		{
			sortBy: "name asc",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:    "PrimaryKey",
					SortByFieldName: "FakeName",
					IsDesc:          false,
				},
			},
		},
		{
			sortBy: "name desc",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:    "PrimaryKey",
					SortByFieldName: "FakeName",
					IsDesc:          true,
				},
			},
		},
		{
			sortBy: "id desc",
			want: &Options{
				PageSize: pageSize,
				token: &token{
					KeyFieldName:    "PrimaryKey",
					SortByFieldName: "PrimaryKey",
					IsDesc:          true,
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
			&api.Predicate{
				Key:   "name",
				Op:    api.Predicate_EQUALS,
				Value: &api.Predicate_StringValue{StringValue: "SomeName"},
			},
		},
	}

	protoFilterWithRightKeyNames := &api.Filter{
		Predicates: []*api.Predicate{
			&api.Predicate{
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

	got, err := NewOptions(&fakeListable{}, 10, "timestamp", protoFilter)
	want := &Options{
		PageSize: 10,
		token: &token{
			KeyFieldName:    "PrimaryKey",
			SortByFieldName: "CreatedTimestamp",
			IsDesc:          false,
			Filter:          f,
		},
	}

	opts := []cmp.Option{
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
			&api.Predicate{
				Key:   "unknownfield",
				Op:    api.Predicate_EQUALS,
				Value: &api.Predicate_StringValue{StringValue: "SomeName"},
			},
		},
	}

	got, err := NewOptions(&fakeListable{}, 10, "timestamp", protoFilter)
	if err == nil {
		t.Errorf("NewOptions(protoFilter=%+v) =\nGot: %+v, <nil>\nWant error", protoFilter, got)
	}
}

func TestAddToSelect(t *testing.T) {
	protoFilter := &api.Filter{
		Predicates: []*api.Predicate{
			&api.Predicate{
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
					SortByFieldName:  "SortField",
					SortByFieldValue: "value",
					KeyFieldName:     "KeyField",
					KeyFieldValue:    1111,
					IsDesc:           true,
				},
			},
			wantSQL:  "SELECT * FROM MyTable WHERE (SortField < ? OR (SortField = ? AND KeyField <= ?)) ORDER BY SortField DESC, KeyField DESC LIMIT 124",
			wantArgs: []interface{}{"value", "value", 1111},
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:  "SortField",
					SortByFieldValue: "value",
					KeyFieldName:     "KeyField",
					KeyFieldValue:    1111,
					IsDesc:           false,
				},
			},
			wantSQL:  "SELECT * FROM MyTable WHERE (SortField > ? OR (SortField = ? AND KeyField >= ?)) ORDER BY SortField ASC, KeyField ASC LIMIT 124",
			wantArgs: []interface{}{"value", "value", 1111},
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:  "SortField",
					SortByFieldValue: "value",
					KeyFieldName:     "KeyField",
					KeyFieldValue:    1111,
					IsDesc:           false,
					Filter:           f,
				},
			},
			wantSQL:  "SELECT * FROM MyTable WHERE (SortField > ? OR (SortField = ? AND KeyField >= ?)) AND Name = ? ORDER BY SortField ASC, KeyField ASC LIMIT 124",
			wantArgs: []interface{}{"value", "value", 1111, "SomeName"},
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName: "SortField",
					KeyFieldName:    "KeyField",
					KeyFieldValue:   1111,
					IsDesc:          true,
				},
			},
			wantSQL:  "SELECT * FROM MyTable ORDER BY SortField DESC, KeyField DESC LIMIT 124",
			wantArgs: nil,
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:  "SortField",
					SortByFieldValue: "value",
					KeyFieldName:     "KeyField",
					IsDesc:           false,
				},
			},
			wantSQL:  "SELECT * FROM MyTable ORDER BY SortField ASC, KeyField ASC LIMIT 124",
			wantArgs: nil,
		},
		{
			in: &Options{
				PageSize: 123,
				token: &token{
					SortByFieldName:  "SortField",
					SortByFieldValue: "value",
					KeyFieldName:     "KeyField",
					IsDesc:           false,
					Filter:           f,
				},
			},
			wantSQL:  "SELECT * FROM MyTable WHERE Name = ? ORDER BY SortField ASC, KeyField ASC LIMIT 124",
			wantArgs: []interface{}{"SomeName"},
		},
	}

	for _, test := range tests {
		sql := sq.Select("*").From("MyTable")
		gotSQL, gotArgs, err := test.in.AddToSelect(sql).ToSql()

		if gotSQL != test.wantSQL || !reflect.DeepEqual(gotArgs, test.wantArgs) || err != nil {
			t.Errorf("BuildListSQLQuery(%+v) =\nGot: %q, %v, %v\nWant: %q, %v, nil",
				test.in, gotSQL, gotArgs, err, test.wantSQL, test.wantArgs)
		}
	}
}

func TestTokenSerialization(t *testing.T) {
	tests := []struct {
		in   *token
		want *token
	}{
		// string values in sort by fields
		{
			in: &token{
				SortByFieldName:  "SortField",
				SortByFieldValue: "string_field_value",
				KeyFieldName:     "KeyField",
				KeyFieldValue:    "string_key_value",
				IsDesc:           true},
			want: &token{
				SortByFieldName:  "SortField",
				SortByFieldValue: "string_field_value",
				KeyFieldName:     "KeyField",
				KeyFieldValue:    "string_key_value",
				IsDesc:           true},
		},
		// int values get deserialized as floats by JSON unmarshal.
		{
			in: &token{
				SortByFieldName:  "SortField",
				SortByFieldValue: 100,
				KeyFieldName:     "KeyField",
				KeyFieldValue:    200,
				IsDesc:           true},
			want: &token{
				SortByFieldName:  "SortField",
				SortByFieldValue: float64(100),
				KeyFieldName:     "KeyField",
				KeyFieldValue:    float64(200),
				IsDesc:           true},
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
		if !cmp.Equal(got, test.want) {
			t.Errorf("token.unmarshal(%q) =\nGot: %+v\nWant: %+v\nDiff:\n%s",
				s, got, test.want, cmp.Diff(test.want, got))
		}
	}
}
