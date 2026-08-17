package goqlite

import (
	"net/http"
)

// filter...

type Filter struct {
	And    []Filter             `json:"$and,omitempty"`
	Or     []Filter             `json:"$or,omitempty"`
	Not    *Filter              `json:"$not,omitempty"`
	Fields map[string]FieldExpr `json:"-"`
}

type FieldExprOp struct {
	Op    string      `json:"op,omitempty"`
	Value interface{} `json:"value,omitempty"`
}

type FieldExpr struct {
	Eq      interface{}   `json:"$eq,omitempty"`
	Ne      interface{}   `json:"$ne,omitempty"`
	Gt      interface{}   `json:"$gt,omitempty"`
	Gte     interface{}   `json:"$gte,omitempty"`
	Lt      interface{}   `json:"$lt,omitempty"`
	Lte     interface{}   `json:"$lte,omitempty"`
	In      []interface{} `json:"$in,omitempty"`
	Nin     []interface{} `json:"$nin,omitempty"`
	Like    string        `json:"$like,omitempty"`
	ILike   string        `json:"$ilike,omitempty"`
	Between []interface{} `json:"$between,omitempty"`
	Exists  *bool         `json:"$exists,omitempty"`
	IsNull  *bool         `json:"$null,omitempty"`

	Op *FieldExprOp `json:"$op,omitempty"`
}

// ...filter

// nested...

type NestedNode struct {
	Name   string
	Query  *QueryPayload
	Childs []*NestedNode
}

// ...nested

type QueryBuilder interface {
	Where(cond string, args ...interface{}) QueryBuilder
	And(sub QueryBuilder) QueryBuilder
	Or(sub QueryBuilder) QueryBuilder
	Not(sub QueryBuilder) QueryBuilder
	Build() interface{}
	Clone() QueryBuilder
}

type Order struct {
	Field string `json:"field"`
	Dir   string `json:"dir"` // asc | desc
}

// request...

type QueryPayload struct {
	Where  Filter   `json:"where,omitempty"`
	Order  []Order  `json:"sort,omitempty"`
	Select []string `json:"select,omitempty"`
	Nested string   `json:"nested,omitempty"`
	Limit  *int     `json:"limit,omitempty"`
	Offset *int     `json:"skip,omitempty"`
	Page   *int     `json:"page,omitempty"`
}

// ...request

// response...

type GetListData[T any] struct {
	Payload    []T             `json:"payload,omitempty"`
	Pagination *PaginationMeta `json:"pagination,omitempty"`
}

type PaginationMeta struct {
	Skip        *int `json:"skip,omitempty"`
	Limit       *int `json:"limit,omitempty"`
	Count       *int `json:"count,omitempty"`
	PageCount   *int `json:"pageCount,omitempty"`
	CurrentPage *int `json:"currentPage,omitempty"`
}

// ...response

//

type FieldExprApplier func(builder QueryBuilder, field string, expr FieldExpr, dbUtils DbUtils) QueryBuilder

type DbUtils interface {
	GetFieldExpr(builder QueryBuilder, expr FieldExpr, sqlField string, isJSONB bool) QueryBuilder
}

type BaseOptions struct {
	DbUtils DbUtils
}

type Base[T any] interface {
	GetHandler()
	ReadFromHttpRequestParams(r *http.Request, additionalWhere Filter) (GetListData[T], error)
	ReadFromQueryPayload(payload QueryPayload) (GetListData[T], error)
}
