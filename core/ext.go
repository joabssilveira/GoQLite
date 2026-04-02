package fwork_server_orm

type Field[T any] string

type FilterBuilder[T any] struct {
	filter Filter
}

func NewFilter[T any]() *FilterBuilder[T] {
	return &FilterBuilder[T]{
		filter: Filter{
			Fields: make(map[string]FieldExpr),
		},
	}
}

func (b *FilterBuilder[T]) Build() Filter {
	return b.filter
}

func (b *FilterBuilder[T]) set(
	field Field[T],
	fn func(*FieldExpr),
) *FilterBuilder[T] {

	key := string(field)

	expr := b.filter.Fields[key]
	fn(&expr)

	b.filter.Fields[key] = expr
	return b
}

//

func (b *FilterBuilder[T]) Eq(field Field[T], v any) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		e.Eq = v
	})
}

func (b *FilterBuilder[T]) Ne(field Field[T], v any) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		e.Ne = v
	})
}

//

func (b *FilterBuilder[T]) Gt(field Field[T], v any) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		e.Gt = v
	})
}

func (b *FilterBuilder[T]) Gte(field Field[T], v any) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		e.Gte = v
	})
}

func (b *FilterBuilder[T]) Lt(field Field[T], v any) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		e.Lt = v
	})
}

func (b *FilterBuilder[T]) Lte(field Field[T], v any) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		e.Lte = v
	})
}

//

func (b *FilterBuilder[T]) In(field Field[T], v ...any) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		e.In = v
	})
}

func (b *FilterBuilder[T]) Nin(field Field[T], v ...any) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		e.Nin = v
	})
}

//

func (b *FilterBuilder[T]) Like(field Field[T], v string) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		e.Like = v
	})
}

func (b *FilterBuilder[T]) ILike(field Field[T], v string) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		e.ILike = v
	})
}

//

func (b *FilterBuilder[T]) Between(field Field[T], a, c any) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		e.Between = []any{a, c}
	})
}

//

func (b *FilterBuilder[T]) IsNull(field Field[T]) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		v := true
		e.IsNull = &v
	})
}

func (b *FilterBuilder[T]) NotNull(field Field[T]) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		v := false
		e.IsNull = &v
	})
}

func (b *FilterBuilder[T]) Exists(field Field[T]) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		v := true
		e.Exists = &v
	})
}

func (b *FilterBuilder[T]) NotExists(field Field[T]) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		v := false
		e.Exists = &v
	})
}

func (b *FilterBuilder[T]) Op(field Field[T], v FieldExprOp) *FilterBuilder[T] {
	return b.set(field, func(e *FieldExpr) {
		e.Op = &v
	})
}

//

func (b *FilterBuilder[T]) And(filters ...*FilterBuilder[T]) *FilterBuilder[T] {
	for _, f := range filters {
		b.filter.And = append(b.filter.And, f.filter)
	}
	return b
}

func (b *FilterBuilder[T]) Or(filters ...*FilterBuilder[T]) *FilterBuilder[T] {
	for _, f := range filters {
		b.filter.Or = append(b.filter.Or, f.filter)
	}
	return b
}

func (b *FilterBuilder[T]) Not(f *FilterBuilder[T]) *FilterBuilder[T] {
	b.filter.Not = &f.filter
	return b
}

//

// EXAMPLE

type MyModelExample struct {
	Uuid        string                 `json:"uuid" gorm:"column:uuid;type:uuid;primaryKey"`
	SomeProp    string                 `json:"some_prop" gorm:"column:name"`
	AnotherProp string                 `json:"another_prop" gorm:"column:realm_uuid;type:uuid;not null"`
	SubDoc      []MySubdocModelExample `json:"subdoc" gorm:"type:jsonb"`
}

type MySubdocModelExample struct {
	Uuid        string `json:"uuid,omitempty" gorm:"column:uuid;type:uuid;primaryKey"`
	SomeProp    string `json:"some_prop,omitempty" gorm:"column:name"`
	AnotherProp string `json:"another_prop,omitempty" gorm:"column:realm_uuid;type:uuid;not null"`
}

const (
	MyModelExampleUuid        Field[MyModelExample] = "uuid"
	MyModelExampleSomeProp    Field[MyModelExample] = "some_prop"
	MyModelExampleAnotherProp Field[MyModelExample] = "another_prop"
	MyModelExampleSubDoc      Field[MyModelExample] = "subdoc"
)

func example() Filter {
	filter := NewFilter[MyModelExample]().
		Eq(MyModelExampleUuid, "value").
		In(MyModelExampleAnotherProp, "valuea", "valueb").
		And(
			NewFilter[MyModelExample]().Like(MyModelExampleSomeProp, "%admin%"),
		).
		Build()

	return filter
}

func exampleCustomOp() Filter {
	filter := NewFilter[MyModelExample]().
		Op(MyModelExampleSubDoc, FieldExprOp{
			Op:    "@>",
			Value: "[{\"uuid\": \"XXX\"}]",
		}).
		Build()

	return filter
}
