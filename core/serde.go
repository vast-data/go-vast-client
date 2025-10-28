package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"reflect"
	"sort"
	"strings"

	"github.com/bndr/gotabulate"
)

const (
	ResourceTypeKey = "@resourceType"
	customRawKey    = "@raw" // used to store raw string values in Record
)

var empty = struct{}{}
var printableAttrs = map[string]struct{}{
	"id":                 empty,
	"name":               empty,
	"sys_version":        empty,
	"path":               empty,
	"tenant_id":          empty,
	"nqn":                empty,
	"ip_ranges":          empty,
	"volumes":            empty,
	"nguid":              empty,
	"subsystem_name":     empty,
	"size":               empty,
	"block_host":         empty,
	"volume":             empty,
	"state":              empty,
	"access_key":         empty,
	"secret_key":         empty,
	"kadmin_servers":     empty,
	"service_principals": empty,
	"kdc":                empty,
}

type FillFunc func(Record, any) error

var fillFunc FillFunc = func(r Record, container any) error {
	dbByte, err := json.Marshal(r)
	if err != nil {
		return err
	}
	// Use FlexibleUnmarshal to automatically convert numbers to strings for string fields
	return FlexibleUnmarshal(dbByte, container)
}

//  ######################################################
//              FUNCTION PARAMS
//  ######################################################

// Params represents a generic set of key-value parameters,
// used for constructing query strings or request bodies.
type Params map[string]any

// FileData represents a file to be uploaded in multipart form data
type FileData struct {
	Filename string
	Content  []byte
}

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

// MultipartFormData represents the result of ToMultipartFormData()
type MultipartFormData struct {
	Body        io.Reader
	ContentType string
}

// ToMultipartFormData serializes the Params into multipart/form-data format.
// Files should be provided as FileData values in the Params map.
// Returns a MultipartFormData struct containing the body and content type.
func (pr *Params) ToMultipartFormData() (*MultipartFormData, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	for key, value := range *pr {
		switch v := value.(type) {
		case FileData:
			// Handle file uploads
			fileWriter, err := writer.CreateFormFile(key, v.Filename)
			if err != nil {
				return nil, fmt.Errorf("failed to create form file for %s: %w", key, err)
			}
			if _, err := fileWriter.Write(v.Content); err != nil {
				return nil, fmt.Errorf("failed to write file content for %s: %w", key, err)
			}
		case []byte:
			// Handle raw byte data as file without filename
			fileWriter, err := writer.CreateFormFile(key, key)
			if err != nil {
				return nil, fmt.Errorf("failed to create form file for %s: %w", key, err)
			}
			if _, err := fileWriter.Write(v); err != nil {
				return nil, fmt.Errorf("failed to write byte content for %s: %w", key, err)
			}
		default:
			// Handle regular form fields
			if err := writer.WriteField(key, fmt.Sprintf("%v", value)); err != nil {
				return nil, fmt.Errorf("failed to write field %s: %w", key, err)
			}
		}
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return &MultipartFormData{
		Body:        &body,
		ContentType: writer.FormDataContentType(),
	}, nil
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

// Without removes the specified keys from the Params map.
// This is useful when you want to exclude certain parameters before sending a request.
func (pr *Params) Without(keys ...string) {
	for _, key := range keys {
		delete(*pr, key)
	}
}

// FromStruct converts any struct to Params while maintaining the json tags as keys.
// This method uses reflection to directly extract struct fields and their json tags,
// avoiding the overhead of JSON marshaling/unmarshaling.
//
// Example usage:
//
//	type MyRequest struct {
//	    Name     string `json:"name"`
//	    Age      int    `json:"age"`
//	    Optional *bool  `json:"optional,omitempty"`
//	}
//
//	req := MyRequest{Name: "John", Age: 30}
//	params := make(Params)
//	err := params.FromStruct(req)
//	// params now contains: {"name": "John", "age": 30}
//
// Returns an error if the input is not a struct or pointer to struct.
func (pr *Params) FromStruct(obj any) error {
	if obj == nil {
		return nil
	}

	structMap := structToMap(obj)

	// Update the Params map with the new values
	for key, value := range structMap {
		(*pr)[key] = value
	}

	return nil
}

// NewParamsFromStruct creates a new Params map from any struct, respecting json tags.
// This is a convenience function that creates a new Params and calls FromStruct on it.
//
// Special handling for RawData field:
// If the struct has a RawData field (type Params) with len > 0, the RawData map is returned
// directly instead of parsing the struct fields. This allows bypassing typed field parsing
// when custom query parameters are needed.
//
// Example usage:
//
//	type MyRequest struct {
//	    Name string `json:"name"`
//	    Age  int    `json:"age"`
//	    RawData Params `json:"-"`
//	}
//
//	// Using typed fields:
//	req := MyRequest{Name: "John", Age: 30}
//	params, err := NewParamsFromStruct(req)
//	// params contains: {"name": "John", "age": 30}
//
//	// Using RawData (bypasses typed fields):
//	req := MyRequest{RawData: Params{"custom__filter": "value"}}
//	params, err := NewParamsFromStruct(req)
//	// params contains: {"custom__filter": "value"}
//
// Returns a new Params map or an error if the conversion fails.
func NewParamsFromStruct(obj any) (Params, error) {
	params := make(Params)

	// Return empty params if obj is nil
	if obj == nil {
		return params, nil
	}

	// Check if the struct has a RawData field with content
	val := reflect.ValueOf(obj)

	// Dereference pointer if needed
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return params, nil
		}
		val = val.Elem()
	}

	// Only check for RawData if it's a struct
	if val.Kind() == reflect.Struct {
		rawDataField := val.FieldByName("RawData")
		if rawDataField.IsValid() && rawDataField.Type() == reflect.TypeOf(Params{}) {
			rawData, ok := rawDataField.Interface().(Params)
			if ok && len(rawData) > 0 {
				// Return RawData directly, don't parse struct fields
				return rawData, nil
			}
		}
	}

	err := params.FromStruct(obj)
	return params, err
}

