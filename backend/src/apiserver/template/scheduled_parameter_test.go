// Copyright 2018 The Kubeflow Authors
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

package template

import (
	"testing"

	"github.com/kubeflow/pipelines/backend/src/common/util"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestScheduledParameterMacrosPreserveNestedValues(t *testing.T) {
	value, err := structpb.NewValue(map[string]interface{}{
		"nested": []interface{}{"[[Index]]", map[string]interface{}{"time": "[[ScheduledTime]]", "run": "[[RunUUID]]"}},
		"number": float64(42), "boolean": true, "null": nil,
	})
	require.NoError(t, err)
	formatParameterMacros(value, util.NewSWFParameterFormatter("run-id", 100, 200, 3))
	require.Equal(t, map[string]interface{}{
		"nested": []interface{}{"3", map[string]interface{}{"time": "19700101000140", "run": "run-id"}},
		"number": float64(42), "boolean": true, "null": nil,
	}, value.AsInterface())
}
