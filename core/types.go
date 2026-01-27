package fwork_server_orm

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

// filter...

type Filter struct {
	And    []Filter             `json:"$and,omitempty"`
	Or     []Filter             `json:"$or,omitempty"`
	Not    *Filter              `json:"$not,omitempty"`
	Fields map[string]FieldExpr `json:"-"`
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

type FieldExprApplier func(builder QueryBuilder, field string, expr FieldExpr) QueryBuilder
