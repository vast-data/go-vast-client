package vast_client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bndr/gotabulate"
	"io"
	"net/http"
	"reflect"
	"sort"
	"strings"
)

const resourceTypeKey = "@resourceType"

var empty = struct{}{}
var printableAttrs = map[string]struct{}{
	"id":             empty,
	"name":           empty,
	"sys_version":    empty,
	"path":           empty,
	"tenant_id":      empty,
	"nqn":            empty,
	"ip_ranges":      empty,
	"volumes":        empty,
	"nguid":          empty,
	"subsystem_name": empty,
	"size":           empty,
	"block_host":     empty,
	"volume":         empty,
	"state":          empty,
	"access_key":     empty,
	"secret_key":     empty,
}

//  ######################################################
//              FUNCTION PARAMS
//  ######################################################

// Params represents a generic set of key-value parameters,
// used for constructing query strings or request bodies.
type Params map[string]any

// ToQuery serializes the Params into a URL-encoded query string.
// This is useful for GET requests where parameters are passed via the URL.
func (pr *Params) ToQuery() string {
	return convertMapToQuery(*pr)
}

// ToBody serializes the Params into a JSON-encoded io.Reader,
// suitable for use as the body of an HTTP POST, PUT, or PATCH request.
func (pr *Params) ToBody() (io.Reader, error) {
	buffer, err := json.Marshal(*pr)
	if err != nil {
		return nil, err
	}
	return bytes.NewReader(buffer), nil
}

// Update merges another Params map into the original Params.
// If a key already exists and `override` is true, its value is skipped.
// If a key doesn't exist, the key-value pair is added.
func (pr *Params) Update(other Params, override bool) {
	for key, value := range other {
		// If the key already exists in the original Params and override is false, skip it.
		if _, exists := (*pr)[key]; exists && override {
			continue
		}
		(*pr)[key] = value
	}
}

// UpdateWithout merges another Params map into the original Params.
// If a key exists in the `without` slice, it is skipped.
// If a key already exists and `override` is false, its value is also skipped.
// Otherwise, the key-value pair is added or updated based on the `override` flag.
func (pr *Params) UpdateWithout(other Params, override bool, without []string) {
	for key, value := range other {
		// Skip if the key is in the `without` list
		if contains(without, key) {
			continue
		}
		if _, exists := (*pr)[key]; exists && !override {
			continue
		}
		(*pr)[key] = value
	}
}

//  ######################################################
//              RETURN TYPES
//  ######################################################

// getPrintableAttrs returns a slice of keys to be printed from the Record
func getPrintableAttrs(r Record) []string {
	var attrs []string
	for key := range r {
		if _, ok := printableAttrs[key]; ok {
			attrs = append(attrs, key)
		}
	}
	sort.Strings(attrs) // Sort to keep consistent order
	return attrs
}

// Renderable is an interface implemented by types that can render themselves
// into a human-readable string format, typically for CLI display or logging.
type Renderable interface {
	PrettyTable() string
	PrettyJson(indent ...string) string
}

// Record represents a single generic data object as a key-value map.
// It's commonly used to unmarshal a single JSON object from an API response.
type Record map[string]any

// EmptyRecord represents a placeholder for methods that do not return data,
// such as DELETE operations. It maintains the same structure as Record
// but is used semantically to indicate the absence of returned content.
type EmptyRecord map[string]any

// RecordSet represents a list of Record objects.
// It is typically used to represent responses containing multiple items.
type RecordSet []Record

// RecordUnion defines a union of supported record types for generic operations.
// It can be a single Record, an EmptyRecord, or a RecordSet.
// This allows functions to operate on any supported response type
// using Go generics.
type RecordUnion interface {
	Record | EmptyRecord | RecordSet
}

// Fill populates the exported fields of the given struct pointer using values
// from the Record (a map[string]any). It uses JSON marshaling and unmarshaling
// to automatically map keys to struct fields based on their `json` tags and
// perform type conversions where needed.
//
// The target container must be a non-nil pointer to a struct. Fields in the struct
// must be exported (i.e., start with an uppercase letter) and optionally tagged
// with `json` to match keys in the Record.
//
// JSON-based conversion handles common type mismatches (e.g., string to int, int to string)
// and nested structures if compatible.
//
// Returns an error if the container is not a pointer to a struct or if serialization fails.
func (r Record) Fill(container any) error {
	val := reflect.ValueOf(container)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("container must be a non-nil pointer to a struct")
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return fmt.Errorf("container must point to a struct")
	}
	dbByte, err := json.Marshal(r)
	if err != nil {
		return err
	}
	return json.Unmarshal(dbByte, container)
}

