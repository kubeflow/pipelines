package filter

import (
	"encoding/json"
	"testing"

	"github.com/Masterminds/squirrel"
	"github.com/golang/protobuf/proto"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
)

func TestValidNewFilters(t *testing.T) {
	opts := []cmp.Option{
		cmp.AllowUnexported(Filter{}),
		cmp.FilterPath(func(p cmp.Path) bool {
			return p.String() == "filterProto"
		}, cmp.Ignore()),
		cmpopts.EquateEmpty(),
	}

	tests := []struct {
		protoStr string
		want     *Filter
	}{
		{
			`predicates { key: "status" op: EQUALS string_value: "Running" }`,
			&Filter{eq: map[string][]interface{}{"status": {"Running"}}},
		},
		{
			`predicates { key: "status" op: NOT_EQUALS string_value: "Running" }`,
			&Filter{neq: map[string][]interface{}{"status": {"Running"}}},
		},
		{
			`predicates { key: "total" op: GREATER_THAN int_value: 10 }`,
			&Filter{gt: map[string][]interface{}{"total": {int32(10)}}},
		},
		{
			`predicates { key: "total" op: GREATER_THAN_EQUALS long_value: 10 }`,
			&Filter{gte: map[string][]interface{}{"total": {int64(10)}}},
		},
		{
			`predicates { key: "total" op: LESS_THAN timestamp_value { seconds: 10 }}`,
			&Filter{lt: map[string][]interface{}{"total": {int64(10)}}},
		},
		{
			`predicates { key: "total" op: LESS_THAN_EQUALS timestamp_value { seconds: 10 }}`,
			&Filter{lte: map[string][]interface{}{"total": {int64(10)}}},
		},
		{
			`predicates {
				key: "label" op: IN
				string_values { values: 'label_1' values: 'label_2' } }`,
			&Filter{in: map[string][]interface{}{"label": {[]string{"label_1", "label_2"}}}},
		},
		{
			`predicates {
				key: "intvalues" op: IN
				int_values { values: 10 values: 20 } }`,
			&Filter{in: map[string][]interface{}{"intvalues": {[]int32{10, 20}}}},
		},
		{
			`predicates {
				key: "longvalues" op: IN
				long_values { values: 100 values: 200 } }`,
			&Filter{in: map[string][]interface{}{"longvalues": {[]int64{100, 200}}}},
		},
		{
			`predicates {
				key: "label" op: IS_SUBSTRING string_value: "label_substring" }`,
			&Filter{substring: map[string][]interface{}{"label": {"label_substring"}}},
		},
	}

	for _, test := range tests {
		filterProto := &api.Filter{}
		if err := proto.UnmarshalText(test.protoStr, filterProto); err != nil {
			t.Errorf("Failed to unmarshal Filter text proto\n%q\nError: %v", test.protoStr, err)
			continue
		}

		got, err := New(filterProto)
		if !cmp.Equal(got, test.want, opts...) || err != nil {
			t.Errorf("New(%+v) = %+v, %v\nWant %+v, nil", *filterProto, got, err, test.want)
		}
	}
}

func TestValidNewFiltersWithKeyMap(t *testing.T) {
	opts := []cmp.Option{
		cmp.AllowUnexported(Filter{}),
		cmp.FilterPath(func(p cmp.Path) bool {
			return p.String() == "filterProto"
		}, cmp.Ignore()),
		cmpopts.EquateEmpty(),
	}

	tests := []struct {
		protoStr string
		want     *Filter
	}{
		{
			`predicates { key: "name" op: EQUALS string_value: "pipeline" }`,
			&Filter{eq: map[string][]interface{}{"pipelines.Name": {"pipeline"}}},
		},
		{
			`predicates { key: "name" op: NOT_EQUALS string_value: "pipeline" }`,
			&Filter{neq: map[string][]interface{}{"pipelines.Name": {"pipeline"}}},
		},
		{
			`predicates {
				key: "name" op: IN
				string_values { values: 'pipeline_1' values: 'pipeline_2' } }`,
			&Filter{in: map[string][]interface{}{"pipelines.Name": {[]string{"pipeline_1", "pipeline_2"}}}},
		},
		{
			`predicates {
				key: "name" op: IS_SUBSTRING string_value: "pipeline" }`,
			&Filter{substring: map[string][]interface{}{"pipelines.Name": {"pipeline"}}},
		},
	}

	for _, test := range tests {
		filterProto := &api.Filter{}
		if err := proto.UnmarshalText(test.protoStr, filterProto); err != nil {
			t.Errorf("Failed to unmarshal Filter text proto\n%q\nError: %v", test.protoStr, err)
			continue
		}

		keyMap := map[string]string{
			"id":          "UUID",
			"name":        "Name",
			"created_at":  "CreatedAtInSec",
			"description": "Description",
		}
		modelName := "pipelines"
		got, err := NewWithKeyMap(filterProto, keyMap, modelName)
		if !cmp.Equal(got, test.want, opts...) || err != nil {
			t.Errorf("New(%+v) = %+v, %v\nWant %+v, nil", *filterProto, got, err, test.want)
		}
	}
}

