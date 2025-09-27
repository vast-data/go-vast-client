package vast_client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
)

// Test structures for Fill operations
type TestUser struct {
	ID     int64    `json:"id"`
	Name   string   `json:"name"`
	Email  string   `json:"email,omitempty"`
	Active bool     `json:"active"`
	Age    int      `json:"age,omitempty"`
	Tags   []string `json:"tags,omitempty"`
}

// Test structures for FromStruct operations
type SampleStruct struct {
	Name string `json:"name"`
	ID   string `json:"id"`
}

type EmbededStruct struct {
	FieldStruct `json:"field"`
	Hello       string `json:"hello"`
}

type FieldStruct struct {
	OnePoint string        `json:"one_point"`
	Sample   *SampleStruct `json:"sample"`
}

type TestRequest struct {
	Name     string  `json:"name"`
	Age      int     `json:"age"`
	Optional *bool   `json:"optional,omitempty"`
	Score    float64 `json:"score"`
}

// Test struct for omitempty edge cases
type TestUserWithOmit struct {
	Uid         string      `json:"uid,omitempty"`
	Name        string      `json:"name,omitempty"`
	Email       string      `json:"email,omitempty"`
	MobileNo    string      `json:"mobile_no,omitempty"`
	Age         int         `json:"age,omitempty"`
	Score       float64     `json:"score,omitempty"`
	Active      bool        `json:"active,omitempty"`
	Shortlisted []ShortList `json:"shortlisted,omitempty"`
}

type ShortList struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// Test struct for various JSON tag formats
type TagTestStruct struct {
	Field1 string `json:"field1"`                  // Simple tag
	Field2 string `json:"field2,omitempty"`        // With omitempty
	Field3 string `json:",omitempty"`              // Empty name with omitempty
	Field4 string `json:"-"`                       // Skip field
	Field5 string `json:"custom_name,omitempty"`   // Custom name with omitempty
	Field6 string `json:"field6,string"`           // With string option
	Field7 string `json:"field7,omitempty,string"` // Multiple options
	Field8 string // No tag
}

// Test struct for nested struct omitempty behavior
type NestedStruct struct {
	Value string `json:"value,omitempty"`
	Count int    `json:"count,omitempty"`
}

type ParentStruct struct {
	Name   string        `json:"name"`
	Nested NestedStruct  `json:"nested,omitempty"`
	Ptr    *NestedStruct `json:"ptr,omitempty"`
}

// Test struct for slice behavior
type SliceTestStruct struct {
	Items    []string `json:"items,omitempty"`
	Numbers  []int    `json:"numbers,omitempty"`
	Required []string `json:"required"` // No omitempty
}

func TestParams_ToQuery(t *testing.T) {
	tests := []struct {
		name   string
		params Params
		want   []string // Multiple valid orderings
	}{
		{
			name:   "empty params",
			params: Params{},
			want:   []string{""},
		},
		{
			name:   "single param",
			params: Params{"name": "test"},
			want:   []string{"name=test"},
		},
		{
			name:   "multiple params",
			params: Params{"name": "test", "id": 123},
			want:   []string{"id=123&name=test", "name=test&id=123"},
		},
		{
			name:   "special characters",
			params: Params{"query": "test value", "filter": "name=admin"},
			want:   []string{"filter=name%3Dadmin&query=test+value", "query=test+value&filter=name%3Dadmin"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.params.ToQuery()

			// Check if result matches any of the expected possibilities
			found := false
			for _, expected := range tt.want {
				if got == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Params.ToQuery() = %v, want one of %v", got, tt.want)
			}
		})
	}
}

func TestParams_ToBody(t *testing.T) {
	tests := []struct {
		name    string
		params  Params
		wantErr bool
	}{
		{
			name:    "empty params",
			params:  Params{},
			wantErr: false,
		},
		{
			name:    "simple params",
			params:  Params{"name": "test", "id": 123},
			wantErr: false,
		},
		{
			name:    "complex params",
			params:  Params{"user": map[string]interface{}{"name": "test", "age": 30}},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := tt.params.ToBody()
			if (err != nil) != tt.wantErr {
				t.Errorf("Params.ToBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && body == nil {
				t.Error("Params.ToBody() should return non-nil body")
			}
		})
	}
}

func TestParams_Update(t *testing.T) {
	tests := []struct {
		name     string
		original Params
		other    Params
		override bool
		want     Params
	}{
		{
			name:     "add new keys",
			original: Params{"a": 1},
			other:    Params{"b": 2, "c": 3},
			override: false,
			want:     Params{"a": 1, "b": 2, "c": 3},
		},
		{
			name:     "no override existing",
			original: Params{"a": 1, "b": 2},
			other:    Params{"b": 999, "c": 3},
			override: true, // when override is true, existing values are kept
			want:     Params{"a": 1, "b": 2, "c": 3},
		},
		{
			name:     "override existing",
			original: Params{"a": 1, "b": 2},
			other:    Params{"b": 999, "c": 3},
			override: false, // when override is false, new values update existing
			want:     Params{"a": 1, "b": 999, "c": 3},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.original.Update(tt.other, tt.override)
			if !reflect.DeepEqual(tt.original, tt.want) {
				t.Errorf("Params.Update() = %v, want %v", tt.original, tt.want)
			}
		})
	}
}

func TestParams_UpdateWithout(t *testing.T) {
	original := Params{"a": 1, "b": 2}
	other := Params{"b": 999, "c": 3, "d": 4}
	without := []string{"d"}

	original.UpdateWithout(other, false, without)

	want := Params{"a": 1, "b": 2, "c": 3} // b should not be overridden when override=false
	if !reflect.DeepEqual(original, want) {
		t.Errorf("Params.UpdateWithout() = %v, want %v", original, want)
	}
}

func TestParams_Without(t *testing.T) {
	params := Params{"a": 1, "b": 2, "c": 3, "d": 4}
	params.Without("b", "d")

	want := Params{"a": 1, "c": 3}
	if !reflect.DeepEqual(params, want) {
		t.Errorf("Params.Without() = %v, want %v", params, want)
	}
}

