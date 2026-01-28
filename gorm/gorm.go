package fwork_server_gorm

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"strings"

	fwork_server_orm "github.com/joabssilveira/GoQLite/core"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

type GormQueryBuilder struct {
	Db     *gorm.DB
	Schema *schema.Schema
}

func NewGormQueryBuilder(db *gorm.DB) *GormQueryBuilder {
	stmt := &gorm.Statement{DB: db}
	_ = stmt.Parse(db.Statement.Model)

	return &GormQueryBuilder{
		Db:     db,
		Schema: stmt.Schema,
	}
}

func (g *GormQueryBuilder) Where(cond string, args ...interface{}) fwork_server_orm.QueryBuilder {
	g.Db = g.Db.Where(cond, args...)
	return g
}

func (g *GormQueryBuilder) And(sub fwork_server_orm.QueryBuilder) fwork_server_orm.QueryBuilder {
	g.Db = g.Db.Where(sub.Build())
	return g
}

func (g *GormQueryBuilder) Or(sub fwork_server_orm.QueryBuilder) fwork_server_orm.QueryBuilder {
	g.Db = g.Db.Or(sub.Build())
	return g
}

func (g *GormQueryBuilder) Not(sub fwork_server_orm.QueryBuilder) fwork_server_orm.QueryBuilder {
	g.Db = g.Db.Not(sub.Build())
	return g
}

func (g *GormQueryBuilder) Build() interface{} {
	return g.Db
}

func (g *GormQueryBuilder) Clone() fwork_server_orm.QueryBuilder {
	newDB := g.Db.Session(&gorm.Session{NewDB: true})
	return &GormQueryBuilder{
		Db:     newDB,
		Schema: g.Schema, // 🔥 mantém schema
	}
}

func ApplyQuery(builder *GormQueryBuilder, payload fwork_server_orm.QueryPayload) *GormQueryBuilder {
	// WHERE
	ApplyJoinsFromFilter(builder.Db, builder.Db.Statement.Model, payload.Where)
	// builder = fwork_server_orm.ApplyFilter(builder, payload.Where).(*GormQueryBuilder)
	builder = fwork_server_orm.ApplyFilter(builder, payload.Where, applyFieldExpr).(*GormQueryBuilder)

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
		tree := fwork_server_orm.ParseNestedTree(payload.Nested)
		for _, node := range tree {
			applyNestedNode(builder.Db, builder.Db.Statement.Model, node, "")
		}
	}

	return builder
}

func toGormRelationPath(path string) string {
	parts := strings.Split(path, ".")
	for i, p := range parts {
		parts[i] = fwork_server_orm.SnakeToCamel(p)
	}
	return strings.Join(parts, ".")
}