// RecordID returns the ID of the record as an int64.
// It looks up the "id" field in the record map.
func (r Record) RecordID() int64 {
	idVal, ok := r["id"]
	if !ok {
		panic(fmt.Sprintf("record id not found in record %s", r.PrettyTable()))
	}
	intIdVal, err := toInt(idVal)
	if err != nil {
		panic(err)
	}
	return intIdVal
}

// RecordGUID returns the name of the record as a string.
// It looks up the "name" field in the record map.
func (r Record) RecordGUID() string {
	nameVal, ok := r["guid"]
	if !ok {
		panic(fmt.Sprintf("GUID not found in record %s", r.PrettyTable()))
	}
	return fmt.Sprintf("%v", nameVal)
}

// RecordTenantID returns the ID of the record as an int64.
// It looks up the "id" field in the record map.
func (r Record) RecordTenantID() int64 {
	idVal, ok := r["tenant_id"]
	if !ok {
		panic(fmt.Sprintf("record tenant_id not found in record %s", r.PrettyTable()))
	}
	intIdVal, err := toInt(idVal)
	if err != nil {
		panic(err)
	}
	return intIdVal
}

// RecordName returns the name of the record as a string.
// It looks up the "name" field in the record map.
func (r Record) RecordName() string {
	nameVal, ok := r["name"]
	if !ok {
		panic(fmt.Sprintf("record name not found in record %s", r.PrettyTable()))
	}
	return fmt.Sprintf("%v", nameVal)
}

// RecordTenantName returns the name of the tenant as a string.
// It looks up the "name" field in the record map.
func (r Record) RecordTenantName() string {
	nameVal, ok := r["tenant_name"]
	if !ok {
		panic(fmt.Sprintf("tenant_name not found in record %s", r.PrettyTable()))
	}
	return fmt.Sprintf("%v", nameVal)
}

// PrettyTable prints a single Record as a table
func (r Record) PrettyTable() string {
	headers := []string{"attr", "value"}
	var rows [][]any
	var name string
	if resourceTyp, ok := r[resourceTypeKey]; ok {
		name = resourceTyp.(string)
	}
	if len(r) == 0 {
		return "<>"
	}
	// Iterate over printable attributes and add them to rows
	for _, key := range getPrintableAttrs(r) {
		if val, ok := r[key]; ok && val != nil {
			rows = append(rows, []any{key, fmt.Sprintf("%v", val)})
		}
	}

	// Collect remaining attributes that are not in printableAttrs
	remainingAttrs := make(map[string]any)
	for key, value := range r {
		if _, ok := printableAttrs[key]; !ok {
			if key == resourceTypeKey || value == nil {
				continue
			}
			remainingAttrs[key] = value
		}
	}
	if len(remainingAttrs) > 0 {
		// Marshal remainingAttrs into compact JSON
		remainingJSON, _ := json.Marshal(remainingAttrs)
		remainingJSONStr := string(remainingJSON)
		rows = append(rows, []any{"<<remaining attrs>>", remainingJSONStr})
	}
	t := gotabulate.Create(rows)
	t.SetHeaders(headers)
	t.SetAlign("left")
	t.SetWrapStrings(true)
	t.SetMaxCellSize(85)
	if name != "" {
		return fmt.Sprintf("%s:\n%s", name, t.Render("grid"))
	} else {
		return fmt.Sprintf("%s", t.Render("grid"))
	}
}

// PrettyJson prints the Record as JSON, optionally indented
func (r Record) PrettyJson(indent ...string) string {
	var b []byte
	var err error
	if len(indent) > 0 {
		b, err = json.MarshalIndent(r, "", indent[0])
	} else {
		b, err = json.Marshal(r)
	}
	if err != nil {
		return fmt.Sprintf("failed to marshal JSON: %v", err)
	}
	return string(b)
}

func (r Record) empty() bool {
	return len(r) == 0
}

func (r Record) String() string {
	return r.PrettyTable()
}

// PrettyTable prints the full RecordSet by rendering each individual Record
func (rs RecordSet) PrettyTable() string {
	if len(rs) == 0 {
		return "[]"
	}
	var out strings.Builder
	out.WriteString("[\n")
	for i, record := range rs {
		out.WriteString(record.PrettyTable())
		if i < len(rs)-1 {
			out.WriteString("\n\n") // separate entries with a blank line
		}
	}
	out.WriteString("\n]")
	return out.String()
}

