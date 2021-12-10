// Copyright 2021 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package expression

import (
	"encoding/json"
	"fmt"

	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/pb"
	"github.com/google/cel-go/common/types/ref"
	"github.com/google/cel-go/interpreter/functions"
	"github.com/kubeflow/pipelines/v2/metadata"
	exprpb "google.golang.org/genproto/googleapis/api/expr/v1alpha1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/known/structpb"
)

const (
	parseJsonCelOverloadID = "parse_json"
)

type Expr struct {
	env *cel.Env
}

func New() (*Expr, error) {
	env, err := cel.NewEnv(
		cel.DeclareContextProto(new(structpb.Value).ProtoReflect().Descriptor()),
		cel.Declarations(
			decls.NewFunction("parseJson",
				decls.NewOverload(
					parseJsonCelOverloadID,
					[]*exprpb.Type{decls.String},
					decls.Dyn,
				),
			),
		),
	)
	if err != nil {
		return nil, err
	}
	return &Expr{env: env}, nil
}

// Select from a protobuf.Value using a CEL expression.
func (e *Expr) Select(v *structpb.Value, expr string) (*structpb.Value, error) {
	ast, issues := e.env.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}
	program, err := e.env.Program(ast, cel.Functions(
		&functions.Overload{
			Operator: parseJsonCelOverloadID,
			Unary:    celParseJson,
		},
	))
	if err != nil {
		return nil, err
	}
	// Add the protobuf.Value field that is set.
	fields := make(map[string]interface{})
	if v != nil {
		v.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
			var celValue interface{}
			celValue, err = pb.NewFieldDescription(fd).GetFrom(v)
			if err != nil {
				return false
			}
			fields[fd.TextName()] = celValue
			return true
		})
		if err != nil {
			return nil, err
		}
		// TODO(Bobgy): discuss whether we need to remove this.
		// We always allow accessing the value as string_value, it gets JSON serialized version of the value.
		text, err := metadata.PbValueToText(v)
		if err != nil {
			return nil, err
		}
		fields["string_value"] = text
	}
	result, _, err := program.Eval(fields)
	if err != nil {
		return nil, fmt.Errorf("evaluation error: %w", err)
	}
	raw := result.Value()
	if _, isProtoMessage := raw.(proto.Message); isProtoMessage {
		switch v := raw.(type) {
		case *structpb.ListValue:
			return structpb.NewListValue(v), nil
		case *structpb.NullValue:
			return structpb.NewNullValue(), nil
		case *structpb.Struct:
			return structpb.NewStructValue(v), nil
		default:
			return nil, fmt.Errorf("failed to convert result to protobuf.Value: unsupported type %T", v)
		}
	}
	resultVal, err := structpb.NewValue(raw)
	if err != nil {
		return nil, fmt.Errorf("failed to convert result to protobuf.Value: %w", err)
	}
	return resultVal, nil
}

// celParseJson is a CEL custom function to parse JSON from string.
func celParseJson(arg ref.Val) ref.Val {
	if arg.Type() != types.StringType {
		return types.MaybeNoSuchOverloadErr(arg)
	}
	str, ok := arg.Value().(string)
	if !ok {
		return types.NewErr("bug: arg is string type, but cannot convert")
	}
	var result interface{}
	err := json.Unmarshal([]byte(str), &result)
	if err != nil {
		return types.NewErr(fmt.Sprintf("failed to unmarshal JSON: %s", err.Error()))
	}
	return types.DefaultTypeAdapter.NativeToValue(result)
}
