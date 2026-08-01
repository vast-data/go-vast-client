package core

import "testing"

func TestStructToMap_NestedAndNonStructInputs(t *testing.T) {
	type Tags struct {
		Values []string `json:"values,omitempty"`
		Empty  []int    `json:"empty,omitempty"`
		Nil    []string `json:"nil,omitempty"`
	}
	type Body struct {
		Name string `json:"name"`
		Tags Tags   `json:"tags"`
	}

	m := structToMap(Body{Name: "x", Tags: Tags{Values: []string{"a"}}})
	if m["name"] != "x" {
		t.Fatalf("unexpected map: %v", m)
	}
	if _, ok := m["tags"].(map[string]interface{}); !ok {
		t.Fatalf("expected nested tags map, got %v", m["tags"])
	}

	if len(structToMap(42)) != 0 {
		t.Fatal("non-struct should return empty map")
	}

	var nilPtr *Body
	if len(structToMap(nilPtr)) != 0 {
		t.Fatal("nil pointer to struct should return empty map")
	}
}

func TestStructToMap_EmptySliceIncluded(t *testing.T) {
	type Body struct {
		Tags []string `json:"tags"`
	}
	m := structToMap(Body{Tags: []string{}})
	if _, ok := m["tags"]; !ok {
		t.Fatal("expected empty slice to be included without omitempty")
	}
}