func TestRecord_Fill(t *testing.T) {
	tests := []struct {
		name      string
		record    Record
		container interface{}
		wantErr   bool
		check     func(interface{}) bool
	}{
		{
			name: "fill user struct",
			record: Record{
				"id":     int64(123),
				"name":   "John Doe",
				"email":  "john@example.com",
				"active": true,
				"age":    30,
				"tags":   []string{"admin", "user"},
			},
			container: &TestUser{},
			wantErr:   false,
			check: func(container interface{}) bool {
				user := container.(*TestUser)
				return user.ID == 123 &&
					user.Name == "John Doe" &&
					user.Email == "john@example.com" &&
					user.Active == true &&
					user.Age == 30 &&
					len(user.Tags) == 2
			},
		},
		{
			name:      "nil container",
			record:    Record{"id": 123},
			container: nil,
			wantErr:   true,
		},
		{
			name:      "non-pointer container",
			record:    Record{"id": 123},
			container: TestUser{},
			wantErr:   true,
		},
		{
			name:      "non-struct container",
			record:    Record{"id": 123},
			container: new(string),
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.record.Fill(tt.container)
			if (err != nil) != tt.wantErr {
				t.Errorf("Record.Fill() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.check != nil && !tt.check(tt.container) {
				t.Error("Record.Fill() did not fill container correctly")
			}
		})
	}
}

