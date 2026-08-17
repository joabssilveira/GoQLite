// TODO-rename to goqlite
package goqlitegorm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"strings"

	"github.com/joabssilveira/GoQLite/goqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type GormQueryBuilder struct {
	Db     *gorm.DB
	Schema *schema.Schema
}

func newGormQueryBuilder(db *gorm.DB) *GormQueryBuilder {
	stmt := &gorm.Statement{DB: db}
	_ = stmt.Parse(db.Statement.Model)

	return &GormQueryBuilder{
		Db:     db,
		Schema: stmt.Schema,
	}
}

func applyQuery(builder *GormQueryBuilder, payload goqlite.QueryPayload, dbUtils goqlite.DbUtils) *GormQueryBuilder {
	// WHERE
	applyJoinsFromFilter(builder.Db, builder.Db.Statement.Model, payload.Where)
	builder = goqlite.ApplyFilter(builder, payload.Where, applyFieldExpr, dbUtils).(*GormQueryBuilder)

	// SELECT
	if len(payload.Select) > 0 {
		qualified := make([]string, 0, len(payload.Select))

		for _, fieldName := range payload.Select {
			if strings.Contains(fieldName, ".") {
				// já vem qualificado (ex: relation.field)
				qualified = append(qualified, fieldName)
				continue
			}

			if builder.Schema != nil {
				qualified = append(
					qualified,
					quoteIdent(builder.Schema.Table)+"."+quoteIdent(fieldName),
				)
			} else {
				qualified = append(qualified, quoteIdent(fieldName))
			}
		}

		builder.Db = builder.Db.Select(qualified)
	}

	// ORDER
	for _, o := range payload.Order {
		dir := strings.ToUpper(o.Dir)
		if dir != "ASC" && dir != "DESC" {
			dir = "ASC"
		}
		builder.Db = builder.Db.Order(o.Field + " " + dir)
	}

	// LIMIT
	if payload.Limit != nil {
		builder.Db = builder.Db.Limit(*payload.Limit)
	}

	// OFFSET
	if payload.Offset != nil {
		builder.Db = builder.Db.Offset(*payload.Offset)
	}

	// NESTED (join automático)
	if payload.Nested != "" {
		tree := goqlite.ParseNestedTree(payload.Nested)
		for _, node := range tree {
			applyNestedNode(builder.Db, builder.Db.Statement.Model, node, "", dbUtils)
		}
	}

	return builder
}

func toGormRelationPath(path string) string {
	parts := strings.Split(path, ".")
	for i, p := range parts {
		parts[i] = goqlite.SnakeToCamel(p)
	}
	return strings.Join(parts, ".")
}

func applyNestedNode(db *gorm.DB, parentModel any, node *goqlite.NestedNode, prefix string, dbUtils goqlite.DbUtils) {
	// resolve nome real da relação no struct
	gormName := toGormRelationPath(node.Name)

	// monta path completo pro preload
	var gormPath string
	if prefix == "" {
		gormPath = gormName
	} else {
		gormPath = prefix + "." + gormName
	}

	db = db.Preload(gormPath, func(tx *gorm.DB) *gorm.DB {
		if node.Query != nil {

			// 🔥 auto-inject PK + FK
			pk, fk := resolveRelationKeysFromModel(parentModel, gormName, db)

			if len(node.Query.Select) > 0 {
				if pk != "" && !contains(node.Query.Select, pk) {
					node.Query.Select = append(node.Query.Select, pk)
				}
				if fk != "" && !contains(node.Query.Select, fk) {
					node.Query.Select = append(node.Query.Select, fk)
				}
			}

			sub := newGormQueryBuilder(tx)
			sub = applyQuery(sub, *node.Query, dbUtils)
			return sub.Db
		}
		return tx
	})

	// resolve model filho corretamente
	childModel := getChildModel(parentModel, gormName, db)

	// recursão
	for _, child := range node.Childs {
		applyNestedNode(db, childModel, child, gormPath, dbUtils)
	}
}

func resolveRelationKeysFromModel(model any, path string, Db *gorm.DB) (pk string, fk string) {
	parts := strings.Split(path, ".")
	relationName := parts[len(parts)-1]

	stmt := &gorm.Statement{DB: Db}
	_ = stmt.Parse(model)

	if stmt.Schema == nil {
		return
	}

	rel, ok := stmt.Schema.Relationships.Relations[relationName]
	if !ok {
		return
	}

	// PK do filho
	if rel.FieldSchema != nil && len(rel.FieldSchema.PrimaryFields) > 0 {
		pk = rel.FieldSchema.PrimaryFields[0].DBName
	}

	// FK de ligação
	if len(rel.References) > 0 {
		fk = rel.References[0].ForeignKey.DBName
	}

	return
}

