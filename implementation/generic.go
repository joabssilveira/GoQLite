package goqliteimplementation

import "github.com/joabssilveira/GoQLite/goqlite"

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

type DbUtilsGeneric struct {
}

func (u DbUtilsGeneric) GetFieldExpr(builder goqlite.QueryBuilder, expr goqlite.FieldExpr, sqlField string, isJSONB bool) goqlite.QueryBuilder {
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
