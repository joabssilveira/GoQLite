package fwork_server_orm

import (
	"encoding/json"
	"math"

	"strings"
)

func isEmptyFilter(f Filter) bool {
	return len(f.And) == 0 &&
		len(f.Or) == 0 &&
		f.Not == nil &&
		len(f.Fields) == 0
}

func (f FieldExpr) isEmpty() bool {
	return f.Eq == nil &&
		f.Ne == nil &&
		f.Gt == nil &&
		f.Gte == nil &&
		f.Lt == nil &&
		f.Lte == nil &&
		len(f.In) == 0 &&
		len(f.Nin) == 0 &&
		f.Like == "" &&
		f.ILike == "" &&
		len(f.Between) == 0 &&
		f.Exists == nil &&
		f.IsNull == nil
}

func parseNestedRecursive(s string, prefix string, result *[]string) {
	i := 0
	for i < len(s) {
		if s[i] == ',' {
			i++
			continue
		}

		start := i
		for i < len(s) && s[i] != '{' && s[i] != ',' {
			i++
		}
		key := strings.TrimSpace(s[start:i])
		currentPath := key
		if prefix != "" {
			currentPath = prefix + "." + key
		}

		*result = append(*result, currentPath)

		if i < len(s) && s[i] == '{' {
			i++
			level := 1
			subStart := i
			for i < len(s) && level > 0 {
				if s[i] == '{' {
					level++
				} else if s[i] == '}' {
					level--
				}
				i++
			}
			sub := s[subStart : i-1]
			parseNestedRecursive(sub, currentPath, result)
		}
	}
}

func parseNestedTreeRecursive(s string, result *[]*NestedNode) {
	i := 0

	for i < len(s) {
		if s[i] == ',' {
			i++
			continue
		}

		start := i
		for i < len(s) && s[i] != '{' && s[i] != ',' {
			i++
		}

		name := strings.TrimSpace(s[start:i])
		node := &NestedNode{Name: name}

		if i < len(s) && s[i] == '{' {
			i++
			level := 1
			subStart := i

			for i < len(s) && level > 0 {
				if s[i] == '{' {
					level++
				} else if s[i] == '}' {
					level--
				}
				i++
			}

			block := strings.TrimSpace(s[subStart : i-1])

			// separa query e filhos
			query, children := splitNestedBlock(block)
			if query != "" {
				var qp QueryPayload
				_ = json.Unmarshal([]byte(query), &qp)
				node.Query = &qp
			}

			if children != "" {
				parseNestedTreeRecursive(children, &node.Childs)
			}
		}

		*result = append(*result, node)
	}
}

func splitNestedBlock(s string) (query string, children string) {
	s = strings.TrimSpace(s)

	// se começa com { -> é JSON
	if strings.HasPrefix(s, "{") {
		level := 0
		for i := 0; i < len(s); i++ {
			if s[i] == '{' {
				level++
			} else if s[i] == '}' {
				level--
				if level == 0 {
					query = s[:i+1]
					if i+1 < len(s) {
						children = strings.TrimPrefix(s[i+1:], ",")
					}
					return
				}
			}
		}
	}

	// senão, tudo é children
	children = s
	return
}

//

func ApplyPagination(payload *QueryPayload) {
	// Se skip já veio, respeita e não recalcula
	if payload.Offset != nil {
		return
	}

	// Só calcula se page E limit existirem
	if payload.Page != nil && payload.Limit != nil {
		page := *payload.Page
		limit := *payload.Limit

		if page < 1 {
			page = 1
		}

		skip := (page - 1) * limit
		payload.Offset = &skip
	}
}

func ExtractCountPayload(payload QueryPayload) QueryPayload {
	return QueryPayload{
		Where: payload.Where,
		// tudo o resto vazio de propósito
	}
}