func (rs RecordSet) Empty() bool {
	return len(rs) == 0
}

// PrettyJson prints the Record as JSON, optionally indented
func (rs RecordSet) PrettyJson(indent ...string) string {
	var b []byte
	var err error
	if len(indent) > 0 {
		b, err = json.MarshalIndent(rs, "", indent[0])
	} else {
		b, err = json.Marshal(rs)
	}
	if err != nil {
		return fmt.Sprintf("failed to marshal JSON: %v", err)
	}
	return string(b)
}

// PrettyTable EmptyRecord
func (er EmptyRecord) PrettyTable() string {
	return "<->"
}

func (er EmptyRecord) PrettyJson(indent ...string) string {
	return "{}"
}

func (er EmptyRecord) String() string {
	return er.PrettyTable()
}

// unmarshalToRecordUnion parses an HTTP response body into one of the supported record types:
// - EmptyRecord: returned when the response body is empty or the status code is 204 No Content.
// - Record: a map representing a single JSON object.
// - RecordSet: a slice of Records representing a JSON array.
//
// It inspects the first non-whitespace character of the response body to determine whether
// to unmarshal it into a Record or RecordSet. If the JSON format is unsupported (i.e., not an object or array),
// an error is returned.
func unmarshalToRecordUnion(response *http.Response) (Renderable, error) {
	defer response.Body.Close()

	// Handle empty response
	if response.ContentLength == 0 || response.StatusCode == http.StatusNoContent {
		return EmptyRecord{}, nil
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	// Check first non-whitespace character
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return EmptyRecord{}, nil
	}
	switch trimmed[0] {
	case '{': // JSON object
		var rec Record
		if err := json.Unmarshal(body, &rec); err != nil {
			return nil, err
		}
		return rec, nil
	case '[': // JSON array
		var recSet RecordSet
		if err := json.Unmarshal(body, &recSet); err != nil {
			return nil, err
		}
		return recSet, nil
	default:
		return nil, fmt.Errorf("unsupported JSON format: must be object or array")
	}
}

// applyCallbackForRecordUnion applies the provided callback function to a response if
// the response type matches the specified generic type T. It supports different types
// of Renderable responses (Record, RecordSet, and EmptyRecord), and will only apply the
// callback for the exact type matching the generic type T.
func applyCallbackForRecordUnion[T RecordUnion](response Renderable, callback func(Renderable) (Renderable, error)) (Renderable, error) {
	switch typed := response.(type) {
	case Record:
		var zero T
		if _, ok := any(zero).(Record); ok {
			return callback(typed)
		}
		return typed, nil

	case RecordSet:
		var zero T
		if _, ok := any(zero).(RecordSet); ok {
			return callback(typed)
		}
		return typed, nil

	case EmptyRecord:
		var zero T
		if _, ok := any(zero).(EmptyRecord); ok {
			return callback(typed)
		}
		return typed, nil

	default:
		return nil, fmt.Errorf("unsupported type %T for result", response)
	}
}

// typeMatch checks whether the dynamic type of given Renderable value
// matches the generic type T at runtime.
//
// It is typically used to determine if a response object corresponds to
// a specific expected data type (e.g., RecordSet, Record, or EmptyRecord).
//
// This function works by comparing the runtime type of the provided `val`
// against the zero value of the generic type T using reflection.
//
// Example usage:
//
//	if typeMatch[RecordSet](someRenderable) {
//	    // val is of type RecordSet
//	}
func typeMatch[T RecordUnion](val Renderable) bool {
	var zero T
	return reflect.TypeOf(val) == reflect.TypeOf(zero)
}

// setResourceKey Set resource type key for tabular formatting (only if not already set).
func setResourceKey(result Renderable, resourceType string) error {
	switch v := result.(type) {
	case Record:
		if _, ok := v[resourceTypeKey]; !ok && len(v) > 0 {
			v[resourceTypeKey] = resourceType
		}
		return nil
	case RecordSet:
		for _, rec := range v {
			if _, ok := rec[resourceTypeKey]; !ok && len(rec) > 0 {
				rec[resourceTypeKey] = resourceType
			}
		}
		return nil
	case EmptyRecord:
		return nil
	default:
		return fmt.Errorf("unsupported type")
	}
}