func applyNestedNode(db *gorm.DB, parentModel any, node *fwork_server_orm.NestedNode, prefix string) {
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

			sub := NewGormQueryBuilder(tx)
			sub = ApplyQuery(sub, *node.Query)
			return sub.Db
		}
		return tx
	})

	// resolve model filho corretamente
	childModel := getChildModel(parentModel, gormName, db)

	// recursão
	for _, child := range node.Childs {
		applyNestedNode(db, childModel, child, gormPath)
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

func resolveRelationJoin(db *gorm.DB, model any, path string) (joinTable string, joinAlias string, column string, ok bool) {
	parts := strings.Split(path, ".")
	if len(parts) < 2 {
		return
	}

	relationName := fwork_server_orm.SnakeToCamel(parts[0])
	column = parts[1]

	stmt := &gorm.Statement{DB: db}
	_ = stmt.Parse(model)

	if stmt.Schema == nil {
		return
	}

	rel, exists := stmt.Schema.Relationships.Relations[relationName]
	if !exists {
		return
	}

	joinTable = rel.FieldSchema.Table
	joinAlias = parts[0] // usamos o nome snake como alias

	ok = true
	return
}

func getRelationJoinRawName(db *gorm.DB, parentModel any, relationName string, alias string) string {
	stmt := &gorm.Statement{DB: db}
	_ = stmt.Parse(parentModel)

	rel := stmt.Schema.Relationships.Relations[relationName]

	parentTable := stmt.Schema.Table
	relationTable := rel.FieldSchema.Table

	var parentKey string
	var childKey string

	for _, ref := range rel.References {
		parentKey = ref.PrimaryKey.DBName
		childKey = ref.ForeignKey.DBName
		break
	}

	if rel.Type == schema.BelongsTo {
		return fmt.Sprintf(
			`LEFT JOIN "%s" "%s" ON "%s"."%s" = "%s"."%s"`,
			relationTable,
			alias,
			alias,
			parentKey,
			parentTable,
			childKey,
		)
	} else {
		return fmt.Sprintf(
			`LEFT JOIN "%s" "%s" ON "%s"."%s" = "%s"."%s"`,
			relationTable,
			alias,
			alias,
			childKey,
			parentTable,
			parentKey,
		)
	}

}

func applyRelationJoin(db *gorm.DB, parentModel any, relationName string, alias string) {
	raw := getRelationJoinRawName(db, parentModel, relationName, alias)

	if hasJoin(db, raw) {
		return
	}

	db.Joins(
		raw,
	)
}

func applyFieldExpr(builder fwork_server_orm.QueryBuilder, field string, expr fwork_server_orm.FieldExpr) fwork_server_orm.QueryBuilder {
	gormBuilder, ok := builder.(*GormQueryBuilder)
	if !ok {
		return builder
	}

	db := gormBuilder.Db
	sqlField := ""
	isJSONB := false

	if strings.Contains(field, ".") {
		parts := strings.Split(field, ".")
		first := parts[0] // ex: children
		rest := parts[1:] // ex: someField

		stmt := &gorm.Statement{DB: db}
		_ = stmt.Parse(db.Statement.Model)

		if stmt.Schema != nil {
			// tenta resolver relação
			relName := fwork_server_orm.SnakeToCamel(first)
			rel := stmt.Schema.Relationships.Relations[relName]

			if rel != nil {
				// garante join
				applyRelationJoin(db, db.Statement.Model, rel.Name, first)

				// resolve campo real no schema do filho
				if len(rest) == 0 {
					return builder
				}

				fieldName := rest[0]
				var dbFieldName string

				for _, f := range rel.FieldSchema.Fields {
					if f.Name == fwork_server_orm.SnakeToCamel(fieldName) || f.DBName == fieldName {
						dbFieldName = f.DBName
						break
					}
				}

				if dbFieldName == "" {
					// fallback (não deveria acontecer, mas evita panic)
					dbFieldName = fieldName
				}

				sqlField = quoteIdent(first) + "." + quoteIdent(dbFieldName)
			} else {
				// JSONB fallback
				isJSONB = true
				column := first
				path := rest

				sqlField = fmt.Sprintf(
					"%s #>> '{%s}'",
					quoteIdent(column),
					strings.Join(path, ","),
				)
			}
		}
	} else {
		// stmt := &gorm.Statement{DB: db}
		// _ = stmt.Parse(db.Statement.Model)
		// if stmt.Schema != nil {
		// 	sqlField = quoteIdent(stmt.Schema.Table) + "." + quoteIdent(field)
		// } else if stmt.Table != "" {
		// 	sqlField = quoteIdent(stmt.Table) + "." + quoteIdent(field)
		// } else {
		// 	sqlField = quoteIdent(field)
		// }

		if gormBuilder.Schema != nil {
			sqlField = quoteIdent(gormBuilder.Schema.Table) + "." + quoteIdent(field)
		} else {
			sqlField = quoteIdent(field)
		}
	}

	// =========================
	// Operadores
	// =========================

	if expr.Eq != nil {
		builder = builder.Where(sqlField+" = ?", expr.Eq)
	}

	if expr.Ne != nil {
		builder = builder.Where(sqlField+" <> ?", expr.Ne)
	}

	if expr.Gt != nil {
		builder = builder.Where(fwork_server_orm.CastIfJSONB(sqlField, isJSONB, expr.Gt)+" > ?", expr.Gt)
	}

	if expr.Gte != nil {
		builder = builder.Where(fwork_server_orm.CastIfJSONB(sqlField, isJSONB, expr.Gte)+" >= ?", expr.Gte)
	}

	if expr.Lt != nil {
		builder = builder.Where(fwork_server_orm.CastIfJSONB(sqlField, isJSONB, expr.Lt)+" < ?", expr.Lt)
	}

	if expr.Lte != nil {
		builder = builder.Where(fwork_server_orm.CastIfJSONB(sqlField, isJSONB, expr.Lte)+" <= ?", expr.Lte)
	}

	if len(expr.In) > 0 {
		builder = builder.Where(sqlField+" IN ?", expr.In)
	}

	if len(expr.Nin) > 0 {
		builder = builder.Where(sqlField+" NOT IN ?", expr.Nin)
	}

	if expr.Like != "" {
		builder = builder.Where(sqlField+" LIKE ?", "%"+expr.Like+"%")
	}

	if expr.ILike != "" {
		builder = builder.Where(sqlField+" ILIKE ?", "%"+expr.ILike+"%")
	}

	if len(expr.Between) == 2 {
		builder = builder.Where(
			fwork_server_orm.CastIfJSONB(sqlField, isJSONB, expr.Between[0])+" BETWEEN ? AND ?",
			expr.Between[0],
			expr.Between[1],
		)
	}

	if expr.Exists != nil {
		if *expr.Exists {
			builder = builder.Where(sqlField + " IS NOT NULL")
		} else {
			builder = builder.Where(sqlField + " IS NULL")
		}
	}

	if expr.IsNull != nil {
		if *expr.IsNull {
			builder = builder.Where(sqlField + " IS NULL")
		} else {
			builder = builder.Where(sqlField + " IS NOT NULL")
		}
	}

	return builder
}

func quoteIdent(s string) string {
	return `"` + s + `"`
}

func GormGetList[T any](db *gorm.DB, payload fwork_server_orm.QueryPayload) (fwork_server_orm.GetListData[T], error) {
	fwork_server_orm.ApplyPagination(&payload)

	// =========================
	// 1) COUNT
	// =========================

	var total int64

	countPayload := fwork_server_orm.ExtractCountPayload(payload)

	countBuilder := NewGormQueryBuilder(db.Model(new(T)))
	countBuilder = ApplyQuery(countBuilder, countPayload)

	// TODO descomentar na versao final
	// if err := countBuilder.Db.Count(&total).Error; err != nil {
	// 	return GetListData[T]{}, err
	// }

	// =========================
	// 2) DATA
	// =========================

	var list []T

	dataBuilder := NewGormQueryBuilder(db.Model(new(T)))
	dataBuilder = ApplyQuery(dataBuilder, payload)

	if err := dataBuilder.Db.Find(&list).Error; err != nil {
		return fwork_server_orm.GetListData[T]{}, err
	}

	// =========================
	// RESPONSE
	// =========================

	return fwork_server_orm.GetListData[T]{
		Payload:    list,
		Pagination: fwork_server_orm.BuildPaginationMeta(payload, total),
	}, nil
}

func GormGetListHttp[T any](db *gorm.DB, r *http.Request, additionalWhere fwork_server_orm.Filter) (fwork_server_orm.GetListData[T], error) {
	var payload fwork_server_orm.QueryPayload

	// where
	if raw := r.URL.Query().Get("where"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload.Where); err != nil {
			return fwork_server_orm.GetListData[T]{}, fmt.Errorf("invalid where json: %w", err)
		}
	}

	payload.Where = fwork_server_orm.MergeWhereWithAnd(payload.Where, additionalWhere)

	// select
	if raw := r.URL.Query().Get("select"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload.Select); err != nil {
			return fwork_server_orm.GetListData[T]{}, fmt.Errorf("invalid select json: %w", err)
		}
	}

	// sort
	if raw := r.URL.Query().Get("sort"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &payload.Order); err != nil {
			return fwork_server_orm.GetListData[T]{}, fmt.Errorf("invalid sort json: %w", err)
		}
	}

	// limit
	if raw := r.URL.Query().Get("limit"); raw != "" {
		var v int
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return fwork_server_orm.GetListData[T]{}, fmt.Errorf("invalid limit: %w", err)
		}
		payload.Limit = &v
	}

	// skip
	if raw := r.URL.Query().Get("skip"); raw != "" {
		var v int
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return fwork_server_orm.GetListData[T]{}, fmt.Errorf("invalid skip: %w", err)
		}
		payload.Offset = &v
	}

	// page
	if raw := r.URL.Query().Get("page"); raw != "" {
		var v int
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return fwork_server_orm.GetListData[T]{}, fmt.Errorf("invalid page: %w", err)
		}
		payload.Page = &v
	}

	// nested
	payload.Nested = r.URL.Query().Get("nested")

	// page -> skip
	fwork_server_orm.ApplyPagination(&payload)

	// =========================
	// 1) COUNT
	// =========================

	return GormGetList[T](db, payload)

	// var total int64

	// countPayload := fwork_server_orm.ExtractCountPayload(payload)

	// countBuilder := NewGormQueryBuilder(db.Model(new(T)))
	// countBuilder = ApplyQuery(countBuilder, countPayload)

	// // TODO descomentar na versao final
	// // if err := countBuilder.Db.Count(&total).Error; err != nil {
	// // 	return GetListData[T]{}, err
	// // }

	// // =========================
	// // 2) DATA
	// // =========================

	// var list []T

	// dataBuilder := NewGormQueryBuilder(db.Model(new(T)))
	// dataBuilder = ApplyQuery(dataBuilder, payload)

	// if err := dataBuilder.Db.Find(&list).Error; err != nil {
	// 	return fwork_server_orm.GetListData[T]{}, err
	// }

	// // =========================
	// // RESPONSE
	// // =========================

	// return fwork_server_orm.GetListData[T]{
	// 	Payload:    list,
	// 	Pagination: fwork_server_orm.BuildPaginationMeta(payload, total),
	// }, nil
}

