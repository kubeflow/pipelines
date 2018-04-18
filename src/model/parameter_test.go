// Copyright 2018 Google LLC
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

package model

import (
	"testing"

	"github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	"github.com/magiconair/properties/assert"
)

func TestToParameters(t *testing.T) {
	v1 := "abc"
	argoParam := []v1alpha1.Parameter{{Name: "parameter1"}, {Name: "parameter2", Value: &v1}}
	expectParam := []Parameter{
		{Name: "parameter1"},
		{Name: "parameter2", Value: &v1}}
	assert.Equal(t, expectParam, ToParameters(argoParam))
}