func TestRecord_RecordID(t *testing.T) {
	tests := []struct {
		name      string
		record    Record
		want      int64
		wantPanic bool
	}{
		{
			name:   "valid int64 id",
			record: Record{"id": int64(123)},
			want:   123,
		},
		{
			name:   "valid float64 id",
			record: Record{"id": float64(123)},
			want:   123,
		},
		{
			name:   "valid int id",
			record: Record{"id": int(123)},
			want:   123,
		},
		{
			name:      "missing id",
			record:    Record{"name": "test"},
			wantPanic: true,
		},
		{
			name:      "invalid id type",
			record:    Record{"id": "not-a-number"},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("Record.RecordID() panic = %v, wantPanic %v", r != nil, tt.wantPanic)
				}
			}()

			got := tt.record.RecordID()
			if !tt.wantPanic && got != tt.want {
				t.Errorf("Record.RecordID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecord_RecordName(t *testing.T) {
	tests := []struct {
		name      string
		record    Record
		want      string
		wantPanic bool
	}{
		{
			name:   "valid name",
			record: Record{"name": "test-name"},
			want:   "test-name",
		},
		{
			name:   "numeric name",
			record: Record{"name": 123},
			want:   "123",
		},
		{
			name:      "missing name",
			record:    Record{"id": 123},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("Record.RecordName() panic = %v, wantPanic %v", r != nil, tt.wantPanic)
				}
			}()

			got := tt.record.RecordName()
			if !tt.wantPanic && got != tt.want {
				t.Errorf("Record.RecordName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecord_RecordGUID(t *testing.T) {
	tests := []struct {
		name      string
		record    Record
		want      string
		wantPanic bool
	}{
		{
			name:   "valid guid",
			record: Record{"guid": "test-guid-123"},
			want:   "test-guid-123",
		},
		{
			name:      "missing guid",
			record:    Record{"id": 123},
			wantPanic: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Errorf("Record.RecordGUID() panic = %v, wantPanic %v", r != nil, tt.wantPanic)
				}
			}()

			got := tt.record.RecordGUID()
			if !tt.wantPanic && got != tt.want {
				t.Errorf("Record.RecordGUID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRecord_SetMissingValue(t *testing.T) {
	record := Record{"existing": "value"}

	// Should set missing key
	record.SetMissingValue("new_key", "new_value")
	if record["new_key"] != "new_value" {
		t.Error("SetMissingValue should set missing key")
	}

	// Should not overwrite existing key
	record.SetMissingValue("existing", "different_value")
	if record["existing"] != "value" {
		t.Error("SetMissingValue should not overwrite existing key")
	}
}

func TestRecord_PrettyTable(t *testing.T) {
	record := Record{
		"id":            123,
		"name":          "test",
		"extra":         "data",
		resourceTypeKey: "TestResource",
	}

	output := record.PrettyTable()

	// Should contain resource type
	if !strings.Contains(output, "TestResource") {
		t.Error("PrettyTable should contain resource type")
	}

	// Should contain printable attributes
	if !strings.Contains(output, "id") || !strings.Contains(output, "123") {
		t.Error("PrettyTable should contain id")
	}
	if !strings.Contains(output, "name") || !strings.Contains(output, "test") {
		t.Error("PrettyTable should contain name")
	}
}

func TestRecord_PrettyJson(t *testing.T) {
	record := Record{
		"id":   123,
		"name": "test",
	}

	// Test without indentation
	output := record.PrettyJson()
	var parsed map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("PrettyJson output should be valid JSON: %v", err)
	}

	// Test with indentation
	indented := record.PrettyJson("  ")
	if !strings.Contains(indented, "  ") {
		t.Error("PrettyJson with indent should contain indentation")
	}
}

func TestRecordSet_Fill(t *testing.T) {
	recordSet := RecordSet{
		{"id": int64(1), "name": "user1", "active": true},
		{"id": int64(2), "name": "user2", "active": false},
	}

	var users []TestUser
	err := recordSet.Fill(&users)
	if err != nil {
		t.Errorf("RecordSet.Fill() error = %v", err)
		return
	}

	if len(users) != 2 {
		t.Errorf("RecordSet.Fill() filled %d users, want 2", len(users))
		return
	}

	if users[0].ID != 1 || users[0].Name != "user1" || users[0].Active != true {
		t.Errorf("RecordSet.Fill() did not fill first user correctly: got ID=%d Name=%q Active=%t, want ID=1 Name=\"user1\" Active=true", users[0].ID, users[0].Name, users[0].Active)
	}

	if users[1].ID != 2 || users[1].Name != "user2" || users[1].Active != false {
		t.Errorf("RecordSet.Fill() did not fill second user correctly: got ID=%d Name=%q Active=%t, want ID=2 Name=\"user2\" Active=false", users[1].ID, users[1].Name, users[1].Active)
	}
}

func TestRecordSet_Fill_Pointers(t *testing.T) {
	recordSet := RecordSet{
		{"id": int64(10), "name": "alpha", "active": true},
		{"id": int64(20), "name": "beta", "active": false},
	}

	var users []*TestUser
	if err := recordSet.Fill(&users); err != nil {
		t.Fatalf("RecordSet.Fill() pointers error = %v", err)
	}

	if len(users) != 2 {
		t.Fatalf("RecordSet.Fill() filled %d users, want 2", len(users))
	}

	if users[0] == nil || users[1] == nil {
		t.Fatal("RecordSet.Fill() should fill non-nil pointers")
	}

	if users[0].ID != 10 || users[0].Name != "alpha" || users[0].Active != true {
		t.Errorf("first user mismatch: %+v", users[0])
	}
	if users[1].ID != 20 || users[1].Name != "beta" || users[1].Active != false {
		t.Errorf("second user mismatch: %+v", users[1])
	}
}

func TestRecordSet_PrettyTable(t *testing.T) {
	recordSet := RecordSet{
		{"id": 1, "name": "user1"},
		{"id": 2, "name": "user2"},
	}

	output := recordSet.PrettyTable()

	// Should contain array indicators
	if !strings.Contains(output, "[") || !strings.Contains(output, "]") {
		t.Error("PrettyTable should contain array indicators")
	}

	// Should contain data from both records
	if !strings.Contains(output, "user1") || !strings.Contains(output, "user2") {
		t.Error("PrettyTable should contain data from all records")
	}
}

func TestRecordSet_Empty(t *testing.T) {
	emptySet := RecordSet{}
	if !emptySet.Empty() {
		t.Error("Empty RecordSet should return true for Empty()")
	}

	nonEmptySet := RecordSet{{"id": 1}}
	if nonEmptySet.Empty() {
		t.Error("Non-empty RecordSet should return false for Empty()")
	}
}

func TestRecordSet_PrettyJson(t *testing.T) {
	recordSet := RecordSet{
		{"id": 1, "name": "user1"},
		{"id": 2, "name": "user2"},
	}

	output := recordSet.PrettyJson()
	var parsed []map[string]interface{}
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Errorf("PrettyJson output should be valid JSON: %v", err)
	}

	if len(parsed) != 2 {
		t.Errorf("Parsed JSON should have 2 items, got %d", len(parsed))
	}
}

func TestEmptyRecord_Methods(t *testing.T) {
	er := EmptyRecord{}

	// Test Fill
	var user TestUser
	err := er.Fill(&user)
	if err != nil {
		t.Errorf("EmptyRecord.Fill() should not error, got %v", err)
	}

	// Test PrettyTable
	output := er.PrettyTable()
	if output != "<->" {
		t.Errorf("EmptyRecord.PrettyTable() = %v, want <->", output)
	}

	// Test PrettyJson
	json := er.PrettyJson()
	if json != "{}" {
		t.Errorf("EmptyRecord.PrettyJson() = %v, want {}", json)
	}

	// Test String
	str := er.String()
	if str != "<->" {
		t.Errorf("EmptyRecord.String() = %v, want <->", str)
	}
}

func TestUnmarshalToRecordUnion(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantType   string
		wantErr    bool
	}{
		{
			name:       "empty response",
			statusCode: 204,
			body:       "",
			wantType:   "EmptyRecord",
		},
		{
			name:       "JSON object",
			statusCode: 200,
			body:       `{"id": 123, "name": "test"}`,
			wantType:   "Record",
		},
		{
			name:       "JSON array",
			statusCode: 200,
			body:       `[{"id": 1}, {"id": 2}]`,
			wantType:   "RecordSet",
		},
		{
			name:       "string response",
			statusCode: 200,
			body:       `"OK"`,
			wantType:   "Record",
		},
		{
			name:       "invalid JSON",
			statusCode: 200,
			body:       `invalid json`,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP response
			response := &http.Response{
				StatusCode:    tt.statusCode,
				Body:          &MockReadCloser{strings.NewReader(tt.body)},
				ContentLength: int64(len(tt.body)),
			}

			result, err := unmarshalToRecordUnion(response)
			if (err != nil) != tt.wantErr {
				t.Errorf("unmarshalToRecordUnion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				resultType := reflect.TypeOf(result).Name()
				if resultType != tt.wantType {
					t.Errorf("unmarshalToRecordUnion() type = %v, want %v", resultType, tt.wantType)
				}
			}
		})
	}
}

func TestTypeMatch(t *testing.T) {
	record := Record{"id": 123}
	recordSet := RecordSet{{"id": 1}, {"id": 2}}
	emptyRecord := EmptyRecord{}

	// Test Record matching
	if !typeMatch[Record](record) {
		t.Error("typeMatch should return true for matching Record type")
	}
	if typeMatch[RecordSet](record) {
		t.Error("typeMatch should return false for non-matching RecordSet type")
	}

	// Test RecordSet matching
	if !typeMatch[RecordSet](recordSet) {
		t.Error("typeMatch should return true for matching RecordSet type")
	}
	if typeMatch[Record](recordSet) {
		t.Error("typeMatch should return false for non-matching Record type")
	}

	// Test EmptyRecord matching
	if !typeMatch[EmptyRecord](emptyRecord) {
		t.Error("typeMatch should return true for matching EmptyRecord type")
	}
	if typeMatch[Record](emptyRecord) {
		t.Error("typeMatch should return false for non-matching Record type")
	}
}

func TestSetResourceKey(t *testing.T) {
	// Test Record
	record := Record{"id": 123}
	err := setResourceKey(record, "TestResource")
	if err != nil {
		t.Errorf("setResourceKey() error = %v", err)
	}
	if record[resourceTypeKey] != "TestResource" {
		t.Error("setResourceKey should set resource type key in Record")
	}

	// Test RecordSet
	recordSet := RecordSet{{"id": 1}, {"id": 2}}
	err = setResourceKey(recordSet, "TestResource")
	if err != nil {
		t.Errorf("setResourceKey() error = %v", err)
	}
	for _, rec := range recordSet {
		if rec[resourceTypeKey] != "TestResource" {
			t.Error("setResourceKey should set resource type key in all RecordSet items")
		}
	}

	// Test EmptyRecord
	emptyRecord := EmptyRecord{}
	err = setResourceKey(emptyRecord, "TestResource")
	if err != nil {
		t.Errorf("setResourceKey() error = %v", err)
	}
}

// Mock ReadCloser for testing
type MockReadCloser struct {
	*strings.Reader
}

func (m *MockReadCloser) Close() error {
	return nil
}

// ------------------------------------------------------
// Tests for pagination envelope normalization in defaultResponseMutations

func TestDefaultResponseMutations_UnwrapsPaginationEnvelope_MapsList(t *testing.T) {
	// Prepare a RecordSet wrapping a paginated envelope
	envelope := Record{
		"results": []map[string]any{
			{"id": 1, "name": "a"},
			{"id": 2, "name": "b"},
		},
		"count":    2,
		"next":     nil,
		"previous": nil,
	}
	wrapped := RecordSet{envelope}

	out, err := defaultResponseMutations(wrapped)
	if err != nil {
		t.Fatalf("defaultResponseMutations error: %v", err)
	}

	rs, ok := out.(RecordSet)
	if !ok {
		t.Fatalf("expected RecordSet, got %T", out)
	}
	if len(rs) != 2 {
		t.Fatalf("expected 2 results, got %d", len(rs))
	}
	if rs[0]["id"] != 1 || rs[0]["name"] != "a" || rs[1]["id"] != 2 || rs[1]["name"] != "b" {
		t.Fatalf("unexpected unwrapped content: %+v", rs)
	}
}

func TestDefaultResponseMutations_UnwrapsPaginationEnvelope_AnyList(t *testing.T) {
	// Same as above but "results" is []any of maps
	envelope := Record{
		"results": []any{
			map[string]any{"id": 10, "name": "x"},
			map[string]any{"id": 20, "name": "y"},
		},
		"count":    2,
		"next":     nil,
		"previous": nil,
	}
	wrapped := RecordSet{envelope}

	out, err := defaultResponseMutations(wrapped)
	if err != nil {
		t.Fatalf("defaultResponseMutations error: %v", err)
	}

	rs, ok := out.(RecordSet)
	if !ok {
		t.Fatalf("expected RecordSet, got %T", out)
	}
	if len(rs) != 2 {
		t.Fatalf("expected 2 results, got %d", len(rs))
	}
	if rs[0]["id"].(int) != 10 || rs[0]["name"].(string) != "x" || rs[1]["id"].(int) != 20 || rs[1]["name"].(string) != "y" {
		t.Fatalf("unexpected unwrapped content: %+v", rs)
	}
}

func TestDefaultResponseMutations_NoUnwrap_WhenMissingKeys(t *testing.T) {
	// Missing required keys (only results present) should NOT unwrap
	envelope := Record{
		"results": []map[string]any{
			{"id": 1},
		},
		// missing: count, next, previous
	}
	wrapped := RecordSet{envelope}

	out, err := defaultResponseMutations(wrapped)
	if err != nil {
		t.Fatalf("defaultResponseMutations error: %v", err)
	}

	// Should remain as the original single-item RecordSet (the envelope)
	rs, ok := out.(RecordSet)
	if !ok {
		t.Fatalf("expected RecordSet, got %T", out)
	}
	if len(rs) != 1 {
		t.Fatalf("expected 1 envelope record, got %d", len(rs))
	}
}

func TestDefaultResponseMutations_UnwrapsPaginationEnvelope_Record_Mixed(t *testing.T) {
	envelope := Record{
		"results": []any{
			map[string]any{"id": 100, "name": "r1"},
			map[string]any{"id": 200, "name": "r2"},
		},
		"count":    2,
		"next":     nil,
		"previous": nil,
	}
	out, err := defaultResponseMutations(envelope)
	if err != nil {
		t.Fatalf("defaultResponseMutations error: %v", err)
	}
	rs, ok := out.(RecordSet)
	if !ok {
		t.Fatalf("expected RecordSet, got %T", out)
	}
	if len(rs) != 2 {
		t.Fatalf("expected 2 results, got %d", len(rs))
	}
	if rs[0]["id"].(int) != 100 || rs[1]["id"].(int) != 200 {
		t.Fatalf("unexpected unwrapped content: %+v", rs)
	}
}

func TestDefaultResponseMutations_NoUnwrap_Record_WhenMissingKeys(t *testing.T) {
	envelope := Record{
		"results": []map[string]any{{"id": 1}},
		// missing count/next/previous
	}
	out, err := defaultResponseMutations(envelope)
	if err != nil {
		t.Fatalf("defaultResponseMutations error: %v", err)
	}
	if _, ok := out.(Record); !ok {
		t.Fatalf("expected Record to remain intact, got %T", out)
	}
}

func TestStructToMap_Normal(t *testing.T) {
	sample := SampleStruct{
		Name: "John Doe",
		ID:   "12121",
	}

	res := structToMap(sample)
	if res == nil {
		t.Fatal("expected non-nil result")
	}

	t.Logf("Result: %+v", res)
	// Expected: map[name:John Doe id:12121]

	jbyt, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	t.Logf("JSON: %s", string(jbyt))
	// Expected: {"id":"12121","name":"John Doe"}

	// Verify the content
	if res["name"] != "John Doe" {
		t.Errorf("expected name 'John Doe', got %v", res["name"])
	}
	if res["id"] != "12121" {
		t.Errorf("expected id '12121', got %v", res["id"])
	}
}

func TestStructToMap_FieldStruct(t *testing.T) {
	sample := &SampleStruct{
		Name: "John Doe",
		ID:   "12121",
	}
	field := FieldStruct{
		Sample:   sample,
		OnePoint: "yuhuhuu",
	}

	res := structToMap(field)
	if res == nil {
		t.Fatal("expected non-nil result")
	}

	t.Logf("Result: %+v", res)
	// Expected: map[sample:0xc4200f04a0 one_point:yuhuhuu]

	jbyt, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	t.Logf("JSON: %s", string(jbyt))
	// Expected: {"one_point":"yuhuhuu","sample":{"name":"John Doe","id":"12121"}}

	// Verify the content
	if res["one_point"] != "yuhuhuu" {
		t.Errorf("expected one_point 'yuhuhuu', got %v", res["one_point"])
	}

	// Verify nested struct
	sampleMap, ok := res["sample"].(map[string]interface{})
	if !ok {
		t.Errorf("expected sample to be map[string]interface{}, got %T", res["sample"])
	} else {
		if sampleMap["name"] != "John Doe" {
			t.Errorf("expected nested name 'John Doe', got %v", sampleMap["name"])
		}
		if sampleMap["id"] != "12121" {
			t.Errorf("expected nested id '12121', got %v", sampleMap["id"])
		}
	}
}

func TestStructToMap_EmbeddedStruct(t *testing.T) {
	sample := &SampleStruct{
		Name: "John Doe",
		ID:   "12121",
	}
	field := FieldStruct{
		Sample:   sample,
		OnePoint: "yuhuhuu",
	}

	embed := EmbededStruct{
		FieldStruct: field,
		Hello:       "WORLD!!!!",
	}

	res := structToMap(embed)
	if res == nil {
		t.Fatal("expected non-nil result")
	}

	t.Logf("Result: %+v", res)
	// Expected: map[field:map[one_point:yuhuhuu sample:0xc420106420] hello:WORLD!!!!]

	jbyt, err := json.Marshal(res)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	t.Logf("JSON: %s", string(jbyt))
	// Expected: {"field":{"one_point":"yuhuhuu","sample":{"name":"John Doe","id":"12121"}},"hello":"WORLD!!!!"}

	// Verify the content
	if res["hello"] != "WORLD!!!!" {
		t.Errorf("expected hello 'WORLD!!!!', got %v", res["hello"])
	}

	// Verify nested field struct
	fieldMap, ok := res["field"].(map[string]interface{})
	if !ok {
		t.Errorf("expected field to be map[string]interface{}, got %T", res["field"])
	} else {
		if fieldMap["one_point"] != "yuhuhuu" {
			t.Errorf("expected nested one_point 'yuhuhuu', got %v", fieldMap["one_point"])
		}

		// Verify deeply nested struct
		sampleMap, ok := fieldMap["sample"].(map[string]interface{})
		if !ok {
			t.Errorf("expected nested sample to be map[string]interface{}, got %T", fieldMap["sample"])
		} else {
			if sampleMap["name"] != "John Doe" {
				t.Errorf("expected deeply nested name 'John Doe', got %v", sampleMap["name"])
			}
			if sampleMap["id"] != "12121" {
				t.Errorf("expected deeply nested id '12121', got %v", sampleMap["id"])
			}
		}
	}
}

func TestParams_FromStruct(t *testing.T) {
	// Test basic struct conversion
	sample := SampleStruct{
		Name: "John Doe",
		ID:   "12121",
	}

	params := make(Params)
	err := params.FromStruct(sample)
	if err != nil {
		t.Fatalf("FromStruct failed: %v", err)
	}

	if params["name"] != "John Doe" {
		t.Errorf("expected name 'John Doe', got %v", params["name"])
	}
	if params["id"] != "12121" {
		t.Errorf("expected id '12121', got %v", params["id"])
	}
}

func TestParams_FromStruct_WithOmitEmpty(t *testing.T) {
	// Test omitempty behavior
	req := TestRequest{
		Name:     "Bob Smith",
		Age:      40,
		Optional: nil, // This should be omitted due to omitempty
		Score:    92.0,
	}

	params := make(Params)
	err := params.FromStruct(req)
	if err != nil {
		t.Fatalf("FromStruct failed: %v", err)
	}

	// Check that optional field is omitted
	if _, exists := params["optional"]; exists {
		t.Error("expected optional field to be omitted when nil")
	}

	// Check other fields are present
	if params["name"] != "Bob Smith" {
		t.Errorf("expected name 'Bob Smith', got %v", params["name"])
	}
	if params["age"] != 40 {
		t.Errorf("expected age 40, got %v", params["age"])
	}
	if params["score"] != 92.0 {
		t.Errorf("expected score 92.0, got %v", params["score"])
	}
}

func TestParseJSONTag(t *testing.T) {
	tests := []struct {
		tag          string
		expectedName string
		expectedOmit bool
	}{
		{"", "", false},
		{"name", "name", false},
		{"name,omitempty", "name", true},
		{",omitempty", "", true},
		{"-", "-", false},
		{"mobile_no,omitempty", "mobile_no", true},
		{"field,string", "field", false},
		{"field,omitempty,string", "field", true},
		{"field,string,omitempty", "field", true},
		{"custom_name,omitempty", "custom_name", true},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("tag_%s", tt.tag), func(t *testing.T) {
			name, omit := parseJSONTag(tt.tag)
			if name != tt.expectedName {
				t.Errorf("parseJSONTag(%q) name = %q, want %q", tt.tag, name, tt.expectedName)
			}
			if omit != tt.expectedOmit {
				t.Errorf("parseJSONTag(%q) omit = %v, want %v", tt.tag, omit, tt.expectedOmit)
			}
		})
	}
}

func TestStructToMap_OmitEmptyEdgeCases(t *testing.T) {
	// Test the TestUserWithOmit struct from the issue
	user := TestUserWithOmit{
		Uid:         "12345",
		Name:        "", // Empty string should be omitted
		Email:       "test@example.com",
		MobileNo:    "",    // Empty string should be omitted
		Age:         0,     // Zero int should be omitted
		Score:       0.0,   // Zero float should be omitted
		Active:      false, // Zero bool should be omitted
		Shortlisted: nil,   // Nil slice should be omitted
	}

	res := structToMap(user)

	// Should only contain non-zero values
	expected := map[string]interface{}{
		"uid":   "12345",
		"email": "test@example.com",
	}

	if len(res) != len(expected) {
		t.Errorf("expected %d fields, got %d: %+v", len(expected), len(res), res)
	}

	for key, expectedValue := range expected {
		if actualValue, exists := res[key]; !exists {
			t.Errorf("expected key %q not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("key %q: expected %v, got %v", key, expectedValue, actualValue)
		}
	}

	// Verify omitted fields are not present
	omittedFields := []string{"name", "mobile_no", "age", "score", "active", "shortlisted"}
	for _, field := range omittedFields {
		if _, exists := res[field]; exists {
			t.Errorf("expected field %q to be omitted, but it was present with value %v", field, res[field])
		}
	}
}

func TestStructToMap_TagVariations(t *testing.T) {
	// Test various JSON tag formats
	test := TagTestStruct{
		Field1: "value1",
		Field2: "",       // Should be omitted due to omitempty
		Field3: "value3", // Empty tag name with omitempty
		Field4: "value4", // Should be skipped due to "-"
		Field5: "",       // Should be omitted due to omitempty
		Field6: "value6",
		Field7: "value7",
		Field8: "value8", // No tag, should be skipped
	}

	res := structToMap(test)

	// Expected results
	expected := map[string]interface{}{
		"field1": "value1",
		// field2 omitted (empty string with omitempty)
		// field3 should not appear (empty tag name)
		// field4 skipped (tag is "-")
		// field5 omitted (empty string with omitempty)
		"field6": "value6",
		"field7": "value7",
		// field8 skipped (no tag)
	}

	if len(res) != len(expected) {
		t.Errorf("expected %d fields, got %d: %+v", len(expected), len(res), res)
	}

	for key, expectedValue := range expected {
		if actualValue, exists := res[key]; !exists {
			t.Errorf("expected key %q not found", key)
		} else if actualValue != expectedValue {
			t.Errorf("key %q: expected %v, got %v", key, expectedValue, actualValue)
		}
	}

	// Verify that fields with problematic tags are handled correctly
	problematicKeys := []string{"field2,omitempty", "mobile_no,omitempty", "-", "field8"}
	for _, key := range problematicKeys {
		if _, exists := res[key]; exists {
			t.Errorf("found problematic key %q in result, this indicates tag parsing failed", key)
		}
	}
}

func TestStructToMap_WithNonZeroOmitEmpty(t *testing.T) {
	// Test that non-zero values are included even with omitempty
	user := TestUserWithOmit{
		Uid:      "12345",
		Name:     "John Doe",
		Email:    "john@example.com",
		MobileNo: "555-1234",
		Age:      30,
		Score:    95.5,
		Active:   true,
		Shortlisted: []ShortList{
			{ID: 1, Name: "Item 1"},
		},
	}

	res := structToMap(user)

	// All fields should be present since they have non-zero values
	expected := map[string]interface{}{
		"uid":         "12345",
		"name":        "John Doe",
		"email":       "john@example.com",
		"mobile_no":   "555-1234",
		"age":         30,
		"score":       95.5,
		"active":      true,
		"shortlisted": []ShortList{{ID: 1, Name: "Item 1"}},
	}

	if len(res) != len(expected) {
		t.Errorf("expected %d fields, got %d: %+v", len(expected), len(res), res)
	}

	for key, expectedValue := range expected {
		if actualValue, exists := res[key]; !exists {
			t.Errorf("expected key %q not found", key)
		} else if !reflect.DeepEqual(actualValue, expectedValue) {
			t.Errorf("key %q: expected %v (%T), got %v (%T)", key, expectedValue, expectedValue, actualValue, actualValue)
		}
	}
}

func TestStructToMap_NestedStructOmitEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    ParentStruct
		expected map[string]interface{}
	}{
		{
			name: "empty nested struct should be omitted",
			input: ParentStruct{
				Name:   "test",
				Nested: NestedStruct{}, // Empty struct should be omitted
				Ptr:    nil,            // Nil pointer should be omitted
			},
			expected: map[string]interface{}{
				"name": "test",
				// nested and ptr should be omitted
			},
		},
		{
			name: "nested struct with values should be included",
			input: ParentStruct{
				Name: "test",
				Nested: NestedStruct{
					Value: "hello",
					Count: 0, // Zero value should be omitted within nested struct
				},
				Ptr: &NestedStruct{
					Value: "", // Empty string should be omitted
					Count: 42, // Non-zero should be included
				},
			},
			expected: map[string]interface{}{
				"name": "test",
				"nested": map[string]interface{}{
					"value": "hello",
					// count omitted because it's zero
				},
				"ptr": map[string]interface{}{
					"count": 42,
					// value omitted because it's empty string
				},
			},
		},
		{
			name: "nested struct with all zero values should be omitted",
			input: ParentStruct{
				Name: "test",
				Nested: NestedStruct{
					Value: "", // Empty string
					Count: 0,  // Zero int
				},
				Ptr: &NestedStruct{
					Value: "", // Empty string
					Count: 0,  // Zero int
				},
			},
			expected: map[string]interface{}{
				"name": "test",
				// Both nested and ptr should be omitted because they contain only zero values
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := structToMap(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d fields, got %d: %+v", len(tt.expected), len(result), result)
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("expected key %q not found", key)
				} else if !reflect.DeepEqual(actualValue, expectedValue) {
					t.Errorf("key %q: expected %v (%T), got %v (%T)", key, expectedValue, expectedValue, actualValue, actualValue)
				}
			}

			// Check that no unexpected keys are present
			for key := range result {
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("unexpected key %q found with value %v", key, result[key])
				}
			}
		})
	}
}