func getChildModel(parentModel any, relationName string, db *gorm.DB) any {
	stmt := &gorm.Statement{DB: db}
	_ = stmt.Parse(parentModel)

	if stmt.Schema == nil {
		return nil
	}

	rel, ok := stmt.Schema.Relationships.Relations[relationName]
	if !ok {
		return nil
	}

	return reflect.New(rel.FieldSchema.ModelType).Interface()
}

func contains(arr []string, s string) bool {
	for _, v := range arr {
		if v == s {
			return true
		}
	}
	return false
}

func applyRelationJoinWithParentAlias(
	db *gorm.DB,
	parentModel any,
	rel *schema.Relationship,
	alias string,
	parentAlias string,
) {

	var parentKey, childKey string
	for _, ref := range rel.References {
		parentKey = ref.PrimaryKey.DBName
		childKey = ref.ForeignKey.DBName
		break
	}

	parentTable := parentAlias
	if parentTable == "" {
		stmt := &gorm.Statement{DB: db}
		_ = stmt.Parse(parentModel)
		parentTable = stmt.Schema.Table

		if strings.Contains(parentTable, ".") {
			parts := strings.Split(parentTable, ".")
			parentTableSchema := parts[0]
			parentTableNameOnly := parts[1]
			parentTable = quoteIdent(parentTableSchema) + "." + quoteIdent(parentTableNameOnly)
		} else {
			parentTable = quoteIdent(parentTable)
		}
	} else {
		parentTable = quoteIdent(parentTable)
	}

	relationTable := rel.FieldSchema.Table
	if strings.Contains(relationTable, ".") {
		parts := strings.Split(relationTable, ".")
		relationTableSchema := parts[0]
		relationTableNameOnly := parts[1]
		relationTable = quoteIdent(relationTableSchema) + "." + quoteIdent(relationTableNameOnly)
	} else {
		relationTable = quoteIdent(relationTable)
	}

	var join string
	if rel.Type == schema.BelongsTo {
		join = fmt.Sprintf(
			// `LEFT JOIN %s "%s" ON "%s"."%s" = "%s"."%s"`,
			`LEFT JOIN %s "%s" ON "%s"."%s" = %s."%s"`,
			relationTable,
			alias,
			alias,
			parentKey,
			parentTable,
			childKey,
		)
	} else {
		join = fmt.Sprintf(
			// `LEFT JOIN %s "%s" ON "%s"."%s" = "%s"."%s"`,
			`LEFT JOIN %s "%s" ON "%s"."%s" = %s."%s"`,
			relationTable,
			alias,
			alias,
			childKey,
			parentTable,
			parentKey,
		)
	}

	if !hasJoin(db, join) {
		db.Joins(join)
	}
}

