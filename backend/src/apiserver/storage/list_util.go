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
	"fmt"
	"reflect"

	sq "github.com/Masterminds/squirrel"
	"github.com/googleprivate/ml/backend/src/apiserver/common"
	"github.com/googleprivate/ml/backend/src/apiserver/model"
	"github.com/googleprivate/ml/backend/src/common/util"
)

// Delegate to query the model table for a list of models.
type QueryListableModelTable func(request *common.PaginationContext) ([]model.ListableDataModel, error)

// Logic for listing ListableModels. This would attempt to query one more model than requested,
// in order to generate the page token. If the result is less than request+1, the nextPageToken
// will be empty
func listModel(context *common.PaginationContext, queryTable QueryListableModelTable) (models []model.ListableDataModel, nextPageToken string, err error) {
	newContext := *context
	// List one more item to generate next page token.
	newContext.PageSize = context.PageSize + 1
	results, err := queryTable(&newContext)
	if err != nil {
		return nil, "", util.Wrap(err, "List data model failed.")
	}
	if len(results) < newContext.PageSize {
		return results, "", nil
	}
	tokenString, err := toNextPageToken(context.SortByFieldName, results[context.PageSize])
	if err != nil {
		return nil, "", util.Wrap(err, "Failed to create page token")
	}
	return results[:len(results)-1], tokenString, nil
}

// Generate page token given the first model to be listed in the next page.
func toNextPageToken(sortByFieldName string, model model.ListableDataModel) (string, error) {
	newToken := common.Token{
		SortByFieldValue: fmt.Sprint(reflect.ValueOf(model).FieldByName(sortByFieldName)),
		KeyFieldValue:    model.GetValueOfPrimaryKey(),
	}

	tokenBytes, err := json.Marshal(newToken)
	if err != nil {
		return "", util.NewInternalServerError(err, "Failed to serialize page token.")
	}
	return base64.StdEncoding.EncodeToString(tokenBytes), nil
}

// If the PaginationContext is
// {sortByFieldName "name", keyFieldName:"id", token: {SortByFieldValue: "foo", KeyFieldValue: "2"}}
// This function construct query as something like
// select * from table where (name, id)>=("foo","2") order by name, id
func toPaginationQuery(selectBuilder sq.SelectBuilder, context *common.PaginationContext) sq.SelectBuilder {
	if token := context.Token; token != nil {
		if context.IsDesc {
			selectBuilder = selectBuilder.
				Where(sq.Or{sq.Lt{context.SortByFieldName: token.SortByFieldValue},
					sq.And{sq.Eq{context.SortByFieldName: token.SortByFieldValue}, sq.LtOrEq{context.KeyFieldName: token.KeyFieldValue}}})
		} else {
			selectBuilder = selectBuilder.
				Where(sq.Or{sq.Gt{context.SortByFieldName: token.SortByFieldValue},
					sq.And{sq.Eq{context.SortByFieldName: token.SortByFieldValue}, sq.GtOrEq{context.KeyFieldName: token.KeyFieldValue}}})
		}
	}
	order := "ASC"
	if context.IsDesc {
		order = "DESC"
	}
	selectBuilder = selectBuilder.
		OrderBy(fmt.Sprintf("%v %v", context.SortByFieldName, order)).
		OrderBy(fmt.Sprintf("%v %v", context.KeyFieldName, order))
	return selectBuilder
}
