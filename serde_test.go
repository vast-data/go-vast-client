package vast_client

import (
	"io"
	"strings"
	"testing"
)

func TestParams_ToMultipartFormData_FileData(t *testing.T) {
	params := Params{
		"file_field": FileData{
			Filename: "test.txt",
			Content:  []byte("test file content"),
		},
		"text_field": "text value",
	}

	result, err := params.ToMultipartFormData()
	if err != nil {
		t.Fatalf("ToMultipartFormData failed: %v", err)
	}

	// Check content type is multipart/form-data
	if !strings.HasPrefix(result.ContentType, "multipart/form-data") {
		t.Errorf("Expected content type to start with multipart/form-data, got %s", result.ContentType)
	}

	// Check boundary is present
	if !strings.Contains(result.ContentType, "boundary=") {
		t.Errorf("Expected boundary in content type, got %s", result.ContentType)
	}

	// Read the body and verify content
	bodyBytes, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	bodyStr := string(bodyBytes)

	// Check file field is present
	if !strings.Contains(bodyStr, "test file content") {
		t.Errorf("Expected file content in body")
	}

	// Check text field is present
	if !strings.Contains(bodyStr, "text value") {
		t.Errorf("Expected text field in body")
	}

	// Check filename is present
	if !strings.Contains(bodyStr, "test.txt") {
		t.Errorf("Expected filename in body")
	}
}

func TestParams_ToMultipartFormData_ByteData(t *testing.T) {
	params := Params{
		"byte_field": []byte("raw byte data"),
		"text_field": "text value",
	}

	result, err := params.ToMultipartFormData()
	if err != nil {
		t.Fatalf("ToMultipartFormData failed: %v", err)
	}

	// Read the body and verify content
	bodyBytes, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	bodyStr := string(bodyBytes)

	// Check byte data is present
	if !strings.Contains(bodyStr, "raw byte data") {
		t.Errorf("Expected byte data in body")
	}

	// Check text field is present
	if !strings.Contains(bodyStr, "text value") {
		t.Errorf("Expected text field in body")
	}
}

func TestParams_ToMultipartFormData_TextFieldsOnly(t *testing.T) {
	params := Params{
		"field1": "value1",
		"field2": "value2",
		"field3": 123,
	}

	result, err := params.ToMultipartFormData()
	if err != nil {
		t.Fatalf("ToMultipartFormData failed: %v", err)
	}

	// Read the body and verify content
	bodyBytes, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	bodyStr := string(bodyBytes)

	// Check all fields are present
	if !strings.Contains(bodyStr, "value1") {
		t.Errorf("Expected field1 value in body")
	}

	if !strings.Contains(bodyStr, "value2") {
		t.Errorf("Expected field2 value in body")
	}

	if !strings.Contains(bodyStr, "123") {
		t.Errorf("Expected field3 value in body")
	}
}

func TestParams_ToMultipartFormData_EmptyParams(t *testing.T) {
	params := Params{}

	result, err := params.ToMultipartFormData()
	if err != nil {
		t.Fatalf("ToMultipartFormData failed: %v", err)
	}

	// Should still have valid multipart content type
	if !strings.HasPrefix(result.ContentType, "multipart/form-data") {
		t.Errorf("Expected content type to start with multipart/form-data, got %s", result.ContentType)
	}

	// Body should be valid but minimal
	bodyBytes, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	// Should have boundary markers
	bodyStr := string(bodyBytes)
	if !strings.Contains(bodyStr, "--") {
		t.Errorf("Expected boundary markers in empty multipart body")
	}
}

func TestParams_ToMultipartFormData_MixedContent(t *testing.T) {
	params := Params{
		"file1": FileData{
			Filename: "document.pdf",
			Content:  []byte("PDF content here"),
		},
		"raw_bytes": []byte("raw binary data"),
		"text1":     "simple text",
		"number":    42,
	}

	result, err := params.ToMultipartFormData()
	if err != nil {
		t.Fatalf("ToMultipartFormData failed: %v", err)
	}

	// Read the body and verify all content is present
	bodyBytes, err := io.ReadAll(result.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	bodyStr := string(bodyBytes)

	// Check all content types
	if !strings.Contains(bodyStr, "PDF content here") {
		t.Errorf("Expected PDF content in body")
	}

	if !strings.Contains(bodyStr, "document.pdf") {
		t.Errorf("Expected PDF filename in body")
	}

	if !strings.Contains(bodyStr, "raw binary data") {
		t.Errorf("Expected raw bytes in body")
	}

	if !strings.Contains(bodyStr, "simple text") {
		t.Errorf("Expected text1 in body")
	}

	if !strings.Contains(bodyStr, "42") {
		t.Errorf("Expected number field in body")
	}
}

func TestFileData_Struct(t *testing.T) {
	// Test FileData struct creation and access
	fileData := FileData{
		Filename: "example.txt",
		Content:  []byte("example content"),
	}

	if fileData.Filename != "example.txt" {
		t.Errorf("Expected filename example.txt, got %s", fileData.Filename)
	}

	if string(fileData.Content) != "example content" {
		t.Errorf("Expected content 'example content', got %s", string(fileData.Content))
	}
}

func TestMultipartFormData_Struct(t *testing.T) {
	// Test MultipartFormData struct
	body := strings.NewReader("test body")
	contentType := "multipart/form-data; boundary=test123"

	multipartData := &MultipartFormData{
		Body:        body,
		ContentType: contentType,
	}

	if multipartData.ContentType != contentType {
		t.Errorf("Expected content type %s, got %s", contentType, multipartData.ContentType)
	}

	// Read body to verify
	bodyBytes, err := io.ReadAll(multipartData.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	if string(bodyBytes) != "test body" {
		t.Errorf("Expected body 'test body', got %s", string(bodyBytes))
	}
}
