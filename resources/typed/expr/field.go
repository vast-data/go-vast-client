package expr

// field is the private backing type for StrField and IntField.
type field[T any] struct {
	value *T
	expr  queryExpr
}

func exactField[T any](v T) field[T]        { return field[T]{value: &v} }
func exprField[T any](e queryExpr) field[T] { return field[T]{expr: e} }

func (f field[T]) isSet() bool { return f.value != nil || f.expr != nil }

func (f field[T]) serializeToParam(key string) map[string]any {
	if f.expr != nil {
		k, v := f.expr.queryParam(key)
		return map[string]any{k: v}
	}
	return map[string]any{key: *f.value}
}

// StrField is the struct field type for string search parameters.
// Assign values via expr.Str.X(...) factory methods.
//
//	type UserSearchParams struct {
//	    Name expr.StrField `json:"name,omitempty"`
//	}
type StrField = field[string]

// IntField is the struct field type for integer search parameters.
// Assign values via expr.Int.X(...) factory methods.
//
//	type UserSearchParams struct {
//	    Uid expr.IntField `json:"uid,omitempty"`
//	}
type IntField = field[int64]
