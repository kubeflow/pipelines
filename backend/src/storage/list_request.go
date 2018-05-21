package storage

import (
	"ml/backend/src/util"
	"strconv"
)

const (
	defaultPageSize = 200
	maxPageSize     = 200
)

type PaginationContext struct {
	pageSize        int
	sortByFieldName string
	keyFieldName    string
	token           *Token
}

// NewPaginationContext create a new list request, along with validating the list inputs, such as tokens.
func NewPaginationContext(pageToken string, pageSize int, sortByFieldName string, keyFieldName string) (*PaginationContext, error) {
	if pageSize < 0 {
		return nil, util.NewInvalidInputError("The page size should be greater than 0. Got %v", strconv.Itoa(pageSize))
	}
	if pageSize == 0 {
		// Use default page size if not provided.
		pageSize = defaultPageSize
	}
	if pageSize > defaultPageSize {
		pageSize = maxPageSize
	}
	if sortByFieldName == "" {
		// By default, sort by key field.
		sortByFieldName = keyFieldName
	}
	token, err := deserializePageToken(pageToken)
	if err != nil {
		return nil, util.Wrap(err, "Invalid page token.")
	}
	return &PaginationContext{
		pageSize:        pageSize,
		sortByFieldName: sortByFieldName,
		keyFieldName:    keyFieldName,
		token:           token}, nil
}