func ApplyJoinsFromFilter(
	db *gorm.DB,
	model any,
	filter fwork_server_orm.Filter,
) {
	for field := range filter.Fields {
		if strings.Contains(field, ".") {
			ensureJoin(db, model, field)
		}
	}

	for _, f := range filter.And {
		ApplyJoinsFromFilter(db, model, f)
	}

	for _, f := range filter.Or {
		ApplyJoinsFromFilter(db, model, f)
	}

	if filter.Not != nil {
		ApplyJoinsFromFilter(db, model, *filter.Not)
	}
}

func ensureJoin(db *gorm.DB, model any, fieldPath string) {
	if !strings.Contains(fieldPath, ".") {
		return
	}

	parts := strings.Split(fieldPath, ".")
	relationAlias := parts[0]                               // courses_def
	relationName := fwork_server_orm.SnakeToCamel(parts[0]) // CoursesDef

	stmt := &gorm.Statement{DB: db}
	if err := stmt.Parse(model); err != nil || stmt.Schema == nil {
		return
	}

	rel, ok := stmt.Schema.Relationships.Relations[relationName]
	if !ok {
		return
	}

	applyRelationJoin(db, model, rel.Name, relationAlias)
}

func hasJoin(db *gorm.DB, alias string) bool {
	for _, j := range db.Statement.Joins {
		if j.Name == alias {
			return true
		}
	}
	return false
}
