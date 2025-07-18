// Package api
// Copyright 2018-2023 The Kubeflow Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package api

import (
	"fmt"
	"reflect"

	"github.com/onsi/gomega"
)

// MatchMaps - Iterate over 2 maps and compare if they are same or not
func MatchMaps(actual interface{}, expected interface{}, mapType string) {
	expectedMap, _ := expected.(map[any]interface{})
	actualMap, _ := actual.(map[any]interface{})
	for key, value := range expectedMap {
		if reflect.TypeOf(value).Kind() == reflect.Map {
			expectedMapFromValue, _ := value.(map[any]interface{})
			actualMapFromValue, _ := actualMap[key].(map[any]interface{})
			MatchMaps(expectedMapFromValue, actualMapFromValue, mapType)
		}
		actualStringValue := fmt.Sprintf("%v", actualMap[key])
		expectedStringValue := fmt.Sprintf("%v", value)
		gomega.Expect(actualStringValue).To(gomega.Equal(expectedStringValue), fmt.Sprintf("'%s' key's value not matching for type %s", key, mapType))
	}
}
