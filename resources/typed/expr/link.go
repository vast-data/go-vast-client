package expr

import _ "unsafe"

// coreExprSerializeField aliases core.exprSerializeField via go:linkname.
// Assigning serializeFieldParam here wires up the hook at program start without
// requiring core to import this package (which would create a cycle).
//
//go:linkname coreExprSerializeField github.com/vast-data/go-vast-client/core.exprSerializeField
var coreExprSerializeField func(v any, key string) (map[string]any, bool) = serializeFieldParam

// fieldSerializer is the private interface satisfied by the internal field[T] type
// (backing StrField and IntField). Never exported — method names stay unexported.
type fieldSerializer interface {
	isSet() bool
	serializeToParam(key string) map[string]any
}

// serializeFieldParam is the bridge called by core.structToMap via coreExprSerializeField.
// It returns (params, true) when v is a fieldSerializer with a value set,
// and (nil, false) for any other type so structToMap continues normally.
func serializeFieldParam(v any, key string) (map[string]any, bool) {
	f, ok := v.(fieldSerializer)
	if !ok || !f.isSet() {
		return nil, false
	}
	return f.serializeToParam(key), true
}
