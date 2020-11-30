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

	"github.com/google/go-cmp/cmp"

	api "github.com/kubeflow/pipelines/backend/api/go_client"
	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
	"github.com/kubeflow/pipelines/backend/src/apiserver/list"
	"github.com/kubeflow/pipelines/backend/src/common/util"
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

func TestParseAPIFilter_EmptyStringYieldsNilFilter(t *testing.T) {
	f, err := parseAPIFilter("")

	assert.Nil(t, err)
	assert.Nil(t, f)
}

func TestParseAPIFilter_InvalidStringYieldsError(t *testing.T) {
	f, err := parseAPIFilter("lkjlkjlkj")

	assert.NotNil(t, err)
	assert.Nil(t, f)
}

func TestParseAPIFilter_DecodesEncodedString(t *testing.T) {
	// in was generated by calling JSON.stringify followed by encodeURIComponent in
	// the browser on the following JSON string:
	//   {"predicates":[{"op":"EQUALS","key":"testkey","stringValue":"testvalue"}]}

	in := "%7B%22predicates%22%3A%5B%7B%22op%22%3A%22EQUALS%22%2C%22key%22%3A%22testkey%22%2C%22stringValue%22%3A%22testvalue%22%7D%5D%7D"

	// The above should correspond the following filter:
	want := &api.Filter{
		Predicates: []*api.Predicate{
			&api.Predicate{
				Key: "testkey", Op: api.Predicate_EQUALS,
				Value: &api.Predicate_StringValue{StringValue: "testvalue"},
			},
		},
	}

	got, err := parseAPIFilter(in)
	if !cmp.Equal(got, want) || err != nil {
		t.Errorf("parseAPIString(%q) =\nGot %+v, %v\n Want %+v, <nil>\nDiff: %s",
			in, got, err, want, cmp.Diff(want, got))
	}
}

type fakeListable struct {
	PrimaryKey       string
	FakeName         string
	CreatedTimestamp int64
}

func (f *fakeListable) PrimaryKeyColumnName() string {
	return "PrimaryKey"
}

func (f *fakeListable) DefaultSortField() string {
	return "CreatedTimestamp"
}

var fakeAPIToModelMap = map[string]string{
	"timestamp": "CreatedTimestamp",
	"name":      "FakeName",
	"id":        "PrimaryKey",
}

func (f *fakeListable) APIToModelFieldMap() map[string]string {
	return fakeAPIToModelMap
}

func (f *fakeListable) GetModelName() string {
	return ""
}

func (f *fakeListable) GetField(name string) (string, bool) {
	if field, ok := fakeAPIToModelMap[name]; ok {
		return field, true
	} else {
		return "", false
	}
}

func (f *fakeListable) GetFieldValue(name string) interface{} {
	switch name {
	case "CreatedTimestamp":
		return f.CreatedTimestamp
	case "FakeName":
		return f.FakeName
	case "PrimaryKey":
		return f.PrimaryKey
	}
	return nil
}

func (f *fakeListable) GetSortByFieldPrefix(name string) string {
	return ""
}

func (f *fakeListable) GetKeyFieldPrefix() string {
	return ""
}

func TestValidatedListOptions_Errors(t *testing.T) {
	opts, err := list.NewOptions(&fakeListable{}, 10, "name asc", nil)
	if err != nil {
		t.Fatalf("list.NewOptions() = _, %+v; Want nil error", err)
	}

	npt, err := opts.NextPageToken(&fakeListable{})
	if err != nil {
		t.Fatalf("opt.NextPageToken() = _, %+v; Want nil error", err)
	}

	_, err = validatedListOptions(&fakeListable{}, npt, 10, "name asc", "")
	if err != nil {
		t.Fatalf("validatedListOptions(fakeListable, 10, \"name asc\") = _, %+v; Want nil error", err)
	}

	_, err = validatedListOptions(&fakeListable{}, npt, 10, "name desc", "")
	if err == nil {
		t.Fatalf("validatedListOptions(fakeListable, 10, \"name desc\") = _, %+v; Want error", err)
	}
}