// listifyParams converts a slice of arbitrary structs into a slice of Params (map[string]any).
// This is useful when constructing request bodies that require a list of objects, such as
// bulk operations where each entry is a structured parameter set.
//
// Example:
//
//	input: []HostVolumePair{{HostID: 1, VolumeID: 2}, {HostID: 3, VolumeID: 4}}

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

// Filler is a generic interface for filling a struct or slice of structs.
type Filler interface {
	// Fill populates the given container with data from the implementing type.
	// The container can be a pointer to a struct (for Record),
	// or a pointer to a slice of structs (for RecordSet).
	Fill(container any) error
}

// DisplayableRecord defines a unified interface for working with structured data
// that has been deserialized from an API response. It combines both rendering and
// data population capabilities.
//
// Implementing types must support:
//
//   - Rendering themselves as human-readable output via the Renderable interface.
//   - Filling provided container structs or slices using the Filler interface.
//
// This interface is implemented by Record and RecordSet, allowing
// generic handling of different response shapes (single item or list).
type DisplayableRecord interface {
	Renderable
	Filler
}

// Record represents a single generic data object as a key-value map.
// It's commonly used to unmarshal a single JSON object from an API response.
// When a response is empty (e.g., 204 No Content), an empty Record{} is returned.
type Record map[string]any

// RecordSet represents a list of Record objects.
// It is typically used to represent responses containing multiple items.
type RecordSet []Record