func TestStructToMap_SliceOmitEmpty(t *testing.T) {
	tests := []struct {
		name     string
		input    SliceTestStruct
		expected map[string]interface{}
	}{
		{
			name: "nil slices should be omitted with omitempty",
			input: SliceTestStruct{
				Items:    nil, // Nil slice should be omitted
				Numbers:  nil, // Nil slice should be omitted
				Required: nil, // Nil slice without omitempty should be included
			},
			expected: map[string]interface{}{
				"required": nil, // No omitempty, so nil is included
			},
		},
		{
			name: "empty slices should be omitted with omitempty",
			input: SliceTestStruct{
				Items:    []string{}, // Empty slice should be omitted
				Numbers:  []int{},    // Empty slice should be omitted
				Required: []string{}, // Empty slice without omitempty should be included
			},
			expected: map[string]interface{}{
				"required": []string{}, // No omitempty, so empty slice is included
			},
		},
		{
			name: "non-empty slices should be included",
			input: SliceTestStruct{
				Items:    []string{"a", "b"},
				Numbers:  []int{1, 2, 3},
				Required: []string{"x"},
			},
			expected: map[string]interface{}{
				"items":    []string{"a", "b"},
				"numbers":  []int{1, 2, 3},
				"required": []string{"x"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := structToMap(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d fields, got %d: %+v", len(tt.expected), len(result), result)
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("expected key %q not found", key)
				} else if !reflect.DeepEqual(actualValue, expectedValue) {
					t.Errorf("key %q: expected %v (%T), got %v (%T)", key, expectedValue, expectedValue, actualValue, actualValue)
				}
			}

			// Check that no unexpected keys are present
			for key := range result {
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("unexpected key %q found with value %v", key, result[key])
				}
			}
		})
	}
}

