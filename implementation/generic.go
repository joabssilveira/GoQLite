package fwork_server_orm_implementation

import fwork_server_orm "github.com/joabssilveira/GoQLite/core"

func CastIfJSONB(sqlField string, isJSONB bool, value interface{}) string {
	if !isJSONB {
		return sqlField
	}

	switch value.(type) {
	case int, int32, int64, float32, float64:
		return sqlField + "::numeric"
	case bool:
		return sqlField + "::boolean"
	default:
		return sqlField
	}
}

type FieldExprApplier func(builder fwork_server_orm.QueryBuilder, field string, expr fwork_server_orm.FieldExpr, dbUtils DbUtils) fwork_server_orm.QueryBuilder

func ApplyFilter(builder fwork_server_orm.QueryBuilder, filter fwork_server_orm.Filter, fieldExprApplier FieldExprApplier, dbUtils DbUtils) fwork_server_orm.QueryBuilder {

	// Campos
	for field, expr := range filter.Fields {
		// builder = applyFieldExpr(builder, field, expr)
		builder = fieldExprApplier(builder, field, expr, dbUtils)
	}

	// AND
	for _, andItemFilter := range filter.And {
		subBuilder := builder.Clone()
		ApplyFilter(subBuilder, andItemFilter, fieldExprApplier, dbUtils)
		builder = builder.And(subBuilder)
	}

	// OR
	for _, orItemFilter := range filter.Or {
		subBuilder := builder.Clone()
		ApplyFilter(subBuilder, orItemFilter, fieldExprApplier, dbUtils)
		builder = builder.Or(subBuilder)
	}

	// NOT
	if filter.Not != nil {
		subBuilder := builder.Clone()
		ApplyFilter(subBuilder, *filter.Not, fieldExprApplier, dbUtils)
		builder = builder.Not(subBuilder)
	}

	return builder
}

type DbUtils interface {
	GetFieldExpr(builder fwork_server_orm.QueryBuilder, expr fwork_server_orm.FieldExpr, sqlField string, isJSONB bool) fwork_server_orm.QueryBuilder
}

type DbUtilsGeneric struct {
}

func (u DbUtilsGeneric) GetFieldExpr(builder fwork_server_orm.QueryBuilder, expr fwork_server_orm.FieldExpr, sqlField string, isJSONB bool) fwork_server_orm.QueryBuilder {
	if expr.Eq != nil {
		builder = builder.Where(sqlField+" = ?", expr.Eq)
	}

	if expr.Ne != nil {
		builder = builder.Where(sqlField+" <> ?", expr.Ne)
	}

	if expr.Gt != nil {
		builder = builder.Where(CastIfJSONB(sqlField, isJSONB, expr.Gt)+" > ?", expr.Gt)
	}

	if expr.Gte != nil {
		builder = builder.Where(CastIfJSONB(sqlField, isJSONB, expr.Gte)+" >= ?", expr.Gte)
	}

	if expr.Lt != nil {
		builder = builder.Where(CastIfJSONB(sqlField, isJSONB, expr.Lt)+" < ?", expr.Lt)
	}

	if expr.Lte != nil {
		builder = builder.Where(CastIfJSONB(sqlField, isJSONB, expr.Lte)+" <= ?", expr.Lte)
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
			CastIfJSONB(sqlField, isJSONB, expr.Between[0])+" BETWEEN ? AND ?",
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

	if expr.Op != nil {
		builder = builder.Where(sqlField+" "+expr.Op.Op+" ?", expr.Op.Value)
	}

	return builder
}
