package widgets

import "testing"

func TestFlattenResourceList(t *testing.T) {
	entries := []ResourceListEntry{
		ResourceEntry("profiles"),
		ResourceEntry("users"),
		ResourceEntry("vsettings"),
		ResourceGroup("local_database", []string{"api_tokens [local database]", "user_keys [local database]"}),
		ResourceGroup("dataengine", []string{"functions", "pipelines"}),
		ResourceGroup("debug", []string{"ssh_connections"}),
	}

	got := flattenResourceList(entries)
	want := [][]string{
		{"profiles"},
		{"users"},
		{"vsettings"},
		{"[local_database]"},
		{"  api_tokens [local database]"},
		{"  user_keys [local database]"},
		{"[dataengine]"},
		{"  functions"},
		{"  pipelines"},
		{"[debug]"},
		{"  ssh_connections"},
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