func TestStructToMap_ComplexOmitEmptyScenarios(t *testing.T) {
	// Test the exact scenario from the GitHub issue
	type User struct {
		Uid         string      `json:"uid,omitempty"`
		Name        string      `json:"name,omitempty"`
		Email       string      `json:"email,omitempty"`
		MobileNo    string      `json:"mobile_no,omitempty"`
		Shortlisted []ShortList `json:"shortlisted,omitempty"`
	}

	tests := []struct {
		name     string
		input    User
		expected map[string]interface{}
	}{
		{
			name: "all zero values should be omitted",
			input: User{
				Uid:         "",  // Empty string
				Name:        "",  // Empty string
				Email:       "",  // Empty string
				MobileNo:    "",  // Empty string
				Shortlisted: nil, // Nil slice
			},
			expected: map[string]interface{}{
				// All fields should be omitted
			},
		},
		{
			name: "only non-zero values should be included",
			input: User{
				Uid:         "12345",
				Name:        "", // Should be omitted
				Email:       "test@example.com",
				MobileNo:    "",                                  // Should be omitted
				Shortlisted: []ShortList{{ID: 1, Name: "item1"}}, // Non-empty slice
			},
			expected: map[string]interface{}{
				"uid":         "12345",
				"email":       "test@example.com",
				"shortlisted": []ShortList{{ID: 1, Name: "item1"}},
			},
		},
		{
			name: "empty slice should be omitted",
			input: User{
				Uid:         "12345",
				Name:        "John",
				Email:       "john@example.com",
				MobileNo:    "555-1234",
				Shortlisted: []ShortList{}, // Empty slice should be omitted
			},
			expected: map[string]interface{}{
				"uid":       "12345",
				"name":      "John",
				"email":     "john@example.com",
				"mobile_no": "555-1234",
				// shortlisted should be omitted
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := structToMap(tt.input)

			if len(result) != len(tt.expected) {
				t.Errorf("expected %d fields, got %d: %+v", len(tt.expected), len(result), result)
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := result[key]; !exists {
					t.Errorf("expected key %q not found", key)
				} else if !reflect.DeepEqual(actualValue, expectedValue) {
					t.Errorf("key %q: expected %v (%T), got %v (%T)", key, expectedValue, expectedValue, actualValue, actualValue)
				}
			}

			// Check that no unexpected keys are present
			for key := range result {
				if _, expected := tt.expected[key]; !expected {
					t.Errorf("unexpected key %q found with value %v", key, result[key])
				}
			}

			// Verify the key format is correct (no "mobile_no,omitempty")
			for key := range result {
				if strings.Contains(key, ",") {
					t.Errorf("found malformed key %q - JSON tag parsing failed", key)
				}
			}
		})
	}
}

