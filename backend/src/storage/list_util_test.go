package storage

import (
	"encoding/base64"
	"encoding/json"
	"ml/backend/src/model"
	"ml/backend/src/util"
	"testing"

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

func fooListInternal(request *PaginationContext) ([]model.ListableDataModel, error) {
	return []model.ListableDataModel{
		FakeListableModel{Name: "a_name", Author: "a_author"},
		FakeListableModel{Name: "b_name", Author: "b_author"}}, nil
}

func fooBadListInternal(request *PaginationContext) ([]model.ListableDataModel, error) {
	return nil, util.NewInvalidInputError("some error")
}

func TestList(t *testing.T) {
	request := &PaginationContext{pageSize: 1, sortByFieldName: "name", keyFieldName: "name", token: nil}
	models, token, err := listModel(request, fooListInternal)
	assert.Nil(t, err)
	assert.Equal(t, []model.ListableDataModel{FakeListableModel{Name: "a_name", Author: "a_author"}}, models)
	expectedToken, err := toNextPageToken("name", FakeListableModel{Name: "b_name", Author: "b_author"})
	assert.Nil(t, err)
	assert.Equal(t, expectedToken, token)
}

func TestList_ListedAll(t *testing.T) {
	request := &PaginationContext{pageSize: 2, sortByFieldName: "name", token: nil}
	models, token, err := listModel(request, fooListInternal)
	assert.Nil(t, err)
	assert.Equal(t, []model.ListableDataModel{
		FakeListableModel{Name: "a_name", Author: "a_author"},
		FakeListableModel{Name: "b_name", Author: "b_author"}}, models)
	assert.Equal(t, "", token)
}

func TestList_ListInternalError(t *testing.T) {
	request := &PaginationContext{pageSize: 2, sortByFieldName: "name", token: nil}
	_, _, err := listModel(request, fooBadListInternal)
	assert.Equal(t, codes.InvalidArgument, err.(*util.UserError).ExternalStatusCode())
}

func TestToNextPageToken(t *testing.T) {
	model := FakeListableModel{Name: "foo", Author: "bar"}
	token, err := toNextPageToken("Author", model)
	assert.Nil(t, err)
	expectedJson, _ := json.Marshal(Token{
		SortByFieldValue: "bar",
		KeyFieldValue:    "foo",
	})
	assert.Equal(t, base64.StdEncoding.EncodeToString(expectedJson), token)
}

func TestDeserializePageToken(t *testing.T) {
	token := Token{
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
