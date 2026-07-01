// Copyright 2021-2024 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package placeholder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestPbValueToString(t *testing.T) {
	tests := []struct {
		name    string
		value   *structpb.Value
		want    string
		wantErr bool
	}{
		{
			name:  "string value",
			value: structpb.NewStringValue("hello"),
			want:  "hello",
		},
		{
			name:  "empty string value",
			value: structpb.NewStringValue(""),
			want:  "",
		},
		{
			name:  "integer number",
			value: structpb.NewNumberValue(42),
			want:  "42",
		},
		{
			name:  "zero",
			value: structpb.NewNumberValue(0),
			want:  "0",
		},
		{
			name:  "negative integer",
			value: structpb.NewNumberValue(-7),
			want:  "-7",
		},
		{
			name:  "float number",
			value: structpb.NewNumberValue(0.001),
			want:  "0.001",
		},
		{
			name:  "float with trailing zeros trimmed",
			value: structpb.NewNumberValue(3.5),
			want:  "3.5",
		},
		{
			name:  "large integer",
			value: structpb.NewNumberValue(100),
			want:  "100",
		},
		{
			name:  "bool true",
			value: structpb.NewBoolValue(true),
			want:  "true",
		},
		{
			name:  "bool false",
			value: structpb.NewBoolValue(false),
			want:  "false",
		},
		{
			name:  "null value",
			value: structpb.NewNullValue(),
			want:  "",
		},
		{
			name:  "nil value",
			value: nil,
			want:  "",
		},
		{
			name: "list value",
			value: structpb.NewListValue(&structpb.ListValue{
				Values: []*structpb.Value{
					structpb.NewStringValue("a"),
					structpb.NewNumberValue(1),
				},
			}),
			want: `["a",1]`,
		},
		{
			name: "struct value",
			value: func() *structpb.Value {
				s, _ := structpb.NewStruct(map[string]interface{}{"key": "val"})
				return structpb.NewStructValue(s)
			}(),
			want: `{"key":"val"}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := PbValueToString(test.value)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.want, got)
			}
		})
	}
}

func TestResolveInputParameterPlaceholders(t *testing.T) {
	tests := []struct {
		name            string
		arg             string
		parameterValues map[string]*structpb.Value
		want            string
		wantErr         bool
	}{
		{
			name: "standalone placeholder",
			arg:  "{{$.inputs.parameters['file']}}",
			parameterValues: map[string]*structpb.Value{
				"file": structpb.NewStringValue("/etc/hosts"),
			},
			want: "/etc/hosts",
		},
		{
			name: "embedded placeholder in fstring",
			arg:  "prefix-{{$.inputs.parameters['text1']}}",
			parameterValues: map[string]*structpb.Value{
				"text1": structpb.NewStringValue("hello"),
			},
			want: "prefix-hello",
		},
		{
			name: "multiple placeholders in single arg",
			arg:  "{{$.inputs.parameters['p1']}}-{{$.inputs.parameters['p2']}}",
			parameterValues: map[string]*structpb.Value{
				"p1": structpb.NewStringValue("val1"),
				"p2": structpb.NewStringValue("val2"),
			},
			want: "val1-val2",
		},
		{
			name:            "plain text without placeholders",
			arg:             "regular-arg",
			parameterValues: map[string]*structpb.Value{},
			want:            "regular-arg",
		},
		{
			name: "number parameter is stringified",
			arg:  "{{$.inputs.parameters['iterations']}}",
			parameterValues: map[string]*structpb.Value{
				"iterations": structpb.NewNumberValue(42),
			},
			want: "42",
		},
		{
			name: "double parameter is stringified",
			arg:  "{{$.inputs.parameters['lr']}}",
			parameterValues: map[string]*structpb.Value{
				"lr": structpb.NewNumberValue(0.001),
			},
			want: "0.001",
		},
		{
			name: "boolean parameter is stringified",
			arg:  "{{$.inputs.parameters['verbose']}}",
			parameterValues: map[string]*structpb.Value{
				"verbose": structpb.NewBoolValue(true),
			},
			want: "true",
		},
		{
			name: "missing parameter returns error",
			arg:  "{{$.inputs.parameters['missing']}}",
			parameterValues: map[string]*structpb.Value{
				"other": structpb.NewStringValue("value"),
			},
			wantErr: true,
		},
		{
			name: "null parameter stringifies to empty",
			arg:  "{{$.inputs.parameters['opt']}}",
			parameterValues: map[string]*structpb.Value{
				"opt": structpb.NewNullValue(),
			},
			want: "",
		},
		{
			name:            "empty string returns empty string",
			arg:             "",
			parameterValues: map[string]*structpb.Value{},
			want:            "",
		},
		{
			name: "pipeline channel name with dashes",
			arg:  "{{$.inputs.parameters['pipelinechannel--someParameterName']}}",
			parameterValues: map[string]*structpb.Value{
				"pipelinechannel--someParameterName": structpb.NewStringValue("resolved"),
			},
			want: "resolved",
		},
		{
			name: "embedded number in fstring",
			arg:  "--epochs={{$.inputs.parameters['epochs']}}",
			parameterValues: map[string]*structpb.Value{
				"epochs": structpb.NewNumberValue(10),
			},
			want: "--epochs=10",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ResolveInputParameterPlaceholders(test.arg, test.parameterValues)
			if test.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.want, got)
			}
		})
	}
}