func TestInvalidFilters(t *testing.T) {
	tests := []struct {
		protoStr string
	}{
		{
			`predicates { key: "status" op: EQUALS
			 string_values { values: "v1" values: "v2" }} `,
		},
		{
			`predicates { key: "status" op: NOT_EQUALS
			 string_values { values: "v1" values: "v2"} }`,
		},
		{
			`predicates { key: "total" op: GREATER_THAN
			 int_values { values: 10  values: 20} }`,
		},
		{
			`predicates { key: "total" op: GREATER_THAN_EQUALS
			 long_values { values: 10 values: 20} }`,
		},
		{
			`predicates { key: "total" op: LESS_THAN
			 int_values { values: 10  values: 20} }`,
		},
		{
			`predicates { key: "total" op: LESS_THAN_EQUALS
			 long_values { values: 10 values: 20} }`,
		},
		{
			`predicates { key: "total" op: IS_SUBSTRING
			 long_values { values: 10 values: 20} }`,
		},
		{
			`predicates { key: "total" op: IS_SUBSTRING
			 int_values { values: 10  values: 20} }`,
		},

		{
			`predicates { key: "total" op: IN int_value: 10 }`,
		},
		{
			`predicates { key: "total" op: IN long_value: 200}`,
		},
		{
			`predicates { key: "total" op: IN string_value: "value"}`,
		},
		{
			`predicates { key: "total" op: IN timestamp_value { seconds: 10 }}`,
		},
		// Invalid predicate
		{
			`predicates { key: "total" timestamp_value { seconds: 10 }}`,
		},
		// No value
		{
			`predicates { key: "total" op: IN }`,
		},
		// Bad timestamp
		{
			`predicates { key: "total" op: LESS_THAN
				timestamp_value { seconds: -100000000000 }}`,
		},
	}

	for _, test := range tests {
		filterProto := &api.Filter{}
		if err := proto.UnmarshalText(test.protoStr, filterProto); err != nil {
			t.Errorf("Failed to unmarshal Filter text proto\n%q\nError: %v", test.protoStr, err)
			continue
		}

		got, err := New(filterProto)
		if err == nil {
			t.Errorf("New(%+v) = %+v, <nil>\nWant non-nil error ", *filterProto, got)
		}
	}
}