func applyFieldExpr(builder goqlite.QueryBuilder, field string, expr goqlite.FieldExpr, dbUtils goqlite.DbUtils) goqlite.QueryBuilder {
	gormBuilder, ok := builder.(*GormQueryBuilder)
	if !ok {
		return builder
	}

	db := gormBuilder.Db
	sqlField := ""
	isJSONB := false

	if strings.Contains(field, ".") {
		parts := strings.Split(field, ".")
		first := parts[0]
		rest := parts[1:]

		stmt := &gorm.Statement{DB: db}
		_ = stmt.Parse(db.Statement.Model)

		if stmt.Schema == nil {
			return builder
		}

		// tenta resolver o PRIMEIRO nível como relação
		relName := goqlite.SnakeToCamel(first)
		_, isRelation := stmt.Schema.Relationships.Relations[relName]

		customjsonop := expr.Op != nil && expr.Op.Op == "@>"

		if customjsonop {
			sqlField = quoteIdent(first) // só a coluna raiz (subjects)
			isJSONB = false              // não precisa cast
		} else if !isRelation && !customjsonop {
			// ❌ NÃO é relação → JSONB (igual antes)
			isJSONB = true
			column := first
			path := rest

			sqlField = fmt.Sprintf(
				"%s #>> '{%s}'",
				quoteIdent(column),
				strings.Join(path, ","),
			)
		} else {
			// ✅ É relação → agora seguimos cadeia ilimitada
			currentModel := db.Statement.Model
			currentAlias := ""
			var currentRel *schema.Relationship

			remainingParts := parts
			for i := 0; i < len(parts)-1; i++ {
				relationSnake := parts[i]
				relationName := goqlite.SnakeToCamel(relationSnake)

				stmt := &gorm.Statement{DB: db}
				_ = stmt.Parse(currentModel)

				currentRel = stmt.Schema.Relationships.Relations[relationName]
				if currentRel == nil {
					isJSONB = true
					column := remainingParts[0]
					path := remainingParts[1:]

					sqlField = fmt.Sprintf(
						"%s.%s #>> '{%s}'",
						quoteIdent(currentAlias),
						quoteIdent(column),
						strings.Join(path, ","),
					)

					break
				}

				applyRelationJoinWithParentAlias(db, currentModel, currentRel, relationSnake, currentAlias)

				currentModel = reflect.New(currentRel.FieldSchema.ModelType).Interface()
				currentAlias = relationSnake

				remainingParts = remainingParts[1:]
			}

			if !isJSONB {
				// último item é o campo real
				lastField := parts[len(parts)-1]

				stmt = &gorm.Statement{DB: db}
				_ = stmt.Parse(currentModel)

				var dbFieldName string
				for _, f := range stmt.Schema.Fields {
					if f.Name == goqlite.SnakeToCamel(lastField) || f.DBName == lastField {
						dbFieldName = f.DBName
						break
					}
				}

				if dbFieldName == "" {
					dbFieldName = lastField // fallback seguro
				}

				sqlField = quoteIdent(currentAlias) + "." + quoteIdent(dbFieldName)
			}
		}
	} else {
		if gormBuilder.Schema != nil {
			tableName := gormBuilder.Schema.Table
			if strings.Contains(tableName, ".") {
				parts := strings.Split(tableName, ".")
				tableNameSchema := parts[0]
				tableNameOnly := parts[1]

				tableName = quoteIdent(tableNameSchema) + "." + quoteIdent(tableNameOnly)
				sqlField = tableName + "." + quoteIdent(field)
			} else {
				sqlField = quoteIdent(gormBuilder.Schema.Table) + "." + quoteIdent(field)
			}

		} else {
			sqlField = quoteIdent(field)
		}
	}

	builder = dbUtils.GetFieldExpr(builder, expr, sqlField, isJSONB)

	return builder
}

func quoteIdent(s string) string {
	return `"` + s + `"`
}

func applyJoinsFromFilter(
	db *gorm.DB,
	model any,
	filter goqlite.Filter,
) {
	for field := range filter.Fields {
		if strings.Contains(field, ".") {
			ensureJoin(db, model, field)
		}
	}

	for _, f := range filter.And {
		applyJoinsFromFilter(db, model, f)
	}

	for _, f := range filter.Or {
		applyJoinsFromFilter(db, model, f)
	}

	if filter.Not != nil {
		applyJoinsFromFilter(db, model, *filter.Not)
	}
}

func ensureJoin(db *gorm.DB, model any, fieldPath string) {
	parts := strings.Split(fieldPath, ".")
	if len(parts) < 2 {
		return
	}

	currentModel := model
	parentAlias := "" // tabela raiz

	for i := 0; i < len(parts)-1; i++ {
		relationSnake := parts[i] // course, course_group
		relationName := goqlite.SnakeToCamel(relationSnake)

		stmt := &gorm.Statement{DB: db}
		_ = stmt.Parse(currentModel)

		rel, ok := stmt.Schema.Relationships.Relations[relationName]
		if !ok {
			return
		}

		alias := relationSnake

		applyRelationJoinWithParentAlias(db, currentModel, rel, alias, parentAlias)

		currentModel = reflect.New(rel.FieldSchema.ModelType).Interface()
		parentAlias = alias
	}
}

func hasJoin(db *gorm.DB, alias string) bool {
	for _, j := range db.Statement.Joins {
		if j.Name == alias {
			return true
		}
	}
	return false
}

func (g *GormQueryBuilder) Where(cond string, args ...interface{}) goqlite.QueryBuilder {
	g.Db = g.Db.Where(cond, args...)
	return g
}

func (g *GormQueryBuilder) And(sub goqlite.QueryBuilder) goqlite.QueryBuilder {
	model := g.Db.Statement.Model
	g.Db = g.Db.Where(sub.Build())
	g.Db.Statement.Model = model
	return g
}

func (g *GormQueryBuilder) Or(sub goqlite.QueryBuilder) goqlite.QueryBuilder {
	model := g.Db.Statement.Model
	g.Db = g.Db.Or(sub.Build())
	g.Db.Statement.Model = model
	return g
}

