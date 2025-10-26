package common

import (
	"fmt"
	"reflect"
)

type Set[T comparable] struct {
	m     map[T]struct{}
	order []T // Tracks insertion order
}

// NewSet creates a new set from a slice. If the slice is nil, the set is empty.
func NewSet[T comparable](items []T) *Set[T] {
	s := &Set[T]{
		m:     make(map[T]struct{}),
		order: make([]T, 0, len(items)),
	}
	for _, item := range items {
		if _, exists := s.m[item]; !exists {
			s.m[item] = struct{}{}
			s.order = append(s.order, item)
		}
	}
	return s
}

// Add inserts the element into the set.
// Returns true if the element was added (i.e., it wasn't already present).
func (s *Set[T]) Add(item T) bool {
	if _, exists := s.m[item]; exists {
		return false
	}
	s.m[item] = struct{}{}
	s.order = append(s.order, item)
	return true
}

// Remove deletes the element from the set.
// Returns true if the element existed and was removed.
func (s *Set[T]) Remove(item T) bool {
	if _, exists := s.m[item]; exists {
		delete(s.m, item)
		// Remove from order slice
		for i, v := range s.order {
			if v == item {
				s.order = append(s.order[:i], s.order[i+1:]...)
				break
			}
		}
		return true
	}
	return false
}

// Contains checks if the item is present in the set.
func (s *Set[T]) Contains(item T) bool {
	_, exists := s.m[item]
	return exists
}

// ToSlice returns all elements in the set as a slice (unordered).
func (s *Set[T]) ToSlice() []T {
	result := make([]T, 0, len(s.m))
	for k := range s.m {
		result = append(result, k)
	}
	return result
}

// ToOrderedSlice returns all elements in the set as a slice, preserving insertion order.
func (s *Set[T]) ToOrderedSlice() []T {
	// Return a copy to prevent external modifications
	result := make([]T, len(s.order))
	copy(result, s.order)
	return result
}

// Len returns the number of elements in the set.
func (s *Set[T]) Len() int {
	return len(s.m)
}

// Clear removes all elements from the set.
func (s *Set[T]) Clear() {
	s.m = make(map[T]struct{})
	s.order = make([]T, 0)
}

// NewSetFromAny creates a set from any input value, handling []interface{} and []T.
// Supports conversion to T: int64, float64, string.
func NewSetFromAny[T comparable](input any) (*Set[T], error) {
	switch casted := input.(type) {
	case nil:
		return NewSet[T](nil), nil
	case []T:
		return NewSet[T](casted), nil
	case []interface{}:
		converted := make([]T, 0, len(casted))
		for _, item := range casted {
			val, err := convertTo[T](item)
			if err != nil {
				return nil, err
			}
			converted = append(converted, val)
		}
		return NewSet[T](converted), nil
	default:
		return nil, fmt.Errorf("unsupported input type for set: %T", input)
	}
}

// convertTo converts an interface{} to T, supporting int64, float64, string.
func convertTo[T comparable](v any) (T, error) {
	var zero T
	switch any(zero).(type) {
	case int64:
		switch val := v.(type) {
		case int:
			return any(int64(val)).(T), nil
		case int64:
			return any(val).(T), nil
		case float64:
			return any(int64(val)).(T), nil
		default:
			return zero, fmt.Errorf("cannot convert %T to int64", v)
		}
	case float64:
		switch val := v.(type) {
		case float64:
			return any(val).(T), nil
		case int:
			return any(float64(val)).(T), nil
		case int64:
			return any(float64(val)).(T), nil
		default:
			return zero, fmt.Errorf("cannot convert %T to float64", v)
		}
	case string:
		if str, ok := v.(string); ok {
			return any(str).(T), nil
		}
		return zero, fmt.Errorf("cannot convert %T to string", v)
	default:
		return zero, fmt.Errorf("unsupported target type: %v", reflect.TypeOf(zero))
	}
}