func TestAddToSelect(t *testing.T) {
	tests := []struct {
		protoStr string
		wantSQL  string
		wantArgs []interface{}
	}{
		{
			`predicates { key: "status" op: EQUALS string_value: "Running" }`,
			"SELECT mycolumn WHERE status = ?",
			[]interface{}{"Running"},
		},
		{
			`predicates { key: "status" op: EQUALS string_value: "Running" }
			 predicates { key: "status" op: EQUALS string_value: "Stopped" }`,
			"SELECT mycolumn WHERE status = ? AND status = ?",
			[]interface{}{"Running", "Stopped"},
		},
		{
			`predicates { key: "status" op: EQUALS string_value: "Running" }`,
			"SELECT mycolumn WHERE status = ?",
			[]interface{}{"Running"},
		},
		{
			`predicates { key: "status" op: EQUALS string_value: "Running" }
		   predicates { key: "total" op: GREATER_THAN_EQUALS  long_value: 100 }`,
			"SELECT mycolumn WHERE status = ? AND total >= ?",
			[]interface{}{"Running", int64(100)},
		},
		{
			`predicates { key: "status" op: NOT_EQUALS string_value: "Running" }
		   predicates { key: "total" op: GREATER_THAN  long_value: 100 }`,
			"SELECT mycolumn WHERE status <> ? AND total > ?",
			[]interface{}{"Running", int64(100)},
		},
		{
			`predicates { key: "date" op: LESS_THAN timestamp_value { seconds: 10 } }
		   predicates { key: "total" op: LESS_THAN_EQUALS  int_value: 100 }`,
			"SELECT mycolumn WHERE date < ? AND total <= ?",
			[]interface{}{int64(10), int32(100)},
		},
		{
			`predicates { key: "total" op: IN int_values {values: 1 values: 2 values: 3} }`,
			"SELECT mycolumn WHERE total IN (?,?,?)",
			[]interface{}{int32(1), int32(2), int32(3)},
		},
		{
			`predicates { key: "runs" op: IN  long_values {values: 100 values: 200}}`,
			"SELECT mycolumn WHERE runs IN (?,?)",
			[]interface{}{int64(100), int64(200)},
		},
		{
			`predicates { key: "label" op: IN  string_values {values: "l1" values: "l2"}}`,
			"SELECT mycolumn WHERE label IN (?,?)",
			[]interface{}{"l1", "l2"},
		},
		{
			`predicates { key: "label" op: IS_SUBSTRING  string_value: "label_substring" }`,
			"SELECT mycolumn WHERE label LIKE ?",
			[]interface{}{"%label_substring%"},
		},
		{
			`predicates { key: "label" op: IS_SUBSTRING  string_value: "label_substring1" }
			 predicates { key: "label" op: IS_SUBSTRING  string_value: "label_substring2" }`,
			"SELECT mycolumn WHERE label LIKE ? AND label LIKE ?",
			[]interface{}{"%label_substring1%", "%label_substring2%"},
		},
	}

	for _, test := range tests {
		filterProto := &api.Filter{}
		if err := proto.UnmarshalText(test.protoStr, filterProto); err != nil {
			t.Errorf("Failed to unmarshal Filter text proto\n%q\nError: %v", test.protoStr, err)
			continue
		}

		filter, err := New(filterProto)
		if err != nil {
			t.Errorf("New(%+v) = %+v, %v\nWant nil error", *filterProto, filter, err)
			continue
		}

		sb := squirrel.Select("mycolumn")
		gotSQL, gotArgs, err := filter.AddToSelect(sb).ToSql()
		if !cmp.Equal(gotSQL, test.wantSQL) || !cmp.Equal(gotArgs, test.wantArgs) || err != nil {
			t.Errorf("Filter.AddToSelect(%+v).ToSql() =\nGot: %+v, %v, %v\nWant: %+v, %+v, <nil>", filter, gotSQL, gotArgs, err, test.wantSQL, test.wantArgs)
		}
	}
}

func TestMarshalJSON(t *testing.T) {
	f := &Filter{
		filterProto: &api.Filter{
			Predicates: []*api.Predicate{
				&api.Predicate{
					Key: "Name", Op: api.Predicate_EQUALS,
					Value: &api.Predicate_StringValue{StringValue: "SomeName"},
				},
			},
		},
		eq: map[string][]interface{}{"name": {"SomeName"}},
	}

	want := `{"FilterProto":"{\"predicates\":[{\"op\":\"EQUALS\",\"key\":\"Name\",\"stringValue\":\"SomeName\"}]}","EQ":{"name":["SomeName"]},"NEQ":null,"GT":null,"GTE":null,"LT":null,"LTE":null,"IN":null,"SUBSTRING":null}`

	got, err := json.Marshal(f)
	if err != nil || string(got) != want {
		t.Errorf("json.Marshal(%+v):\n%s, %v\nWant:%s, nil error\n", f, got, err, want)
	}
}

func TestUnmarshalJSON(t *testing.T) {
	in := `{"FilterProto":"{\"predicates\":[{\"op\":\"EQUALS\",\"key\":\"Name\",\"stringValue\":\"SomeName\"}]}","EQ":{"name":["SomeName"]},"NEQ":null,"GT":null,"GTE":null,"LT":null,"LTE":null,"IN":null,"SUBSTRING":null}`

	want := &Filter{
		filterProto: &api.Filter{
			Predicates: []*api.Predicate{
				&api.Predicate{
					Key: "Name", Op: api.Predicate_EQUALS,
					Value: &api.Predicate_StringValue{StringValue: "SomeName"},
				},
			},
		},
		eq: map[string][]interface{}{"name": {"SomeName"}},
	}

	got := &Filter{}
	err := json.Unmarshal([]byte(in), got)
	if err != nil || !cmp.Equal(got, want, cmp.AllowUnexported(Filter{})) {
		t.Errorf("json.Unmarshal(%+v):\nGot: %v, Error: %v\nWant:\n%+v, Error: nil\nDiff:%s\n", in, got, err, want, cmp.Diff(want, got, cmp.AllowUnexported(Filter{})))
	}
}
