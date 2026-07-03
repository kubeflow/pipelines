// Copyright 2025 The Kubeflow Authors
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

package util

import (
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

// ValidateLiteralParameter validates a resolved parameter value against
// the literal constraints defined in the parameter spec.
// Returns nil if validation passes or if there are no literal constraints.
func ValidateLiteralParameter(
	paramName string,
	value *structpb.Value,
	literals []*structpb.Value,
) error {
	if len(literals) == 0 {
		// No literal constraint
		return nil
	}

	for _, literal := range literals {
		if proto.Equal(value, literal) {
			return nil
		}
	}

	return fmt.Errorf(
		"input parameter %q value %v does not match any of the allowed literal values: %v",
		paramName, value, literals,
	)
}
