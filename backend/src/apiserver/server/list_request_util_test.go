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

package server

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/googleprivate/ml/backend/api/go_client"
	"github.com/googleprivate/ml/backend/src/apiserver/common"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

var fakeModelFieldsBySortableAPIFields = map[string]string{
	"":            "Name",
	"name":        "Name",
	"author":      "Author",
	"description": "Description",
}

func getFakeModelToken() string {
	token := common.Token{
		SortByFieldValue: "bar",
		KeyFieldValue:    "foo",
	}
	expectedJson, _ := json.Marshal(token)
	return base64.StdEncoding.EncodeToString(expectedJson)
}

func TestValidateFilter(t *testing.T) {
	referenceKey := &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: "123"}
	ctx, err := ValidateFilter(referenceKey)
	assert.Nil(t, err)
	assert.Equal(t, &common.FilterContext{ReferenceKey: &common.ReferenceKey{Type: common.Experiment, ID: "123"}}, ctx)
}

func TestValidateFilter_ToModelResourceTypeFailed(t *testing.T) {
	referenceKey := &api.ResourceKey{Type: api.ResourceType_UNKNOWN_RESOURCE_TYPE, Id: "123"}
	_, err := ValidateFilter(referenceKey)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Unrecognized resource reference type")
}

func TestValidatePagination(t *testing.T) {
	token := getFakeModelToken()
	context, err := ValidatePagination(token, 3, "Name",
		"", fakeModelFieldsBySortableAPIFields)
	assert.Nil(t, err)
	expected := &common.PaginationContext{
		PageSize:        3,
		SortByFieldName: "Name",
		KeyFieldName:    "Name",
		Token:           &common.Token{SortByFieldValue: "bar", KeyFieldValue: "foo"}}
	assert.Equal(t, expected, context)
}

func TestValidatePagination_NegativePageSizeError(t *testing.T) {
	token := getFakeModelToken()
	_, err := ValidatePagination(token, -1, "Name",
		"", fakeModelFieldsBySortableAPIFields)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestValidatePagination_DefaultPageSize(t *testing.T) {
	token := getFakeModelToken()
	context, err := ValidatePagination(token, 0, "Name",
		"", fakeModelFieldsBySortableAPIFields)
	expected := &common.PaginationContext{
		PageSize:        defaultPageSize,
		SortByFieldName: "Name",
		KeyFieldName:    "Name",
		Token:           &common.Token{SortByFieldValue: "bar", KeyFieldValue: "foo"}}
	assert.Nil(t, err)
	assert.Equal(t, expected, context)
}

func TestValidatePagination_DefaultSorting(t *testing.T) {
	token := getFakeModelToken()
	context, err := ValidatePagination(token, 0, "Name",
		"", fakeModelFieldsBySortableAPIFields)
	expected := &common.PaginationContext{
		PageSize:        defaultPageSize,
		SortByFieldName: "Name",
		KeyFieldName:    "Name",
		Token:           &common.Token{SortByFieldValue: "bar", KeyFieldValue: "foo"}}
	assert.Nil(t, err)
	assert.Equal(t, expected, context)
}

func TestValidatePagination_InvalidToken(t *testing.T) {
	_, err := ValidatePagination("invalid token", 0, "",
		"", fakeModelFieldsBySortableAPIFields)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestDeserializePageToken(t *testing.T) {
	token := common.Token{
		SortByFieldValue: "bar",
		KeyFieldValue:    "foo",
	}
	expectedJson, _ := json.Marshal(token)
	tokenString := base64.StdEncoding.EncodeToString(expectedJson)
	actualToken, err := deserializePageToken(tokenString)
	assert.Nil(t, err)
	assert.Equal(t, token, *actualToken)
}

func TestDeserializePageToken_InvalidEncodingStringError(t *testing.T) {
	_, err := deserializePageToken("this is a invalid token")
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestDeserializePageToken_UnmarshalError(t *testing.T) {
	_, err := deserializePageToken(base64.StdEncoding.EncodeToString([]byte("invalid token")))
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestParseSortByQueryString_EmptyString(t *testing.T) {
	modelField, isDesc, err := parseSortByQueryString("", fakeModelFieldsBySortableAPIFields)
	assert.Nil(t, err)
	assert.Equal(t, "Name", modelField)
	assert.False(t, isDesc)
}

func TestParseSortByQueryString_FieldNameOnly(t *testing.T) {
	modelField, isDesc, err := parseSortByQueryString("Name", fakeModelFieldsBySortableAPIFields)
	assert.Nil(t, err)
	assert.Equal(t, "Name", modelField)
	assert.False(t, isDesc)
}

func TestParseSortByQueryString_FieldNameWithDescFlag(t *testing.T) {
	modelField, isDesc, err := parseSortByQueryString("Name desc", fakeModelFieldsBySortableAPIFields)
	assert.Nil(t, err)
	assert.Equal(t, "Name", modelField)
	assert.True(t, isDesc)
}

func TestParseSortByQueryString_FieldNameWithAscFlag(t *testing.T) {
	modelField, isDesc, err := parseSortByQueryString("Name asc", fakeModelFieldsBySortableAPIFields)
	assert.Nil(t, err)
	assert.Equal(t, "Name", modelField)
	assert.False(t, isDesc)
}

func TestParseSortByQueryString_NotSortableFieldName(t *testing.T) {
	_, _, err := parseSortByQueryString("foobar", fakeModelFieldsBySortableAPIFields)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Cannot sort on field foobar.")
}

func TestParseSortByQueryString_IncorrectDescFlag(t *testing.T) {
	_, _, err := parseSortByQueryString("id foobar", fakeModelFieldsBySortableAPIFields)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Received invalid sort by format `id foobar`")
}

func TestParseSortByQueryString_StringTooLong(t *testing.T) {
	_, _, err := parseSortByQueryString("Name desc foo", fakeModelFieldsBySortableAPIFields)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Received invalid sort by format `Name desc foo`")
}
