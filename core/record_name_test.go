package core

import (
	"net/url"
	"strings"
	"testing"
)

func TestRecordDisplayName_ConventionalURL(t *testing.T) {
	record := Record{
		"id":   float64(6),
		"name": "view-6",
		"url":  "https://l101:443/api/v5/views/6/",
	}
	if got := recordDisplayName(record); got != "View" {
		t.Fatalf("recordDisplayName = %q, want View", got)
	}
}

func TestRecordDisplayName_LatestVersion(t *testing.T) {
	record := Record{"url": "https://l101/api/latest/users/1/"}
	if got := recordDisplayName(record); got != "User" {
		t.Fatalf("recordDisplayName = %q, want User", got)
	}
}

func TestRecordDisplayName_VTask(t *testing.T) {
	record := Record{"url": "https://l101/api/v5/vtasks/42/"}
	if got := recordDisplayName(record); got != "VTask" {
		t.Fatalf("recordDisplayName = %q, want VTask", got)
	}
}

func TestRecordDisplayName_NonConventionalURL(t *testing.T) {
	cases := []struct {
		name  string
		url   string
		want  string
	}{
		{"extra method", "https://l101/api/v5/clusters/1/rpc/", ""},
		{"missing url", "", ""},
		{"list endpoint", "https://l101/api/v5/views/", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			record := Record{}
			if tc.url != "" {
				record["url"] = tc.url
			}
			if got := recordDisplayName(record); got != tc.want {
				t.Fatalf("recordDisplayName = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestRecord_PrettyTableUsesURLName(t *testing.T) {
	record := Record{
		"id":   float64(1),
		"name": "root",
		"path": "/",
		"url":  "https://l101:443/api/v5/views/1/",
	}
	table := record.PrettyTable()
	if !strings.Contains(table, "View:") {
		t.Fatalf("expected View header, got:\n%s", table)
	}
	if strings.Contains(table, ResourceTypeKey) {
		t.Fatal("PrettyTable should not depend on @resourceType")
	}
}

func TestRecordDisplayName_PluralForms(t *testing.T) {
	cases := []struct {
		segment string
		want    string
	}{
		{"policies", "Policy"},
		{"queries", "Query"},
		{"entries", "Entry"},
		{"indices", "Index"},
		{"categories", "Category"},
		{"views", "View"},
		{"vtasks", "VTask"},
	}
	for _, tc := range cases {
		t.Run(tc.segment, func(t *testing.T) {
			record := Record{"url": "https://l101/api/v5/" + tc.segment + "/1/"}
			if got := recordDisplayName(record); got != tc.want {
				t.Fatalf("recordDisplayName = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestCapitalizeIdentifier_Empty(t *testing.T) {
	if got := capitalizeIdentifier(""); got != "" {
		t.Fatalf("capitalizeIdentifier(\"\") = %q", got)
	}
}

func TestIsVTaskRecord(t *testing.T) {
	if !isVTaskRecord(Record{"url": "https://l101/api/v5/vtasks/1/"}) {
		t.Fatal("expected vtask record")
	}
	if isVTaskRecord(Record{"url": "https://l101/api/v5/views/1/"}) {
		t.Fatal("expected non-vtask record")
	}
}

func TestConventionalResourceSegment_InvalidURL(t *testing.T) {
	if _, ok := conventionalResourceSegmentFromRecord(Record{"url": 123}); ok {
		t.Fatal("expected non-string url to fail")
	}
	if _, ok := conventionalResourceSegmentFromRecord(Record{"url": "://bad"}); ok {
		t.Fatal("expected invalid url to fail")
	}
}

func TestConventionalResourceSegment_PathVariants(t *testing.T) {
	cases := []struct {
		name string
		path string
		want string
		ok   bool
	}{
		{"valid", "/api/v5/views/6/", "views", true},
		{"no api segment", "/v5/views/6/", "", false},
		{"wrong depth list", "/api/v5/views/", "", false},
		{"wrong depth rpc", "/api/v5/clusters/1/rpc/", "", false},
		{"bad version", "/api/bad/views/6/", "", false},
		{"empty resource", "/api/v5//6/", "", false},
		{"empty id", "/api/v5/views//", "", false},
		{"singular segment", "/api/latest/policy/1/", "policy", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			parsed, err := url.Parse("https://host" + tc.path)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			got, ok := conventionalResourceSegment(parsed)
			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Fatalf("segment = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestSingularizeResourceSegment_NoSuffix(t *testing.T) {
	if got := singularizeResourceSegment("host"); got != "host" {
		t.Fatalf("singularizeResourceSegment(host) = %q", got)
	}
}
