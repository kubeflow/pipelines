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

package storage

import (
	"testing"

	"github.com/googleprivate/ml/backend/src/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

func getFakeModelToken(sortByFieldName string) (string, error) {
	model := FakeListableModel{Name: "foo", Author: "bar"}
	return toNextPageToken(sortByFieldName, model)
}

func TestNewPaginationContext(t *testing.T) {
	token, err := getFakeModelToken("Author")
	assert.Nil(t, err)
	request, err := NewPaginationContext(token, 3, "Author", "Name")
	expected := &PaginationContext{
		pageSize:        3,
		sortByFieldName: "Author",
		keyFieldName:    "Name",
		token:           &Token{SortByFieldValue: "bar", KeyFieldValue: "foo"}}
	assert.Nil(t, err)
	assert.Equal(t, expected, request)
}

func TestNewPaginationContext_NegativePageSizeError(t *testing.T) {
	token, err := getFakeModelToken("Author")
	assert.Nil(t, err)
	_, err = NewPaginationContext(token, -1, "Author", "Name")
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestNewPaginationContext_DefaultPageSize(t *testing.T) {
	token, err := getFakeModelToken("Author")
	assert.Nil(t, err)
	request, err := NewPaginationContext(token, 0, "Author", "Name")
	expected := &PaginationContext{
		pageSize:        defaultPageSize,
		sortByFieldName: "Author",
		keyFieldName:    "Name",
		token:           &Token{SortByFieldValue: "bar", KeyFieldValue: "foo"}}
	assert.Nil(t, err)
	assert.Equal(t, expected, request)
}

func TestNewPaginationContext_DefaultSorting(t *testing.T) {
	token, err := getFakeModelToken("Name")
	assert.Nil(t, err)
	request, err := NewPaginationContext(token, 3, "", "Name")
	expected := &PaginationContext{
		pageSize:        3,
		sortByFieldName: "Name",
		keyFieldName:    "Name",
		token:           &Token{SortByFieldValue: "foo", KeyFieldValue: "foo"}}
	assert.Nil(t, err)
	assert.Equal(t, expected, request)
}

func TestNewPaginationContext_InvalidToken(t *testing.T) {
	_, err := NewPaginationContext("invalid token", 3, "", "Name")
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}
