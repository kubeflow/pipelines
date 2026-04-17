// Copyright 2025 The Kubeflow Authors
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

package util

import (
	"fmt"
	"testing"
	"time"

	workflowapi "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-openapi/strfmt"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStringPointer(t *testing.T) {
	result := StringPointer("hello")
	assert.NotNil(t, result)
	assert.Equal(t, "hello", *result)
}

func TestBoolPointer(t *testing.T) {
	resultTrue := BoolPointer(true)
	assert.NotNil(t, resultTrue)
	assert.Equal(t, true, *resultTrue)

	resultFalse := BoolPointer(false)
	assert.NotNil(t, resultFalse)
	assert.Equal(t, false, *resultFalse)
}

func TestTimePointer(t *testing.T) {
	now := time.Now()
	result := TimePointer(now)
	assert.NotNil(t, result)
	assert.Equal(t, now, *result)
}

func TestDateTimePointer(t *testing.T) {
	dateTime := strfmt.DateTime(time.Now())
	result := DateTimePointer(dateTime)
	assert.NotNil(t, result)
	assert.Equal(t, dateTime, *result)
}

func TestMetaV1TimePointer(t *testing.T) {
	metaTime := metav1.Now()
	result := MetaV1TimePointer(metaTime)
	assert.NotNil(t, result)
	assert.Equal(t, metaTime, *result)
}

func TestInt64Pointer(t *testing.T) {
	result := Int64Pointer(42)
	assert.NotNil(t, result)
	assert.Equal(t, int64(42), *result)
}

func TestUInt32Pointer(t *testing.T) {
	result := UInt32Pointer(100)
	assert.NotNil(t, result)
	assert.Equal(t, uint32(100), *result)
}

func TestInt32Pointer(t *testing.T) {
	result := Int32Pointer(-5)
	assert.NotNil(t, result)
	assert.Equal(t, int32(-5), *result)
}

func TestStringNilOrValue(t *testing.T) {
	assert.Equal(t, "<nil>", StringNilOrValue(nil))
	value := "test"
	assert.Equal(t, "test", StringNilOrValue(&value))
}

func TestInt64NilOrValue(t *testing.T) {
	assert.Equal(t, "<nil>", Int64NilOrValue(nil))
	value := int64(99)
	assert.Equal(t, "99", Int64NilOrValue(&value))
}

func TestBoolNilOrValue(t *testing.T) {
	assert.Equal(t, "<nil>", BoolNilOrValue(nil))
	value := true
	assert.Equal(t, "true", BoolNilOrValue(&value))
}

func TestBooleanPointer(t *testing.T) {
	result := BooleanPointer(true)
	assert.NotNil(t, result)
	assert.Equal(t, true, *result)
}

func TestMetav1TimePointer(t *testing.T) {
	metaTime := metav1.Now()
	result := Metav1TimePointer(metaTime)
	assert.NotNil(t, result)
	assert.Equal(t, metaTime, *result)
}

func TestToInt64Pointer(t *testing.T) {
	assert.Nil(t, ToInt64Pointer(nil))

	metaTime := metav1.NewTime(time.Unix(1000, 0))
	result := ToInt64Pointer(&metaTime)
	assert.NotNil(t, result)
	assert.Equal(t, int64(1000), *result)
}

func TestToAnyStringPointer(t *testing.T) {
	assert.Nil(t, ToAnyStringPointer(nil))

	value := "test-string"
	result := ToAnyStringPointer(&value)
	assert.NotNil(t, result)
	assert.Equal(t, "test-string", result.String())
}

func TestToStringPointer(t *testing.T) {
	assert.Nil(t, ToStringPointer(nil))

	anyString := workflowapi.AnyStringPtr("hello")
	result := ToStringPointer(anyString)
	assert.NotNil(t, result)
	assert.Equal(t, "hello", *result)
}

func TestAnyStringPtr(t *testing.T) {
	result := AnyStringPtr(42)
	assert.NotNil(t, result)
	assert.Equal(t, fmt.Sprintf("%v", 42), *result)

	resultStr := AnyStringPtr("abc")
	assert.Equal(t, "abc", *resultStr)
}