func (g *GormQueryBuilder) Not(sub goqlite.QueryBuilder) goqlite.QueryBuilder {
	model := g.Db.Statement.Model
	g.Db = g.Db.Not(sub.Build())
	g.Db.Statement.Model = model
	return g
}

func (g *GormQueryBuilder) Build() interface{} {
	return g.Db
}

func (g *GormQueryBuilder) Clone() goqlite.QueryBuilder {
	newDB := g.Db.Session(&gorm.Session{NewDB: true})
	return &GormQueryBuilder{
		Db:     newDB,
		Schema: g.Schema, // 🔥 mantém schema
	}
}

func ReadFromQueryPayload[T any](db *gorm.DB, payload goqlite.QueryPayload, dbUtils goqlite.DbUtils) (goqlite.GetListData[T], error) {
	goqlite.ApplyPagination(&payload)

	// =========================
	// 1) COUNT
	// =========================

	// var total int64
	var total int64

	countPayload := goqlite.ExtractCountPayload(payload)

	countBuilder := newGormQueryBuilder(db.Model(new(T)))
	countBuilder = applyQuery(countBuilder, countPayload, dbUtils)

	if err := countBuilder.Db.Count(&total).Error; err != nil {
		return goqlite.GetListData[T]{}, err
	}

	// =========================
	// 2) DATA
	// =========================

	var list []T

	dataBuilder := newGormQueryBuilder(db.Model(new(T)))
	dataBuilder = applyQuery(dataBuilder, payload, dbUtils)

	if err := dataBuilder.Db.Find(&list).Error; err != nil {
		return goqlite.GetListData[T]{}, err
	}

	// =========================
	// RESPONSE
	// =========================

	return goqlite.GetListData[T]{
		Payload:    list,
		Pagination: goqlite.BuildPaginationMeta(payload, total),
	}, nil
}

func ReadFromHttpRequestParams[T any](db *gorm.DB, r *http.Request, additionalWhere goqlite.Filter, dbUtils goqlite.DbUtils) (goqlite.GetListData[T], error) {
	var payload goqlite.QueryPayload

	// where
	if raw := r.URL.Query().Get("where"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload.Where); err != nil {
			return goqlite.GetListData[T]{}, fmt.Errorf("invalid where json: %w", err)
		}
	}

	payload.Where = goqlite.MergeWhereWithAnd(payload.Where, additionalWhere)

	// select
	if raw := r.URL.Query().Get("select"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload.Select); err != nil {
			return goqlite.GetListData[T]{}, fmt.Errorf("invalid select json: %w", err)
		}
	}

	// sort
	if raw := r.URL.Query().Get("sort"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload.Order); err != nil {
			return goqlite.GetListData[T]{}, fmt.Errorf("invalid sort json: %w", err)
		}
	}

	// limit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		var v int
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return goqlite.GetListData[T]{}, fmt.Errorf("invalid limit: %w", err)
		}
		payload.Limit = &v
	}

	// skip
	if raw := r.URL.Query().Get("skip"); raw != "" {
		var v int
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return goqlite.GetListData[T]{}, fmt.Errorf("invalid skip: %w", err)
		}
		payload.Offset = &v
	}

	// page
	if raw := r.URL.Query().Get("page"); raw != "" {
		var v int
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return goqlite.GetListData[T]{}, fmt.Errorf("invalid page: %w", err)
		}
		payload.Page = &v
	}

	// nested
	payload.Nested = r.URL.Query().Get("nested")

	// page -> skip
	// It already exists in ReadFromQueryPayload.
	// goqlite.ApplyPagination(&payload)

	return ReadFromQueryPayload[T](db, payload, dbUtils)
}

//

type BaseOptions struct {
	goqlite.BaseOptions
	Db *gorm.DB
}

type Base[T any] struct {
	Options BaseOptions
}

func (base Base[T]) GetHandler() {

}

func (base Base[T]) ReadFromHttpRequestParams(r *http.Request, additionalWhere goqlite.Filter) (goqlite.GetListData[T], error) {
	return ReadFromHttpRequestParams[T](base.Options.Db, r, additionalWhere, base.Options.DbUtils)
}

func (base Base[T]) ReadFromQueryPayload(payload goqlite.QueryPayload) (goqlite.GetListData[T], error) {
	return ReadFromQueryPayload[T](base.Options.Db, payload, base.Options.DbUtils)
}
