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

// Package placeholder provides shared utilities for resolving KFP pipeline
// placeholders and converting structpb.Value instances to their canonical
// string representations. Both the driver and launcher layers use this
// package so that placeholder resolution and type stringification stay in
// one place.
package placeholder

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"

	"google.golang.org/protobuf/types/known/structpb"
)

// InputParameterRe matches the complete {{$.inputs.parameters['name']}}
// placeholder including the surrounding double braces.
var InputParameterRe = regexp.MustCompile(`\{\{\$\.inputs\.parameters\['(.+?)']}}`)

// PbValueToString converts a structpb.Value to its canonical string
// representation.  It handles STRING, NUMBER (integer and double), BOOLEAN,
// NULL, LIST, and STRUCT values.  The number formatting uses
// strconv.FormatFloat with 'f' format to match the launcher's established
// production behavior.
func PbValueToString(v *structpb.Value) (string, error) {
	if v == nil {
		return "", nil
	}
	switch t := v.Kind.(type) {
	case *structpb.Value_NullValue:
		return "", nil
	case *structpb.Value_StringValue:
		return v.GetStringValue(), nil
	case *structpb.Value_NumberValue:
		return strconv.FormatFloat(v.GetNumberValue(), 'f', -1, 64), nil
	case *structpb.Value_BoolValue:
		return strconv.FormatBool(v.GetBoolValue()), nil
	case *structpb.Value_ListValue:
		b, err := json.Marshal(v.GetListValue())
		if err != nil {
			return "", fmt.Errorf("failed to JSON-marshal list value: %w", err)
		}
		return string(b), nil
	case *structpb.Value_StructValue:
		b, err := json.Marshal(v.GetStructValue())
		if err != nil {
			return "", fmt.Errorf("failed to JSON-marshal struct value: %w", err)
		}
		return string(b), nil
	default:
		return "", fmt.Errorf("unknown structpb.Value type %T", t)
	}
}

// ResolveInputParameterPlaceholders performs template substitution on a
// string, replacing all {{$.inputs.parameters['name']}} occurrences with
// their resolved values from parameterValues.  It correctly handles both
// standalone placeholders and placeholders embedded in larger strings
// (e.g. "prefix-{{$.inputs.parameters['x']}}").
func ResolveInputParameterPlaceholders(arg string, parameterValues map[string]*structpb.Value) (string, error) {
	if !InputParameterRe.MatchString(arg) {
		return arg, nil
	}
	var resolveErr error
	result := InputParameterRe.ReplaceAllStringFunc(arg, func(match string) string {
		if resolveErr != nil {
			return match
		}
		submatch := InputParameterRe.FindStringSubmatch(match)
		if len(submatch) < 2 {
			resolveErr = fmt.Errorf("failed to extract parameter name from: %s", match)
			return match
		}
		paramName := submatch[1]
		val, ok := parameterValues[paramName]
		if !ok {
			resolveErr = fmt.Errorf("parameter %q not found in parameter values", paramName)
			return match
		}
		resolved, err := PbValueToString(val)
		if err != nil {
			resolveErr = fmt.Errorf("failed to convert parameter %q to string: %w", paramName, err)
			return match
		}
		return resolved
	})
	if resolveErr != nil {
		return "", resolveErr
	}
	return result, nil
}
