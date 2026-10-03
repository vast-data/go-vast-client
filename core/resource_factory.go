package core

import (
	"fmt"
	"reflect"
)

// NewTypedResource constructs a typed resource of type T that embeds
// *TypedVastResource, wiring it to the given untyped VastRest (VMS, DataEngine, etc.).
// T's type name must exist in untypedRest.GetResourceMap().
func NewTypedResource[T any](untypedRest VastRest) *T {
	var zero T
	t := reflect.TypeOf(zero)
	resourceType := t.Name()

	instance := reflect.New(t).Interface()
	typedRes := NewTypedVastResource(resourceType, untypedRest)

	val := reflect.ValueOf(instance).Elem()
	found := false
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Type() == reflect.TypeOf((*TypedVastResource)(nil)) {
			if field.CanSet() {
				field.Set(reflect.ValueOf(typedRes))
				found = true
				break
			}
		}
	}
	if !found {
		panic(fmt.Sprintf("Resource %s does not embed *TypedVastResource or field is not settable", resourceType))
	}

	if _, ok := untypedRest.GetResourceMap()[resourceType]; !ok {
		panic(fmt.Sprintf("untyped resource type %s not found in REST", resourceType))
	}

	if result, ok := instance.(*T); ok {
		return result
	}
	panic(fmt.Sprintf("Failed to convert instance to type *%s", resourceType))
}

// NewUntypedResource constructs an untyped resource of type T that embeds
// *VastResource, registers it on rest.GetResourceMap(), and returns *T.
func NewUntypedResource[T any](rest VastRest, resourcePath string, resourceOps ...ResourceOps) *T {
	var zero T
	t := reflect.TypeOf(zero)
	resourceType := t.Name()

	instance := reflect.New(t).Interface()
	resource := NewVastResource(resourcePath, resourceType, rest, NewResourceOps(resourceOps...), instance)

	val := reflect.ValueOf(instance).Elem()
	found := false
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Type() == reflect.TypeOf((*VastResource)(nil)) {
			if field.CanSet() {
				field.Set(reflect.ValueOf(resource))
				found = true
				break
			}
		}
	}
	if !found {
		panic(fmt.Sprintf("Resource %s does not embed *VastResource or field is not settable", resourceType))
	}

	if res, ok := instance.(VastResourceAPIWithContext); ok {
		rest.GetResourceMap()[resourceType] = ResourceEntry{
			VastResourceAPIWithContext: res,
			IdentityField:              resource.identityField,
		}
	}

	if result, ok := instance.(*T); ok {
		return result
	}
	panic(fmt.Sprintf("Failed to convert instance to type *%s", resourceType))
}
