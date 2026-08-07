package fwork_server_orm_implementation

import fwork_server_orm "github.com/joabssilveira/GoQLite/core"

func CastIfJSONBSqlserver(sqlField string, isJSONB bool, value interface{}) string {
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

type DbUtilsSqlserverSettings struct {
	Unaccent bool `json:"unaccent,omitempty"`
}

type DbUtilsSqlserver struct {
	Settings DbUtilsSqlserverSettings `json:"settings,omitempty"`
}

func (u DbUtilsSqlserver) GetFieldExpr(builder fwork_server_orm.QueryBuilder, expr fwork_server_orm.FieldExpr, sqlField string, isJSONB bool) fwork_server_orm.QueryBuilder {
	if expr.Eq != nil {
		if u.Settings.Unaccent {
			if s, ok := expr.Eq.(string); ok {
				builder = builder.Where("CAST("+sqlField+" AS VARCHAR(MAX)) COLLATE Latin1_General_CI_AI = ?", s)
			} else {
				builder = builder.Where(sqlField+" = ?", expr.Eq)
			}
		} else {
			builder = builder.Where(sqlField+" = ?", expr.Eq)
		}
	}

	if expr.Ne != nil {
		if u.Settings.Unaccent {
			if s, ok := expr.Ne.(string); ok {
				builder = builder.Where("CAST("+sqlField+" AS VARCHAR(MAX)) COLLATE Latin1_General_CI_AI <> ?", s)
			} else {
				builder = builder.Where(sqlField+" <> ?", expr.Ne)
			}
		} else {
			builder = builder.Where(sqlField+" <> ?", expr.Ne)
		}
	}

	if expr.Gt != nil {
		builder = builder.Where(CastIfJSONBSqlserver(sqlField, isJSONB, expr.Gt)+" > ?", expr.Gt)
	}

	if expr.Gte != nil {
		builder = builder.Where(CastIfJSONBSqlserver(sqlField, isJSONB, expr.Gte)+" >= ?", expr.Gte)
	}

	if expr.Lt != nil {
		builder = builder.Where(CastIfJSONBSqlserver(sqlField, isJSONB, expr.Lt)+" < ?", expr.Lt)
	}

	if expr.Lte != nil {
		builder = builder.Where(CastIfJSONBSqlserver(sqlField, isJSONB, expr.Lte)+" <= ?", expr.Lte)
	}

	if len(expr.In) > 0 {
		builder = builder.Where(sqlField+" IN ?", expr.In)
	}

	if len(expr.Nin) > 0 {
		builder = builder.Where(sqlField+" NOT IN ?", expr.Nin)
	}

	if expr.Like != "" {
		if u.Settings.Unaccent {
			builder = builder.Where("CAST("+sqlField+" AS VARCHAR(MAX)) COLLATE Latin1_General_CI_AI LIKE ?", "%"+expr.Like+"%")
		} else {
			builder = builder.Where(sqlField+" LIKE ?", "%"+expr.Like+"%")
		}
	}

	if expr.ILike != "" {
		if u.Settings.Unaccent {
			builder = builder.Where("CAST("+sqlField+" AS VARCHAR(MAX)) COLLATE Latin1_General_CI_AI ILIKE ?", "%"+expr.ILike+"%")
		} else {
			builder = builder.Where(sqlField+" ILIKE ?", "%"+expr.ILike+"%")
		}
	}

	if len(expr.Between) == 2 {
		builder = builder.Where(
			CastIfJSONBSqlserver(sqlField, isJSONB, expr.Between[0])+" BETWEEN ? AND ?",
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
