package core

import (
	"io"
	"strings"
	"testing"
)

func TestParams_ToBodyAndQuery(t *testing.T) {
	params := Params{"name": "alice", "count": 2}
	reader, err := params.ToBody()
	if err != nil {
		t.Fatalf("ToBody: %v", err)
	}
	body, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if !strings.Contains(string(body), `"name":"alice"`) {
		t.Fatalf("unexpected body: %s", body)
	}
	if params.ToQuery() == "" {
		t.Fatal("expected non-empty query")
	}
}

func TestParams_ToMultipartFormData(t *testing.T) {
	params := Params{
		"name":   "alice",
		"tags":   []string{"a", "b"},
		"values": []any{1, 2},
		"file": FileData{
			Filename:    "test.txt",
			Content:     []byte("hello"),
			ContentType: "text/plain",
		},
		"raw": []byte("bytes"),
	}
	form, err := params.ToMultipartFormData()
	if err != nil {
		t.Fatalf("ToMultipartFormData: %v", err)
	}
	if form == nil || form.ContentType == "" {
		t.Fatal("expected multipart form data")
	}
	if form.Body == nil {
		t.Fatal("expected body")
	}
}

func TestParams_ToMultipartFormData_FileWithoutContentType(t *testing.T) {
	params := Params{
		"upload": FileData{
			Filename: "data.bin",
			Content:  []byte("payload"),
		},
	}
	form, err := params.ToMultipartFormData()
	if err != nil {
		t.Fatalf("ToMultipartFormData: %v", err)
	}
	if form == nil || form.Body == nil {
		t.Fatal("expected multipart body")
	}
}

func TestParams_UpdateAndWithout(t *testing.T) {
	params := Params{"a": 1, "b": 2}
	other := Params{"b": 99, "c": 3}

	params.Update(other, true)
	if params["b"] != 2 {
		t.Fatalf("override=true should skip existing keys, got %v", params["b"])
	}

	params.Update(other, false)
	if params["b"] != 99 {
		t.Fatalf("override=false should update existing keys, got %v", params["b"])
	}

	params.UpdateWithout(Params{"c": 99, "d": 4}, false, []string{"c"})
	if params["c"] != 3 {
		t.Fatalf("expected c to be skipped, got %v", params["c"])
	}

	params.Without("a")
	if _, ok := params["a"]; ok {
		t.Fatal("expected a to be removed")
	}
}

func TestParams_FromStruct(t *testing.T) {
	type Body struct {
		Name string `json:"name"`
		Age  int    `json:"age,omitempty"`
	}
	params := make(Params)
	if err := params.FromStruct(Body{Name: "alice", Age: 0}); err != nil {
		t.Fatalf("FromStruct: %v", err)
	}
	if params["name"] != "alice" {
		t.Fatalf("unexpected params: %v", params)
	}
	if _, ok := params["age"]; ok {
		t.Fatal("expected zero age to be omitted")
	}
}

func TestRecord_DisplayAndAccessors(t *testing.T) {
	record := Record{
		"id":          float64(1),
		"name":        "alice",
		"guid":        "g-1",
		"tenant_id":   9,
		"tenant_name": "tenant-a",
		"extra":       "value",
	}

	if record.Empty() {
		t.Fatal("record should not be empty")
	}
	if record.RecordID() != 1 {
		t.Fatalf("RecordID = %d", record.RecordID())
	}
	if record.RecordGUID() != "g-1" {
		t.Fatalf("RecordGUID = %q", record.RecordGUID())
	}
	if record.RecordTenantID() != 9 {
		t.Fatalf("RecordTenantID = %d", record.RecordTenantID())
	}
	if record.RecordName() != "alice" {
		t.Fatalf("RecordName = %q", record.RecordName())
	}
	if record.RecordTenantName() != "tenant-a" {
		t.Fatalf("RecordTenantName = %q", record.RecordTenantName())
	}

	record.SetMissingValue("name", "bob")
	if record["name"] != "alice" {
		t.Fatal("SetMissingValue should not overwrite existing key")
	}
	record.SetMissingValue("new_key", "new")
	if record["new_key"] != "new" {
		t.Fatal("SetMissingValue should set missing key")
	}

	if table := record.PrettyTable(); table == "" {
		t.Fatal("expected PrettyTable output")
	}
	if json := record.PrettyJson(); json == "" {
		t.Fatal("expected PrettyJson output")
	}
	if s := record.String(); s == "" {
		t.Fatal("expected String output")
	}

	var filled struct {
		Name string `json:"name"`
		ID   int64  `json:"id"`
	}
	if err := record.Fill(&filled); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	if filled.Name != "alice" {
		t.Fatalf("filled name = %q", filled.Name)
	}
}

