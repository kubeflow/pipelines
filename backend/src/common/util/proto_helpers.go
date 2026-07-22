// Copyright 2025 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"encoding/json"
	"fmt"
	"strconv"

	"google.golang.org/protobuf/types/known/structpb"
)

func PBValueToText(v *structpb.Value) (string, error) {
	wrap := func(err error) error {
		return fmt.Errorf("failed to convert protobuf.Value to text: %w", err)
	}
	if v == nil {
		return "", nil
	}
	var text string
	switch t := v.Kind.(type) {
	case *structpb.Value_NullValue:
		text = ""
	case *structpb.Value_StringValue:
		text = v.GetStringValue()
	case *structpb.Value_NumberValue:
		text = strconv.FormatFloat(v.GetNumberValue(), 'f', -1, 64)
	case *structpb.Value_BoolValue:
		text = strconv.FormatBool(v.GetBoolValue())
	case *structpb.Value_ListValue:
		b, err := json.Marshal(v.GetListValue())
		if err != nil {
			return "", wrap(fmt.Errorf("failed to JSON-marshal a list: %w", err))
		}
		text = string(b)
	case *structpb.Value_StructValue:
		b, err := json.Marshal(v.GetStructValue())
		if err != nil {
			return "", wrap(fmt.Errorf("failed to JSON-marshal a struct: %w", err))
		}
		text = string(b)
	default:
		return "", wrap(fmt.Errorf("unknown type %T", t))
	}
	return text, nil
}
