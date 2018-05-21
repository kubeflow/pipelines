package storage

import (
	"ml/backend/src/util"
	"testing"

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