func BuildPaginationMeta(payload QueryPayload, total int64) *PaginationMeta {
	if payload.Limit == nil && payload.Offset == nil && payload.Page == nil {
		return nil
	}

	meta := &PaginationMeta{}

	if payload.Offset != nil {
		meta.Skip = payload.Offset
	}

	if payload.Limit != nil {
		meta.Limit = payload.Limit
	}

	count := int(total)
	meta.Count = &count

	if payload.Limit != nil {
		limit := *payload.Limit

		pageCount := int(math.Ceil(float64(count) / float64(limit)))
		meta.PageCount = &pageCount

		var currentPage int
		if payload.Page != nil {
			currentPage = *payload.Page
		} else if payload.Offset != nil {
			currentPage = (*payload.Offset / limit) + 1
		} else {
			currentPage = 1
		}

		meta.CurrentPage = &currentPage
	}

	return meta
}

func (f *Filter) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	f.Fields = make(map[string]FieldExpr)

	for k, v := range raw {
		switch k {

		case "$and":
			if err := json.Unmarshal(v, &f.And); err != nil {
				return err
			}

		case "$or":
			if err := json.Unmarshal(v, &f.Or); err != nil {
				return err
			}

		case "$not":
			var nf Filter
			if err := json.Unmarshal(v, &nf); err != nil {
				return err
			}
			f.Not = &nf

		default:
			// Aqui é o pulo do gato 👇
			// Pode ser:
			// 1) { "name": "Joao" }
			// 2) { "name": { "$eq": "Joao" } }
			// 3) { "age": { "$gt": 18 } }

			var expr FieldExpr

			// tenta como objeto (operadores)
			if err := json.Unmarshal(v, &expr); err == nil {
				// se pelo menos um operador foi preenchido, usamos
				if !expr.isEmpty() {
					f.Fields[k] = expr
					continue
				}
			}

			// senão, é valor direto -> $eq implícito
			var direct interface{}
			if err := json.Unmarshal(v, &direct); err != nil {
				return err
			}

			f.Fields[k] = FieldExpr{
				Eq: direct,
			}
		}
	}

	return nil
}

func SnakeToCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func ParseNested(input string) []string {
	input = strings.TrimSpace(input)
	input = strings.TrimPrefix(input, "{")
	input = strings.TrimSuffix(input, "}")

	var result []string
	parseNestedRecursive(input, "", &result)
	return result
}

func ApplyFilter(builder QueryBuilder, filter Filter, fieldExprApplier FieldExprApplier) QueryBuilder {

	// Campos
	for field, expr := range filter.Fields {
		// builder = applyFieldExpr(builder, field, expr)
		builder = fieldExprApplier(builder, field, expr)
	}

	// AND
	for _, andItemFilter := range filter.And {
		subBuilder := builder.Clone()
		ApplyFilter(subBuilder, andItemFilter, fieldExprApplier)
		builder = builder.And(subBuilder)
	}

	// OR
	for _, orItemFilter := range filter.Or {
		subBuilder := builder.Clone()
		ApplyFilter(subBuilder, orItemFilter, fieldExprApplier)
		builder = builder.Or(subBuilder)
	}

	// NOT
	if filter.Not != nil {
		subBuilder := builder.Clone()
		ApplyFilter(subBuilder, *filter.Not, fieldExprApplier)
		builder = builder.Not(subBuilder)
	}

	return builder
}

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

func ParseNestedTree(input string) []*NestedNode {
	input = strings.TrimSpace(input)

	if strings.HasPrefix(input, "{") && strings.HasSuffix(input, "}") {
		input = strings.TrimPrefix(input, "{")
		input = strings.TrimSuffix(input, "}")
	}

	var nodes []*NestedNode
	parseNestedTreeRecursive(input, &nodes)
	return nodes
}

func MergeWhereWithAnd(userWhere, additionalWhere Filter) Filter {
	if isEmptyFilter(userWhere) {
		return additionalWhere
	}

	if isEmptyFilter(additionalWhere) {
		return userWhere
	}

	return Filter{
		And: []Filter{
			additionalWhere,
			userWhere,
		},
	}
}
