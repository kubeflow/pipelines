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
	"github.com/kubeflow/pipelines/api/v2alpha1/go/pipelinespec"
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
	selectEnv    *cel.Env
	conditionEnv *cel.Env
}

func New() (*Expr, error) {
	selectEnv, err := cel.NewEnv(
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
	conditionEnv, err := cel.NewEnv(
		cel.DeclareContextProto((new(pipelinespec.ExecutorInput).ProtoReflect().Descriptor())),
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
	return &Expr{selectEnv: selectEnv, conditionEnv: conditionEnv}, nil
}

// Select from a protobuf.Value using a CEL expression.
// Fields of protobuf.Value can be directly referenced in the select expression.
func (e *Expr) Select(v *structpb.Value, expr string) (*structpb.Value, error) {
	ast, issues := e.selectEnv.Compile(expr)
	if issues != nil && issues.Err() != nil {
		return nil, issues.Err()
	}
	program, err := e.selectEnv.Program(ast, cel.Functions(
		&functions.Overload{
			Operator: parseJsonCelOverloadID,
			Unary:    celParseJson,
		},
	))
	if err != nil {
		return nil, err
	}
	// Add the protobuf.Value field that is set.
	// TODO(Bobgy): should we add unset fields as default values?
	fields, err := msgToCELMap(v, true)
	if err != nil {
		return nil, err
	}
	if v != nil {
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

// Condition evaluates condition expression against executor input, it returns
// whether the task should run. Fields of ExecutorInput can be directly referred
// in condition.
// When this returns:
// * true -> run the task
// * false -> skip the task
func (e *Expr) Condition(input *pipelinespec.ExecutorInput, condition string) (bool, error) {
	ast, issues := e.conditionEnv.Compile(condition)
	if issues != nil && issues.Err() != nil {
		return false, issues.Err()
	}
	program, err := e.conditionEnv.Program(ast, cel.Functions(
		&functions.Overload{
			Operator: parseJsonCelOverloadID,
			Unary:    celParseJson,
		},
	))
	if err != nil {
		return false, err
	}
	fields, err := msgToCELMap(input, false)
	if err != nil {
		return false, err
	}
	result, _, err := program.Eval(fields)
	if err != nil {
		return false, fmt.Errorf("evaluation error: %w", err)
	}
	raw := result.Value()
	res, isBool := raw.(bool)
	if !isBool {
		return false, fmt.Errorf("evaluation error: return type is not bool, got %+v", raw)
	}
	return res, nil
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

// Convert a proto message into a map[string]interface{} for CEL.
func msgToCELMap(m proto.Message, skipUnsetFields bool) (map[string]interface{}, error) {
	var err error
	fields := make(map[string]interface{})
	if m == nil {
		return fields, nil
	}
	if skipUnsetFields {
		m.ProtoReflect().Range(func(fd protoreflect.FieldDescriptor, _ protoreflect.Value) bool {
			var raw interface{}
			raw, err = pb.NewFieldDescription(fd).GetFrom(m)
			if err != nil {
				return false
			}
			fields[fd.TextName()] = raw
			return true
		})
		if err != nil {
			return nil, fmt.Errorf("failed to convert proto message to CEL map: %w", err)
		}
	} else {
		// Bind default values for unset fields.
		fds := m.ProtoReflect().Descriptor().Fields()
		for i := 0; i < fds.Len(); i++ {
			fd := fds.Get(i)
			// When the field is unset, GetFrom returns the default.
			raw, err := pb.NewFieldDescription(fd).GetFrom(m)
			if err != nil {
				return nil, fmt.Errorf("failed to convert proto message to CEL map: %w", err)
			}
			fields[fd.TextName()] = raw
		}
	}
	return fields, nil
}
