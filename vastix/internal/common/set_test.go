package common

import (
	"reflect"
	"testing"
)

func TestSet_ToOrderedSlice(t *testing.T) {
	// Test that ToOrderedSlice preserves insertion order
	items := []string{"profiles", "ssh_connections", "user_keys [local store]", "api_tokens [local store]"}
	set := NewSet(items)

	ordered := set.ToOrderedSlice()

	if !reflect.DeepEqual(ordered, items) {
		t.Errorf("ToOrderedSlice() = %v, want %v", ordered, items)
	}
}

func TestSet_ToOrderedSlice_WithAdd(t *testing.T) {
	// Test that ToOrderedSlice preserves order even with Add operations
	set := NewSet[string](nil)

	set.Add("first")
	set.Add("second")
	set.Add("third")
	set.Add("second") // Duplicate, should be ignored

	ordered := set.ToOrderedSlice()
	expected := []string{"first", "second", "third"}

	if !reflect.DeepEqual(ordered, expected) {
		t.Errorf("ToOrderedSlice() = %v, want %v", ordered, expected)
	}
}

func TestSet_ToOrderedSlice_WithRemove(t *testing.T) {
	// Test that ToOrderedSlice preserves order after removal
	items := []string{"first", "second", "third", "fourth"}
	set := NewSet(items)

	set.Remove("second")

	ordered := set.ToOrderedSlice()
	expected := []string{"first", "third", "fourth"}

	if !reflect.DeepEqual(ordered, expected) {
		t.Errorf("ToOrderedSlice() = %v, want %v", ordered, expected)
	}
}

func TestSet_ToSlice_Unordered(t *testing.T) {
	// Test that ToSlice still works (but may be unordered)
	items := []string{"a", "b", "c"}
	set := NewSet(items)

	unordered := set.ToSlice()

	// Check that all items are present (order doesn't matter)
	if len(unordered) != len(items) {
		t.Errorf("ToSlice() length = %d, want %d", len(unordered), len(items))
	}

	for _, item := range items {
		found := false
		for _, v := range unordered {
			if v == item {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("ToSlice() missing item %v", item)
		}
	}
}

func TestSet_Clear_ClearsOrder(t *testing.T) {
	// Test that Clear removes both map and order
	items := []string{"a", "b", "c"}
	set := NewSet(items)

	set.Clear()

	if set.Len() != 0 {
		t.Errorf("After Clear(), Len() = %d, want 0", set.Len())
	}

	ordered := set.ToOrderedSlice()
	if len(ordered) != 0 {
		t.Errorf("After Clear(), ToOrderedSlice() length = %d, want 0", len(ordered))
	}
}

