package list

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"

	sq "github.com/Masterminds/squirrel"
	api "github.com/kubeflow/pipelines/backend/api/go_client"
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
type token struct {
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

func (t *token) unmarshal(pageToken string) error {
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

func (t *token) marshal() (string, error) {
	b, err := json.Marshal(t)
	if err != nil {
		return "", util.NewInternalServerError(err, "Failed to serialize page token.")
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

type Options struct {
	PageSize int
	*token
}

func NewOptionsFromToken(serializedToken string, pageSize int) (*Options, error) {
	pageSize, err := validatePageSize(pageSize)
	if err != nil {
		return nil, err
	}

	t := &token{}
	if err := t.unmarshal(serializedToken); err != nil {
		return nil, err
	}
	return &Options{PageSize: pageSize, token: t}, nil
}

// NewOptions
func NewOptions(listable Listable, pageSize int, sortBy string, filterProto *api.Filter) (*Options, error) {
	pageSize, err := validatePageSize(pageSize)
	if err != nil {
		return nil, err
	}

	token := &token{KeyFieldName: listable.PrimaryKeyColumnName()}

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

	// Filtering.
	if filterProto != nil {
		f, err := filter.NewWithKeyMap(filterProto, listable.APIToModelFieldMap())
		if err != nil {
			return nil, err
		}
		token.Filter = f
	}

	return &Options{PageSize: pageSize, token: token}, nil
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

	if o.Filter != nil {
		sqlBuilder = o.Filter.AddToSelect(sqlBuilder)
	}

	return sqlBuilder.ToSql()
}

// Listable ...
type Listable interface {
	PrimaryKeyColumnName() string
	DefaultSortField() string
	APIToModelFieldMap() map[string]string
}

const (
	defaultPageSize = 20
	maxPageSize     = 200
)

// NextPageToken ...
func (o *Options) NextPageToken(listable Listable) (string, error) {
	t, err := o.nextPageToken(listable)
	if err != nil {
		return "", err
	}
	return t.marshal()
}

func (o *Options) nextPageToken(listable Listable) (*token, error) {
	elem := reflect.ValueOf(listable).Elem()
	elemName := elem.Type().Name()

	sortByField := elem.FieldByName(o.SortByFieldName)
	if !sortByField.IsValid() {
		return nil, fmt.Errorf("cannot sort by field %q on type %q", o.SortByFieldName, elemName)
	}

	keyField := elem.FieldByName(listable.PrimaryKeyColumnName())
	if !keyField.IsValid() {
		return nil, fmt.Errorf("type %q does not have key field %q", elemName, o.KeyFieldName)
	}

	return &token{
		SortByFieldName:  o.SortByFieldName,
		SortByFieldValue: sortByField.Interface(),
		KeyFieldName:     listable.PrimaryKeyColumnName(),
		KeyFieldValue:    keyField.Interface(),
		IsDesc:           o.IsDesc,
	}, nil
}

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