// RecordUnion defines a union of supported record types for generic operations.
// It can be a single Record or a RecordSet.
// This allows functions to operate on any supported response type
// using Go generics.
type RecordUnion interface {
	Record | RecordSet
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
	return fillFunc(r, container)
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

// RecordTenantID returns the tenant's ID as an int64.
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
// It looks up the "tenant_name" field in the record map.
func (r Record) RecordTenantName() string {
	nameVal, ok := r["tenant_name"]
	if !ok {
		panic(fmt.Sprintf("tenant_name not found in record %s", r.PrettyTable()))
	}
	return fmt.Sprintf("%v", nameVal)
}

// SetMissingValue If the key is not present in the Record, set it to the provided value
func (r Record) SetMissingValue(key string, value any) {
	if _, exists := r[key]; !exists {
		r[key] = value
	}
}

// PrettyTable prints a single Record as a table
func (r Record) PrettyTable() string {
	headers := []string{"attr", "value"}
	var rows [][]any
	var name string
	if resourceTyp, ok := r[ResourceTypeKey]; ok {
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
			if key == ResourceTypeKey || value == nil {
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
	}
	return fmt.Sprintf("\n%s", t.Render("grid"))
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

func (r Record) Empty() bool {
	return len(r) == 0
}

func (r Record) String() string {
	return r.PrettyTable()
}

// Fill populates the provided container slice with data from the RecordSet.
// The container must be a non-nil pointer to a slice of structs. Each Record in the RecordSet
// is individually marshaled into an element of the slice using JSON serialization,
// and appended to the resulting slice.
//
// Example usage:
//
//	var users []User
//	err := recordSet.Fill(&users)
//	if err != nil {
//	    // handle error
//	}
//
// Parameters:
//   - container: must be a pointer to a slice of structs (e.g., *[]T or *[]*T).
//
// Returns an error if:
//   - The container is not a non-nil pointer to a slice.
//   - The slice element type is not a struct.
//   - Any Record in the RecordSet fails to unmarshal into an element.
func (rs RecordSet) Fill(container any) error {
	val := reflect.ValueOf(container)
	if val.Kind() != reflect.Ptr || val.IsNil() {
		return fmt.Errorf("container must be a non-nil pointer to a slice")
	}

	sliceVal := val.Elem()
	if sliceVal.Kind() != reflect.Slice {
		return fmt.Errorf("container must point to a slice")
	}

	elemType := sliceVal.Type().Elem()
	isPtrElem := elemType.Kind() == reflect.Ptr

	var targetType reflect.Type
	if isPtrElem {
		if elemType.Elem().Kind() != reflect.Struct {
			return fmt.Errorf("slice element must be pointer to a struct")
		}
		targetType = elemType.Elem()
	} else {
		if elemType.Kind() != reflect.Struct {
			return fmt.Errorf("slice element must be a struct")
		}
		targetType = elemType
	}

	for _, record := range rs {
		// Create pointer to the target struct
		elemPtr := reflect.New(targetType)
		if err := record.Fill(elemPtr.Interface()); err != nil {
			return err
		}
		if isPtrElem {
			// Append as pointer
			sliceVal.Set(reflect.Append(sliceVal, elemPtr))
		} else {
			// Append as value
			sliceVal.Set(reflect.Append(sliceVal, elemPtr.Elem()))
		}
	}
	return nil
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

// unmarshalToRecordUnion parses an HTTP response body into one of the supported record types:
// - Record: a map representing a single JSON object (empty Record{} for empty responses or 204 No Content).
// - RecordSet: a slice of Records representing a JSON array.
//
// It inspects the first non-whitespace character of the response body to determine whether
// to unmarshal it into a Record or RecordSet. If the JSON format is unsupported (i.e., not an object or array),
// an error is returned.
func unmarshalToRecordUnion(response *http.Response) (Renderable, error) {
	defer response.Body.Close()

	// Handle empty response
	if response.ContentLength == 0 || response.StatusCode == http.StatusNoContent {
		return Record{}, nil
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	// Check first non-whitespace character
	trimmed := bytes.TrimSpace(body)
	if len(trimmed) == 0 {
		return Record{}, nil
	}
	switch trimmed[0] {
	case '{': // JSON object
		var rec Record
		if err := json.Unmarshal(body, &rec); err != nil {
			return nil, err
		}
		return rec, nil
	case '[': // JSON array
		// First try to unmarshal as RecordSet (array of objects)
		var recSet RecordSet
		if err := json.Unmarshal(body, &recSet); err == nil {
			return recSet, nil
		}
		// If that fails, it might be an array of any type, convert each to Record
		var anySlice []any
		if err := json.Unmarshal(body, &anySlice); err != nil {
			return nil, err
		}
		// Convert each item to a Record with customRawKey
		recordSet := make(RecordSet, len(anySlice))
		for i, item := range anySlice {
			recordSet[i] = Record{customRawKey: item}
		}
		return recordSet, nil
	case '"': // string
		return Record{customRawKey: body}, nil
	default:
		return nil, fmt.Errorf("unsupported JSON format: must be object or array")
	}
}

// applyCallbackForRecordUnion applies the provided callback function to a response if
// the response type matches the specified generic type T. It supports different types
// of Renderable responses (Record and RecordSet), and will only apply the
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

	default:
		return nil, fmt.Errorf("unsupported type %T for result", response)
	}
}

// typeMatch checks whether the dynamic type of given Renderable value
// matches the generic type T at runtime.
//
// It is typically used to determine if a response object corresponds to
// a specific expected data type (e.g., RecordSet or Record).
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

// setResourceKey sets resource type key for tabular formatting (only if not already set).
func setResourceKey(result Renderable, resourceType string) error {
	switch v := result.(type) {
	case Record:
		if _, ok := v[ResourceTypeKey]; !ok && len(v) > 0 {
			v[ResourceTypeKey] = resourceType
		}
		return nil
	case RecordSet:
		for _, rec := range v {
			if _, ok := rec[ResourceTypeKey]; !ok && len(rec) > 0 {
				rec[ResourceTypeKey] = resourceType
			}
		}
		return nil
	default:
		return fmt.Errorf("unsupported type")
	}
}

// ModelToRecord converts any typed model struct to a Record with @resourceType
// This is a helper function for typed resources to convert their models to Records
func ModelToRecord(model any) Record {
	// Marshal the model struct
	jsonBytes, err := json.Marshal(model)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal struct: %v", err))
	}

	record := make(Record)
	if err := json.Unmarshal(jsonBytes, &record); err != nil {
		panic(fmt.Sprintf("failed to unmarshal to record: %v", err))
	}

	// Get the type name and remove "Model" suffix
	modelType := reflect.TypeOf(model)
	// If it's a pointer, get the element type
	if modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
	}

	resourceType := modelType.Name()

	// Add resource type
	record[ResourceTypeKey] = resourceType

	return record
}

//  ######################################################
//              ASYNC RESULT
//  ######################################################

// AsyncResult represents the result of an asynchronous task.
// It contains the task's ID and necessary context for waiting on the task to complete.
type AsyncResult struct {
	TaskId int64
	Rest   VastRest
	ctx    context.Context
}

// NewAsyncResult creates a new AsyncResult from a task ID and REST client.
//
// This constructor is used to create an AsyncResult when you already have a task ID.
// The context is stored for potential future use with waiting operations.
//
// Parameters:
//   - ctx: The context associated with the task operation
//   - taskId: The ID of the asynchronous task
//   - rest: The REST client that can be used to query task status
//
// Returns:
//   - *AsyncResult: A new AsyncResult instance
func NewAsyncResult(ctx context.Context, taskId int64, rest VastRest) *AsyncResult {
	return &AsyncResult{
		ctx:    ctx,
		TaskId: taskId,
		Rest:   rest,
	}
}

// MaybeAsyncResultFromRecord attempts to extract an async task ID from a record and create an AsyncResult.
//
// This function handles two common patterns in VAST API responses:
//  1. Direct task response: The record itself has a ResourceTypeKey and represents the task
//  2. Nested task response: The record has an "async_task" field containing the task information
//
// If the record doesn't contain any task information, or if the task ID cannot be extracted,
// this function returns nil.
//
// Parameters:
//   - ctx: The context to associate with the async result
//   - record: The record that may contain async task information
//   - rest: The REST client for task operations
//
// Returns:
//   - *AsyncResult: An AsyncResult if task information was found, nil otherwise
func MaybeAsyncResultFromRecord(ctx context.Context, record Record, rest VastRest) *AsyncResult {
	var (
		taskId      int64
		asyncResult *AsyncResult
	)

	if record.Empty() {
		return nil
	}

	// Check if the record itself is a task (has ResourceTypeKey)
	if _, ok := record[ResourceTypeKey]; ok {
		// Only call RecordID if "id" field exists to avoid panic
		if _, hasId := record["id"]; hasId {
			taskId = record.RecordID()
		}
	} else {
		// Check for nested async_task field
		if asyncTask, ok := record["async_task"]; ok {
			var m map[string]any
			if m, ok = asyncTask.(map[string]any); ok {
				if _, hasId := m["id"]; hasId {
					taskId = ToRecord(m).RecordID()
				}
			}
		}
	}

	if taskId != 0 {
		asyncResult = NewAsyncResult(ctx, taskId, rest)
	}

	return asyncResult

}
