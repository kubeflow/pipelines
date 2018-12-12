package list

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	sq "github.com/Masterminds/squirrel"
	"github.com/kubeflow/pipelines/backend/src/apiserver/filter"
	"github.com/kubeflow/pipelines/backend/src/common/util"
)

// Token represents a token for obtaining the next page in a
// ListXXX request. Assuming the list request is sorted by
// name, a typical token should be
//
//     {SortByFieldValue:"foo", KeyFieldValue:"2"}
// The corresponding list query would be select * from table where (name, id)
// >=(foobar,2) order by name, id limit page_size
type Token struct {
	// SortByFieldName
	SortByFieldName string
	// The value of the sorted field of the next row to be returned.
	SortByFieldValue interface{}
	// The value
	KeyFieldName string
	// The value of the key field of the next row to be returned.
	KeyFieldValue interface{}
	// IsDesc
	IsDesc bool
	// Filter ...
	Filter *filter.Filter
}

func (t *Token) Unmarshal(pageToken string) error {
	errorF := func(err error) error {
		return util.NewInvalidInputErrorWithDetails(err, "Invalid package token.")
	}
	b, err := base64.StdEncoding.DecodeString(pageToken)
	if err != nil {
		return errorF(err)
	}

	if err = json.Unmarshal(b, t); err != nil {
		return errorF(err)
	}

	return nil
}