func TestNewParamsFromStruct(t *testing.T) {
	input := TestRequest{
		Name:     "Alice Johnson",
		Age:      35,
		Optional: boolPtr(false),
		Score:    87.5,
	}

	params, err := NewParamsFromStruct(input)
	if err != nil {
		t.Fatalf("NewParamsFromStruct() error = %v", err)
	}

	expected := Params{
		"name":     "Alice Johnson",
		"age":      35, // int, not float64 - our reflection-based approach preserves Go types
		"optional": false,
		"score":    87.5,
	}

	if !reflect.DeepEqual(params, expected) {
		t.Errorf("NewParamsFromStruct() = %v, expected %v", params, expected)
	}
}

func TestParams_FromStruct_InvalidInput(t *testing.T) {
	params := make(Params)

	// Test with nil input
	err := params.FromStruct(nil)
	if err != nil {
		t.Errorf("expected no error for nil input, got %v", err)
	}

	// Test with non-struct input (should return empty map)
	err = params.FromStruct("not a struct")
	if err != nil {
		t.Errorf("expected no error for non-struct input, got %v", err)
	}

	// Params should remain empty for non-struct input
	if len(params) != 0 {
		t.Errorf("expected empty params for non-struct input, got %v", params)
	}
}

// Helper function to create a bool pointer
func boolPtr(b bool) *bool {
	return &b
}

