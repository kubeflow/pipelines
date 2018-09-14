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
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/googleprivate/ml/backend/src/apiserver/common"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
)

type FakeListableModel struct {
	Name        string
	Author      string
	Description string
}

func (FakeListableModel) GetKeyName() string {
	return "name"
}

func (m FakeListableModel) GetValueOfPrimaryKey() string {
	return m.Name
}

func fooListInternal(request *common.PaginationContext) ([]model.ListableDataModel, error) {
	return []model.ListableDataModel{
		FakeListableModel{Name: "a_name", Author: "a_author"},
		FakeListableModel{Name: "b_name", Author: "b_author"}}, nil
}

func fooBadListInternal(request *common.PaginationContext) ([]model.ListableDataModel, error) {
	return nil, util.NewInvalidInputError("some error")
}

func TestList(t *testing.T) {
	request := &common.PaginationContext{PageSize: 1, SortByFieldName: "name", KeyFieldName: "name", Token: nil}
	models, token, err := listModel(request, fooListInternal)
	assert.Nil(t, err)
	assert.Equal(t, []model.ListableDataModel{FakeListableModel{Name: "a_name", Author: "a_author"}}, models)
	expectedToken, err := toNextPageToken("name", FakeListableModel{Name: "b_name", Author: "b_author"})
	assert.Nil(t, err)
	assert.Equal(t, expectedToken, token)
}

func TestList_ListedAll(t *testing.T) {
	request := &common.PaginationContext{PageSize: 2, SortByFieldName: "name", Token: nil}
	models, token, err := listModel(request, fooListInternal)
	assert.Nil(t, err)
	assert.Equal(t, []model.ListableDataModel{
		FakeListableModel{Name: "a_name", Author: "a_author"},
		FakeListableModel{Name: "b_name", Author: "b_author"}}, models)
	assert.Equal(t, "", token)
}

func TestList_ListInternalError(t *testing.T) {
	request := &common.PaginationContext{PageSize: 2, SortByFieldName: "name", Token: nil}
	_, _, err := listModel(request, fooBadListInternal)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestToNextPageToken(t *testing.T) {
	model := FakeListableModel{Name: "foo", Author: "bar"}
	token, err := toNextPageToken("Author", model)
	assert.Nil(t, err)
	expectedJson, _ := json.Marshal(common.Token{
		SortByFieldValue: "bar",
		KeyFieldValue:    "foo",
	})
	assert.Equal(t, base64.StdEncoding.EncodeToString(expectedJson), token)
}
