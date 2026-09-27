package widgets

import "strings"

// ResourceListEntry is either a single selectable resource or a labeled group.
// Exactly one of Resource or Label should be set.
type ResourceListEntry struct {
	// Resource is a single selectable resource key (e.g. "users").
	Resource string
	// Label is a non-selectable section header (shown as <label>); Resources are its children.
	Label     string
	Resources []string
}

// ResourceEntry creates a top-level selectable resource entry.
func ResourceEntry(name string) ResourceListEntry {
	return ResourceListEntry{Resource: name}
}

// ResourceGroup creates a labeled group of resources under a non-selectable header.
func ResourceGroup(label string, resources []string) ResourceListEntry {
	return ResourceListEntry{Label: label, Resources: append([]string(nil), resources...)}
}

// IsGroup reports whether this entry is a labeled section.
func (e ResourceListEntry) IsGroup() bool {
	return strings.TrimSpace(e.Label) != ""
}

// sectionLabel formats a group label for display / row identity, e.g. [dataengine].
func sectionLabel(name string) string {
	name = strings.TrimSpace(name)
	name = strings.Trim(name, "[]")
	return "[" + name + "]"
}

// isSectionLabel reports whether a list cell is a non-selectable section header.
func isSectionLabel(cell string) bool {
	s := strings.TrimSpace(cell)
	return len(s) >= 2 && s[0] == '[' && s[len(s)-1] == ']'
}

// groupedResourcePrefix indents child resources under a section label in the list data.
// Select trims spaces, so the underlying widget key stays the bare resource name.
const groupedResourcePrefix = "  "

// flattenResourceList expands entries into list rows.
// Group labels become <label> rows; children are prefixed for indent.
func flattenResourceList(entries []ResourceListEntry) [][]string {
	var data [][]string
	for _, e := range entries {
		if e.IsGroup() {
			data = append(data, []string{sectionLabel(e.Label)})
			for _, name := range e.Resources {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				data = append(data, []string{groupedResourcePrefix + name})
			}
			continue
		}
		name := strings.TrimSpace(e.Resource)
		if name == "" {
			continue
		}
		data = append(data, []string{name})
	}
	return data
}
