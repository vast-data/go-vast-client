package vast_client

import (
	"encoding/json"
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
