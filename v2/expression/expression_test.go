package expression_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/kubeflow/pipelines/v2/expression"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/testing/protocmp"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestSelect(t *testing.T) {
	arr := []interface{}{
		1.1, 1.2, 1.3,
	}
	map1 := map[string]interface{}{
		"a": "A",
		"b": "B",
	}
	map2 := map[string]interface{}{
		"nested": map1,
		"bool":   true,
		"double": 1.3,
		"int":    10,
		"str":    "abcdefg",
		"list":   arr,
	}
	struct1, _ := structpb.NewStruct(map1)
	struct2, _ := structpb.NewStruct(map2)
	list1, _ := structpb.NewList(arr)
	tt := []struct {
		name       string
		input      *structpb.Value
		expression string
		output     *structpb.Value
		err        string
	}{{
		input:      structpb.NewStringValue("Hello,World!"),
		expression: "string_value",
		output:     structpb.NewStringValue("Hello,World!"),
	}, {
		input:      structpb.NewStringValue("[1.1,1.2,1.3]"),
		expression: "parseJson(string_value)[0]",
		output:     structpb.NewNumberValue(1.1),
	}, {
		input:      structpb.NewStringValue("invalidjson"),
		expression: "parseJson(string_value)",
		err:        "failed to unmarshal JSON",
	}, {
		input:      structpb.NewNullValue(),
		expression: "string_value",
		output:     structpb.NewStringValue(""),
	}, {
		input:      structpb.NewStringValue("Hello"),
		expression: "struct_value",
		err:        "no such attribute",
	}, {
		input:      structpb.NewStructValue(struct1),
		expression: "struct_value.a",
		output:     structpb.NewStringValue("A"),
	}, {
		input:      structpb.NewStructValue(struct1),
		expression: "struct_value.c",
		err:        "no such key: c",
	}, {
		name:       "select_list_field_from_struct",
		input:      structpb.NewStructValue(struct2),
		expression: "struct_value.list",
		output:     structpb.NewListValue(list1),
	}, {
		name:       "select_nested_struct",
		input:      structpb.NewStructValue(struct2),
		expression: "struct_value.nested",
		output:     structpb.NewStructValue(struct1),
	}, {
		name:       "select_nested_field",
		input:      structpb.NewStructValue(struct2),
		expression: "struct_value.nested.b",
		output:     structpb.NewStringValue("B"),
	}}
	for _, test := range tt {
		name := test.name
		if name == "" {
			name = fmt.Sprintf("expr.Select(value={%+v},expression=%q)", test.input, test.expression)
		}
		t.Run(name, func(t *testing.T) {
			expr, err := expression.New()
			if err != nil {
				t.Fatal(err)
			}
			got, err := expr.Select(test.input, test.expression)
			if test.err != "" {
				if err == nil {
					t.Fatalf("got {%+v}, but expected to fail with %q, but ", got, test.err)
				}
				if !strings.Contains(err.Error(), test.err) {
					t.Fatalf("failed with %q, but does not contain %q", err.Error(), test.err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if !proto.Equal(test.output, got) {
				t.Errorf("got:{%+v}\ndiff: %s", got, cmp.Diff(test.output, got, protocmp.Transform()))
			}
		})
	}
}
