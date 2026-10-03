package widgets

import "testing"

func TestFlattenResourceList(t *testing.T) {
	entries := []ResourceListEntry{
		ResourceEntry("profiles"),
		ResourceGroup("dataengine", []string{"functions", "pipelines"}),
		ResourceEntry("users"),
	}

	got := flattenResourceList(entries)
	want := [][]string{
		{"profiles"},
		{"[dataengine]"},
		{"  functions"},
		{"  pipelines"},
		{"users"},
	}

	if len(got) != len(want) {
		t.Fatalf("len=%d want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i][0] != want[i][0] {
			t.Fatalf("row %d: got %q want %q", i, got[i][0], want[i][0])
		}
	}
}

func TestIsSectionLabel(t *testing.T) {
	if !isSectionLabel("[dataengine]") {
		t.Fatal("expected [dataengine] to be a section label")
	}
	if isSectionLabel("functions") {
		t.Fatal("resource must not be a section label")
	}
	if isSectionLabel("  functions") {
		t.Fatal("indented resource must not be a section label")
	}
}

func TestSectionLabel(t *testing.T) {
	if got := sectionLabel("dataengine"); got != "[dataengine]" {
		t.Fatalf("got %q", got)
	}
	if got := sectionLabel("[dataengine]"); got != "[dataengine]" {
		t.Fatalf("got %q", got)
	}
}