func TestRecord_FillErrors(t *testing.T) {
	record := Record{"name": "alice"}
	if err := record.Fill("not-a-pointer"); err == nil {
		t.Fatal("expected error for non-pointer container")
	}
	if err := record.Fill((*int)(nil)); err == nil {
		t.Fatal("expected error for nil pointer container")
	}
}

func TestRecordSet_Display(t *testing.T) {
	set := RecordSet{
		{"id": 1, "name": "alice"},
		{"id": 2, "name": "bob"},
	}
	if set.Empty() {
		t.Fatal("record set should not be empty")
	}
	if table := set.PrettyTable(); table == "" {
		t.Fatal("expected PrettyTable output")
	}
	if json := set.PrettyJson("  "); json == "" {
		t.Fatal("expected PrettyJson output")
	}

	var filled []struct {
		Name string `json:"name"`
	}
	if err := set.Fill(&filled); err != nil {
		t.Fatalf("Fill: %v", err)
	}
	if len(filled) != 2 {
		t.Fatalf("expected 2 records, got %d", len(filled))
	}
}

func TestRecord_EmptyRecord(t *testing.T) {
	var record Record
	if !record.Empty() {
		t.Fatal("nil record should be empty")
	}
	if table := record.PrettyTable(); table != "<>" {
		t.Fatalf("expected <>, got %q", table)
	}
}

func TestRecordSet_Empty(t *testing.T) {
	var set RecordSet
	if !set.Empty() {
		t.Fatal("empty set should be empty")
	}
	if table := set.PrettyTable(); table == "" {
		t.Fatal("expected PrettyTable output for empty set")
	}
}

func TestRecordSet_FillErrors(t *testing.T) {
	set := RecordSet{{"name": "alice"}}
	if err := set.Fill([]struct{ Name string }{}); err == nil {
		t.Fatal("expected error for non-pointer container")
	}
	if err := set.Fill((*[]int)(nil)); err == nil {
		t.Fatal("expected error for non-struct slice")
	}

	var ptrUsers []*struct{ Name string }
	if err := set.Fill(&ptrUsers); err != nil {
		t.Fatalf("Fill pointers: %v", err)
	}
	if len(ptrUsers) != 1 || ptrUsers[0].Name != "alice" {
		t.Fatalf("unexpected ptr fill: %+v", ptrUsers)
	}
}

func TestModelToRecord(t *testing.T) {
	type Model struct {
		Name string `json:"name"`
	}
	record := ModelToRecord(Model{Name: "alice"})
	if record["name"] != "alice" {
		t.Fatalf("unexpected record: %v", record)
	}
	if _, ok := record[ResourceTypeKey]; ok {
		t.Fatal("ModelToRecord should not inject @resourceType")
	}
}

func TestRecord_AccessorPanics(t *testing.T) {
	empty := Record{}
	panics := []struct {
		name string
		fn   func()
	}{
		{"RecordID", func() { _ = empty.RecordID() }},
		{"RecordGUID", func() { _ = empty.RecordGUID() }},
		{"RecordName", func() { _ = empty.RecordName() }},
		{"RecordTenantID", func() { _ = empty.RecordTenantID() }},
		{"RecordTenantName", func() { _ = empty.RecordTenantName() }},
	}
	for _, tc := range panics {
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if recover() == nil {
					t.Fatalf("%s should panic when field missing", tc.name)
				}
			}()
			tc.fn()
		})
	}

	badID := Record{"id": "not-a-number"}
	defer func() {
		if recover() == nil {
			t.Fatal("RecordID should panic on invalid id type")
		}
	}()
	_ = badID.RecordID()
}

func TestRecord_TenantIDInvalidTypePanics(t *testing.T) {
	record := Record{"tenant_id": "bad"}
	defer func() {
		if recover() == nil {
			t.Fatal("RecordTenantID should panic on invalid tenant_id type")
		}
	}()
	_ = record.RecordTenantID()
}