// =============================================================================
// COMPREHENSIVE TESTS FOR NewParamsFromStruct vs JSON MARSHALING
// =============================================================================

// Test structures for comprehensive omitempty testing
type OmitTestStruct struct {
	// Basic types with omitempty
	Name   string  `json:"name,omitempty"`
	Age    int     `json:"age,omitempty"`
	Score  float64 `json:"score,omitempty"`
	Active bool    `json:"active,omitempty"`

	// Slices with omitempty
	Tags    []string `json:"tags,omitempty"`
	Numbers []int    `json:"numbers,omitempty"`

	// Nested struct with omitempty
	Profile NestedProfile `json:"profile,omitempty"`

	// Pointer to nested struct with omitempty
	Settings *NestedSettings `json:"settings,omitempty"`

	// Fields without omitempty (should always be included)
	ID       int64  `json:"id"`
	Required string `json:"required"`
}

type NestedProfile struct {
	Bio     string `json:"bio,omitempty"`
	Website string `json:"website,omitempty"`
	Age     int    `json:"age,omitempty"`
}

type NestedSettings struct {
	Theme         string `json:"theme,omitempty"`
	Language      string `json:"language,omitempty"`
	Notifications bool   `json:"notifications,omitempty"`
}

// Helper function to compare NewParamsFromStruct result with JSON marshaling
// This focuses on semantic equivalence rather than exact type matching
func compareWithJSON(t *testing.T, testName string, input interface{}) {
	t.Helper()

	// Get result from NewParamsFromStruct
	params, err := NewParamsFromStruct(input)
	if err != nil {
		t.Fatalf("%s: NewParamsFromStruct failed: %v", testName, err)
	}

	// Get result from JSON marshaling
	jsonBytes, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("%s: JSON marshal failed: %v", testName, err)
	}

	var jsonResult map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &jsonResult); err != nil {
		t.Fatalf("%s: JSON unmarshal failed: %v", testName, err)
	}

	// Compare keys (should be identical)
	if len(params) != len(jsonResult) {
		t.Errorf("%s: Different number of keys", testName)
		t.Errorf("  NewParamsFromStruct keys: %v", getKeys(params))
		t.Errorf("  JSON marshaling keys:     %v", getKeys(jsonResult))
		return
	}

	// Check that all keys exist in both
	for key := range params {
		if _, exists := jsonResult[key]; !exists {
			t.Errorf("%s: Key '%s' exists in params but not in JSON", testName, key)
		}
	}
	for key := range jsonResult {
		if _, exists := params[key]; !exists {
			t.Errorf("%s: Key '%s' exists in JSON but not in params", testName, key)
		}
	}

	// For semantic comparison, we mainly care about:
	// 1. Same keys are present/absent (omitempty behavior)
	// 2. Values are semantically equivalent (not necessarily same type)

	// The key insight: our function should behave like JSON marshaling for omitempty,
	// but it's OK to preserve Go types instead of converting to JSON types
	if len(params) != len(jsonResult) {
		t.Errorf("%s: Key sets don't match - this indicates omitempty behavior differs", testName)
		t.Errorf("  NewParamsFromStruct: %+v", map[string]interface{}(params))
		t.Errorf("  JSON marshaling:     %+v", jsonResult)
	}
}

