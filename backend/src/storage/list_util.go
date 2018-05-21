package storage

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"ml/backend/src/model"
	"ml/backend/src/util"
	"reflect"

	"github.com/jinzhu/gorm"
)

// A deserialized token. Assuming the list request is sorted by name, a typical token should be
// {SortByFieldValue:"foo", KeyFieldValue:"2"}
// The corresponding list query would be
// select * from table where (name, id) >=(foobar,2) order by name, id limit page_size
type Token struct {
	// The value of the sorted field of the next row to be returned.
	SortByFieldValue string
	// The value of the key field of the next row to be returned.
	KeyFieldValue string
}

// Delegate to query the model table for a list of models.
type QueryListableModelTable func(request *PaginationContext) ([]model.ListableDataModel, error)

// Logic for listing ListableModels. This would attempt to query one more model than requested,
// in order to generate the page token. If the result is less than request+1, the nextPageToken
// will be empty
func listModel(context *PaginationContext, queryTable QueryListableModelTable) (models []model.ListableDataModel, nextPageToken string, err error) {
	newContext := *context
	// List one more item to generate next page token.
	newContext.pageSize = context.pageSize + 1
	results, err := queryTable(&newContext)
	if err != nil {
		return nil, "", util.Wrap(err, "List data model failed.")
	}
	if len(results) < newContext.pageSize {
		return results, "", nil
	}
	tokenString, err := toNextPageToken(context.sortByFieldName, results[context.pageSize])
	if err != nil {
		return nil, "", util.Wrap(err, "Failed to create page token")
	}
	return results[:len(results)-1], tokenString, nil
}

// Generate page token given the first model to be listed in the next page.
func toNextPageToken(sortByFieldName string, model model.ListableDataModel) (string, error) {
	newToken := Token{
		SortByFieldValue: fmt.Sprint(reflect.ValueOf(model).FieldByName(sortByFieldName)),
		KeyFieldValue:    model.GetValueOfPrimaryKey(),
	}

	tokenBytes, err := json.Marshal(newToken)
	if err != nil {
		return "", util.NewInternalServerError(err, "Failed to serialize page token.")
	}
	return base64.StdEncoding.EncodeToString(tokenBytes), nil
}

// Decode page token. If page token is empty, we assume listing the first page and return a nil Token.
func deserializePageToken(pageToken string) (*Token, error) {
	if pageToken == "" {
		return nil, nil
	}
	tokenBytes, err := base64.StdEncoding.DecodeString(pageToken)
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Invalid package token.")
	}
	var token Token
	err = json.Unmarshal(tokenBytes, &token)
	if err != nil {
		return nil, util.NewInvalidInputErrorWithDetails(err, "Invalid package token.")
	}
	return &token, nil
}

// If the PaginationContext is
// {sortByFieldName "name", keyFieldName:"id", token: {SortByFieldValue: "foo", KeyFieldValue: "2"}}
// This function construct query as something like
// select * from table where (name, id)>=("foo","2") order by name, id
func toPaginationQuery(db *gorm.DB, context *PaginationContext) (*gorm.DB, error) {
	if token := context.token; token != nil {
		db = db.Where(
			fmt.Sprintf("(%v,%v)>=(?,?)", context.sortByFieldName, context.keyFieldName),
			token.SortByFieldValue, token.KeyFieldValue)
	}
	return db.Order(context.sortByFieldName).Order(context.keyFieldName), nil
}