func (t *Token) Marshal() (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return "", util.NewInternalServerError(err, "Failed to serialize page token.")
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

type Options struct {
	PageSize int
	*Token
}

func (o *Options) BuildListSQLQuery(table string) (string, []interface{}, error) {
	sqlBuilder := sq.Select("*").From(table)

	// If next row's value is specified, set those values in the clause.
	if o.SortByFieldValue != nil && o.KeyFieldValue != nil {
		if o.IsDesc {
			sqlBuilder = sqlBuilder.
				Where(sq.Or{sq.Lt{o.SortByFieldName: o.SortByFieldValue},
					sq.And{sq.Eq{o.SortByFieldName: o.SortByFieldValue}, sq.LtOrEq{o.KeyFieldName: o.KeyFieldValue}}})
		} else {
			sqlBuilder = sqlBuilder.
				Where(sq.Or{sq.Gt{o.SortByFieldName: o.SortByFieldValue},
					sq.And{sq.Eq{o.SortByFieldName: o.SortByFieldValue}, sq.GtOrEq{o.KeyFieldName: o.KeyFieldValue}}})
		}
	}

	order := "ASC"
	if o.IsDesc {
		order = "DESC"
	}
	sqlBuilder = sqlBuilder.
		OrderBy(fmt.Sprintf("%v %v", o.SortByFieldName, order)).
		OrderBy(fmt.Sprintf("%v %v", o.KeyFieldName, order))

	// Add one more item than what is requested.
	sqlBuilder = sqlBuilder.Limit(uint64(o.PageSize + 1))

	return sqlBuilder.ToSql()
}

// Listable ...
type Listable interface {
	PrimaryKeyColumnName() string
	// Get the value of the key field.
	PrimaryKeyValue() string
	DefaultSortField() string
	APIToModelFieldMap() map[string]string
}

const (
	defaultPageSize = 20
	maxPageSize     = 200
)

func validatePageSize(pageSize int) (int, error) {
	if pageSize < 0 {
		return 0, util.NewInvalidInputError("The page size should be greater than 0. Got %q", pageSize)
	}

	if pageSize == 0 {
		// Use default page size if not provided.
		return defaultPageSize, nil
	}

	if pageSize > maxPageSize {
		return maxPageSize, nil
	}

	return pageSize, nil
}

// NextPageToken ...
func NextPageToken(opts *Options, listable Listable) *Token {
	return &Token{
		SortByFieldName:  opts.SortByFieldName,
		SortByFieldValue: reflect.ValueOf(listable).FieldByName(opts.SortByFieldName),
		KeyFieldName:     opts.KeyFieldName,
		KeyFieldValue:    listable.PrimaryKeyValue,
		IsDesc:           opts.IsDesc,
	}
}

// NewOptions ...
func NewOptions(serializedToken string, pageSize int, sortBy string, listable Listable) (*Options, error) {
	pageSize, err := validatePageSize(pageSize)
	if err != nil {
		return nil, err
	}

	if serializedToken != "" {
		t := &Token{}
		if err := t.Unmarshal(serializedToken); err != nil {
			return nil, err
		}
		return &Options{PageSize: pageSize, Token: t}, nil
	}

	// Build a new one.
	token := &Token{KeyFieldName: listable.PrimaryKeyColumnName()}

	// Ignore the case of the letter. Split query string by space.
	queryList := strings.Fields(strings.ToLower(sortBy))
	// Check the query string format.
	if len(queryList) > 2 || (len(queryList) == 2 && queryList[1] != "desc" && queryList[1] != "asc") {
		return nil, util.NewInvalidInputError(
			"Received invalid sort by format %q. Supported format: \"field_name\", \"field_name desc\", or \"field_name asc\"", sortBy)
	}

	token.SortByFieldName = listable.DefaultSortField()
	if len(queryList) > 0 {
		var err error
		n, ok := listable.APIToModelFieldMap()[queryList[0]]
		if !ok {
			return nil, util.NewInvalidInputError("Invalid sorting field: %q: %s", queryList[0], err)
		}
		token.SortByFieldName = n
	}

	if len(queryList) == 2 {
		token.IsDesc = queryList[1] == "desc"
	}

	return &Options{PageSize: pageSize, Token: token}, nil
}

// func (s *ExperimentStore) ListExperiments(opts *common.PaginationContext) ([]model.Experiment, string, error) {
// 	rows, pageResults, err := listQuery(s.db, opts, "experiments")
// 	defer rows.Close()

// 	if err != nil {
// 		return nil, "", err
// 	}

// 	exps, err := s.scanRows(rows)
// 	if err != nil {
// 		return nil, "", err
// 	}

// 	if len(exps) <= opts.PageSize {
// 		return exps, "", nil
// 	}

// 	token, err := makePageToken(exps[pageSize], opts)
// 	if err != nil {
// 		return nil, "", err
// 	}

// 	return exps[:pageSize], token, nil
// }

// func validatePageSize(pageSize int) (int, error) {
// 	if pageSize < 0 {
// 		return 0, util.NewInvalidInputError("The page size should be greater than 0. Got %q", pageSize)
// 	}

// 	if pageSize == 0 {
// 		// Use default page size if not provided.
// 		return defaultPageSize, nil
// 	}

// 	if pageSize > maxPageSize {
// 		return maxPageSize, nil
// 	}

// 	return pageSize, nil
// }

// func ValidateListOptions(string, pageSize int, sortBy string, listable model.Listable) (*common.ListOptions, error) {
// 	pageSize, err := validatePageSize(pageSize)
// 	if err != nil {
// 		return nil, err
// 	}

// 	if serializedToken != "" {
// 		t := &common.Token{}
// 		if err := t.Unmarshal(serializedToken); err != nil {
// 			return nil, err
// 		}
// 		return &common.PaginationContext{PageSize: pageSize, Token: t}, nil
// 	}

// 	// Build a new one.
// 	token := &common.Token{PrimaryKeyName: listable.PrimaryKeyColumnName()}

// 	// Ignore the case of the letter. Split query string by space.
// 	queryList := strings.Fields(strings.ToLower(sortBy))
// 	// Check the query string format.
// 	if len(queryList) > 2 || (len(queryList) == 2 && queryList[1] != "desc" && queryList[1] != "asc") {
// 		return nil, util.NewInvalidInputError(
// 			"Received invalid sort by format %q. Supported format: \"field_name\", \"field_name desc\", or \"field_name asc\"", sortBy)
// 	}

// 	token.SortByFieldName = listable.DefaultSortField()
// 	if len(queryList) > 0 {
// 		var err error
// 		token.SortByFieldName, err = listable.SortableFieldFromAPI(queryList[-1])
// 		if err != nil {
// 			return nil, util.NewInvalidInputError("Invalid sorting field: %q: %s", queryList[1], err)
// 		}
// 	}

// 	if len(queryList) == 2 {
// 		token.IsDesc = queryList[1] == "desc"
// 	}

// 	return &common.PaginationContext{PageSize: pageSize, Token: token}, nil
// }

// package storage

// import (
// 	"database/sql"
// 	"encoding/base64"
// 	"encoding/json"
// 	"fmt"
// 	"reflect"

// 	sq "github.com/Masterminds/squirrel"
// 	"github.com/kubeflow/pipelines/backend/src/apiserver/common"
// 	"github.com/kubeflow/pipelines/backend/src/apiserver/model"
// )

// // Delegate to query the model table for a list of models.
// type QueryListableModelTable func(request *common.PaginationContext) ([]model.ListableDataModel, error)

// type readRowsF func(rows *sql.Rows) ([]model.Listable, error)

// func listQuery(db *DB, opts *common.PaginationContext, table string) (*sql.Rows, error) {
// 	sqlBuilder := sq.Select("*").From(table)

// 	// Add sorting.
// 	sqlBuilder = addSortingOptions(sqlBuilder, opts)

// 	// Add one more item than what is requested.
// 	sqlBuilder = sqlBuilder.Limit(uint64(opts.PageSize + 1))

// 	sql, args, err := sqlBuilder.ToSql()

// 	if err != nil {
// 		return nil, util.NewInternalServerError(err, "Failed to create query")
// 	}

// 	return db.Query(sql, args...)
// }

// // 	if err != nil {
// // 		return nil, util.NewInternalServerError(err, "Failed to execute query")
// // 	}

// // 	results, err := readRows(rows)
// // 	if err != nil {
// // 		return nil, util.NewInternalServerError(err, "Failed to read query results")
// // 	}

// // 	if len(results) < newContext.PageSize {
// // 		return results, "", nil
// // 	}

// // 	tokenString, err := toNextPageToken(context.SortByFieldName, results[context.PageSize])
// // 	if err != nil {
// // 		return nil, "", util.Wrap(err, "Failed to create page token")
// // 	}
// // 	return results[:len(results)-1], tokenString, nil
// // }

// // Logic for listing ListableModels. This would attempt to query one more model than requested,
// // in order to generate the page token. If the result is less than request+1, the nextPageToken
// // will be empty
// func listResults(on.PaginationContext, queryTable QueryListableModelTable) (models []model.ListableDataModel, nextPageToken string, err error) {
// 	newContext := *context
// 	// List one more item to generate next page token.
// 	newContext.PageSize = context.PageSize + 1
// 	results, err := queryTable(&newContext)
// 	if err != nil {
// 		return nil, "", util.Wrap(err, "List data model failed.")
// 	}
// 	if len(results) < newContext.PageSize {
// 		return results, "", nil
// 	}
// 	tokenString, err := toNextPageToken(context.SortByFieldName, results[context.PageSize])
// 	if err != nil {
// 		return nil, "", util.Wrap(err, "Failed to create page token")
// 	}
// 	return results[:len(results)-1], tokenString, nil
// }

// // Generate page token given the first model to be listed in the next page.
// func toNextPageToken(sortByFieldName string, model model.ListableDataModel) (string, error) {
// 	newToken := common.Token{
// 		SortByFieldValue: fmt.Sprint(reflect.ValueOf(model).FieldByName(sortByFieldName)),
// 		KeyFieldValue:    model.GetValueOfPrimaryKey(),
// 	}

// 	tokenBytes, err := json.Marshal(newToken)
// 	if err != nil {
// 		return "", util.NewInternalServerError(err, "Failed to serialize page token.")
// 	}
// 	return base64.StdEncoding.EncodeToString(tokenBytes), nil
// }

// // If the PaginationContext is
// // {sortByFieldName "name", keyFieldName:"id", token: {SortByFieldValue: "foo", KeyFieldValue: "2"}}
// // This function construct query as something like
// // select * from table where (name, id)>=("foo","2") order by name, id
// func addSortingCondition(selectBuilder sq.SelectBuilder, opts *common.PaginationContext) sq.SelectBuilder {
// 	if opts.SortByFieldValue != nil && opts.PrimaryKeyValue != nil {
// 		if context.IsDesc {
// 			selectBuilder = selectBuilder.
// 				Where(sq.Or{sq.Lt{opts.SortByFieldName: opts.SortByFieldValue},
// 					sq.And{sq.Eq{opts.SortByFieldName: opts.SortByFieldValue}, sq.LtOrEq{opts.KeyFieldName: opts.KeyFieldValue}}})
// 		} else {
// 			selectBuilder = selectBuilder.
// 				Where(sq.Or{sq.Gt{opts.SortByFieldName: opts.SortByFieldValue},
// 					sq.And{sq.Eq{opts.SortByFieldName: opts.SortByFieldValue}, sq.GtOrEq{opts.KeyFieldName: opts.KeyFieldValue}}})
// 		}
// 	}

// 	order := "ASC"
// 	if opts.IsDesc {
// 		order = "DESC"
// 	}
// 	selectBuilder = selectBuilder.
// 		OrderBy(fmt.Sprintf("%v %v", opts.SortByFieldName, order)).
// 		OrderBy(fmt.Sprintf("%v %v", opts.KeyFieldName, order))
// 	return selectBuilder
// }