// Helper function to get map keys
func getKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestNewParamsFromStruct_ComprehensiveOmitEmpty(t *testing.T) {
	tests := []struct {
		name        string
		input       OmitTestStruct
		description string
	}{
		{
			name: "AllZeroValues",
			input: OmitTestStruct{
				// All omitempty fields are zero values - should be omitted
				Name:     "",
				Age:      0,
				Score:    0.0,
				Active:   false,
				Tags:     nil,
				Numbers:  nil,
				Profile:  NestedProfile{}, // Empty nested struct
				Settings: nil,
				// Non-omitempty fields - should be included even if zero
				ID:       0,
				Required: "",
			},
			description: "All omitempty fields should be omitted, non-omitempty fields included",
		},
		{
			name: "SomeNonZeroValues",
			input: OmitTestStruct{
				Name:     "John",                          // Non-zero string - should be included
				Age:      0,                               // Zero int - should be omitted
				Score:    85.5,                            // Non-zero float - should be included
				Active:   false,                           // Zero bool - should be omitted
				Tags:     []string{"go", "test"},          // Non-empty slice - should be included
				Numbers:  []int{},                         // Empty slice - should be omitted
				Profile:  NestedProfile{Bio: "Developer"}, // Nested struct with non-zero field
				Settings: &NestedSettings{Theme: "dark"},  // Pointer to struct with non-zero field
				ID:       123,
				Required: "test",
			},
			description: "Mix of zero and non-zero values",
		},
		{
			name: "EmptyNestedStruct",
			input: OmitTestStruct{
				Name:     "Jane",
				Profile:  NestedProfile{},   // All fields are zero - should be omitted
				Settings: &NestedSettings{}, // All fields are zero - should be omitted
				ID:       456,
				Required: "required",
			},
			description: "Nested structs with all zero values should be omitted",
		},
		{
			name: "PartiallyFilledNestedStruct",
			input: OmitTestStruct{
				Name: "Bob",
				Profile: NestedProfile{
					Bio:     "",                    // Zero value
					Website: "https://example.com", // Non-zero value
					Age:     0,                     // Zero value
				}, // Should be included because Website is non-zero
				Settings: &NestedSettings{
					Theme:         "",    // Zero value
					Language:      "",    // Zero value
					Notifications: false, // Zero value
				}, // Should be omitted because all fields are zero
				ID:       789,
				Required: "test",
			},
			description: "Nested struct included if any field is non-zero, omitted if all are zero",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compareWithJSON(t, tt.description, tt.input)
		})
	}
}

func TestNewParamsFromStruct_SliceHandling(t *testing.T) {
	type SliceTestStruct struct {
		EmptySlice    []string `json:"empty_slice,omitempty"`
		NilSlice      []string `json:"nil_slice,omitempty"`
		NonEmptySlice []string `json:"non_empty_slice,omitempty"`
		RequiredSlice []string `json:"required_slice"` // No omitempty
	}

	tests := []struct {
		name  string
		input SliceTestStruct
	}{
		{
			name: "NilSlices",
			input: SliceTestStruct{
				EmptySlice:    nil,
				NilSlice:      nil,
				NonEmptySlice: nil,
				RequiredSlice: nil,
			},
		},
		{
			name: "EmptySlices",
			input: SliceTestStruct{
				EmptySlice:    []string{},
				NilSlice:      []string{},
				NonEmptySlice: []string{},
				RequiredSlice: []string{},
			},
		},
		{
			name: "MixedSlices",
			input: SliceTestStruct{
				EmptySlice:    []string{},
				NilSlice:      nil,
				NonEmptySlice: []string{"item1", "item2"},
				RequiredSlice: []string{"required"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compareWithJSON(t, "Slice handling: "+tt.name, tt.input)
		})
	}
}

func TestNewParamsFromStruct_PointerHandling(t *testing.T) {
	type PointerTestStruct struct {
		NilPointer    *string `json:"nil_pointer,omitempty"`
		NonNilPointer *string `json:"non_nil_pointer,omitempty"`
		RequiredPtr   *string `json:"required_ptr"` // No omitempty
	}

	nonNilValue := "test"
	requiredValue := "required"

	tests := []struct {
		name  string
		input PointerTestStruct
	}{
		{
			name: "AllNilPointers",
			input: PointerTestStruct{
				NilPointer:    nil,
				NonNilPointer: nil,
				RequiredPtr:   nil,
			},
		},
		{
			name: "MixedPointers",
			input: PointerTestStruct{
				NilPointer:    nil,
				NonNilPointer: &nonNilValue,
				RequiredPtr:   &requiredValue,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compareWithJSON(t, "Pointer handling: "+tt.name, tt.input)
		})
	}
}

func TestNewParamsFromStruct_ComplexNestedStructures(t *testing.T) {
	type DeepNestedStruct struct {
		Level1 struct {
			Level2 struct {
				Level3 struct {
					Value string `json:"value,omitempty"`
				} `json:"level3,omitempty"`
				Name string `json:"name,omitempty"`
			} `json:"level2,omitempty"`
			ID int `json:"id,omitempty"`
		} `json:"level1,omitempty"`
		TopLevel string `json:"top_level,omitempty"`
	}

	tests := []struct {
		name  string
		input DeepNestedStruct
	}{
		{
			name:  "AllEmpty",
			input: DeepNestedStruct{
				// All nested fields are zero values
			},
		},
		{
			name: "DeepValueSet",
			input: DeepNestedStruct{
				Level1: struct {
					Level2 struct {
						Level3 struct {
							Value string `json:"value,omitempty"`
						} `json:"level3,omitempty"`
						Name string `json:"name,omitempty"`
					} `json:"level2,omitempty"`
					ID int `json:"id,omitempty"`
				}{
					Level2: struct {
						Level3 struct {
							Value string `json:"value,omitempty"`
						} `json:"level3,omitempty"`
						Name string `json:"name,omitempty"`
					}{
						Level3: struct {
							Value string `json:"value,omitempty"`
						}{
							Value: "deep value", // Only this is set
						},
					},
				},
				TopLevel: "top",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compareWithJSON(t, "Complex nested: "+tt.name, tt.input)
		})
	}
}

func TestNewParamsFromStruct_RealWorldExample(t *testing.T) {
	// Simulate real-world typed resource structs like QuotaSearchParams
	type QuotaSearchParams struct {
		Name                string `json:"name,omitempty"`
		HardLimit           string `json:"hard_limit,omitempty"`
		SoftLimit           string `json:"soft_limit,omitempty"`
		TenantId            int64  `json:"tenant_id,omitempty"`
		ShowUserRules       bool   `json:"show_user_rules,omitempty"`
		TenantNameIcontains string `json:"tenant_name__icontains,omitempty"`
	}

	tests := []struct {
		name  string
		input QuotaSearchParams
	}{
		{
			name:  "EmptySearchParams",
			input: QuotaSearchParams{},
		},
		{
			name: "PartialSearchParams",
			input: QuotaSearchParams{
				Name:     "test-quota",
				TenantId: 1,
				// Other fields are zero values
			},
		},
		{
			name: "FullSearchParams",
			input: QuotaSearchParams{
				Name:                "full-quota",
				HardLimit:           "1TB",
				SoftLimit:           "800GB",
				TenantId:            2,
				ShowUserRules:       true,
				TenantNameIcontains: "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			compareWithJSON(t, "Real-world example: "+tt.name, tt.input)
		})
	}
}
