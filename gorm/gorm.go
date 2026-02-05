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
	model := g.Db.Statement.Model
	g.Db = g.Db.Where(sub.Build())
	g.Db.Statement.Model = model
	return g
}

func (g *GormQueryBuilder) Or(sub fwork_server_orm.QueryBuilder) fwork_server_orm.QueryBuilder {
	model := g.Db.Statement.Model
	g.Db = g.Db.Or(sub.Build())
	g.Db.Statement.Model = model
	return g
}

func (g *GormQueryBuilder) Not(sub fwork_server_orm.QueryBuilder) fwork_server_orm.QueryBuilder {
	model := g.Db.Statement.Model
	g.Db = g.Db.Not(sub.Build())
	g.Db.Statement.Model = model
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
			`LEFT JOIN %s "%s" ON "%s"."%s" = "%s"."%s"`,
			relationTable,
			alias,
			alias,
			parentKey,
			parentTable,
			childKey,
		)
	} else {
		join = fmt.Sprintf(
			`LEFT JOIN %s "%s" ON "%s"."%s" = "%s"."%s"`,
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
		first := parts[0]
		rest := parts[1:]

		stmt := &gorm.Statement{DB: db}
		_ = stmt.Parse(db.Statement.Model)

		if stmt.Schema == nil {
			return builder
		}

		// tenta resolver o PRIMEIRO nível como relação
		relName := fwork_server_orm.SnakeToCamel(first)
		_, isRelation := stmt.Schema.Relationships.Relations[relName]

		// ❌ NÃO é relação → JSONB (igual antes)
		if !isRelation {
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

			for i := 0; i < len(parts)-1; i++ {
				relationSnake := parts[i]
				relationName := fwork_server_orm.SnakeToCamel(relationSnake)

				stmt := &gorm.Statement{DB: db}
				_ = stmt.Parse(currentModel)

				currentRel = stmt.Schema.Relationships.Relations[relationName]
				if currentRel == nil {
					return builder
				}

				applyRelationJoinWithParentAlias(db, currentModel, currentRel, relationSnake, currentAlias)

				currentModel = reflect.New(currentRel.FieldSchema.ModelType).Interface()
				currentAlias = relationSnake
			}

			// último item é o campo real
			lastField := parts[len(parts)-1]

			stmt = &gorm.Statement{DB: db}
			_ = stmt.Parse(currentModel)

			var dbFieldName string
			for _, f := range stmt.Schema.Fields {
				if f.Name == fwork_server_orm.SnakeToCamel(lastField) || f.DBName == lastField {
					dbFieldName = f.DBName
					break
				}
			}

			if dbFieldName == "" {
				dbFieldName = lastField // fallback seguro
			}

			sqlField = quoteIdent(currentAlias) + "." + quoteIdent(dbFieldName)
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

	// TODO remove comments
	// if err := countBuilder.Db.Count(&total).Error; err != nil {
	// 	return fwork_server_orm.GetListData[T]{}, err
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
	// It already exists in GormGetList.
	// fwork_server_orm.ApplyPagination(&payload)

	return GormGetList[T](db, payload)
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
	parts := strings.Split(fieldPath, ".")
	if len(parts) < 2 {
		return
	}

	currentModel := model
	parentAlias := "" // tabela raiz

	for i := 0; i < len(parts)-1; i++ {
		relationSnake := parts[i] // course, course_group
		relationName := fwork_server_orm.SnakeToCamel(relationSnake)

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
