package common

import (
	"reflect"
	"testing"

	vast_client "github.com/vast-data/go-vast-client"
)

func TestConvertServerParamsToVastParams(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  vast_client.Params
		expectErr bool
		errMsg    string
	}{
		// Basic cases
		{
			name:     "empty string",
			input:    "",
			expected: vast_client.Params{},
		},
		{
			name:  "single regular parameter",
			input: "path__contains=test",
			expected: vast_client.Params{
				"path__contains": "test",
			},
		},
		{
			name:  "single regular parameter with spaces",
			input: " path__contains=test ",
			expected: vast_client.Params{
				"path__contains": "test",
			},
		},

		// __in parameters with integers
		{
			name:  "__in parameter with integers",
			input: "id__in=1,2,3,4,5,6,7",
			expected: vast_client.Params{
				"id__in": []int64{1, 2, 3, 4, 5, 6, 7},
			},
		},
		{
			name:  "__in parameter with single integer",
			input: "id__in=42",
			expected: vast_client.Params{
				"id__in": []int64{42},
			},
		},
		{
			name:  "__in parameter with integers and spaces",
			input: "id__in=1,2,3",
			expected: vast_client.Params{
				"id__in": []int64{1, 2, 3},
			},
		},

		// __in parameters with strings
		{
			name:  "__in parameter with strings",
			input: "name__in=alice,bob,charlie",
			expected: vast_client.Params{
				"name__in": []string{"alice", "bob", "charlie"},
			},
		},
		{
			name:  "__in parameter with mixed integers and strings",
			input: "mixed__in=1,alice,2,bob",
			expected: vast_client.Params{
				"mixed__in": []string{"1", "alice", "2", "bob"},
			},
		},

		// Empty __in parameters
		{
			name:  "__in parameter empty",
			input: "id__in=",
			expected: vast_client.Params{
				"id__in": []string{},
			},
		},
		{
			name:  "__in parameter with trailing comma",
			input: "id__in=1,2,3,",
			expected: vast_client.Params{
				"id__in": []int64{1, 2, 3},
			},
		},
		{
			name:  "__in parameter with leading comma",
			input: "id__in=,1,2,3",
			expected: vast_client.Params{
				"id__in": []int64{1, 2, 3},
			},
		},

		// Multiple parameters - space separated
		{
			name:  "multiple parameters space separated",
			input: "path__contains=test id__in=1,2,3",
			expected: vast_client.Params{
				"path__contains": "test",
				"id__in":         []int64{1, 2, 3},
			},
		},
		{
			name:  "multiple parameters space separated complex",
			input: "path__contains=test name__in=alice,bob path__startswith=prefix",
			expected: vast_client.Params{
				"path__contains":   "test",
				"name__in":         []string{"alice", "bob"},
				"path__startswith": "prefix",
			},
		},

		// Multiple parameters - comma separated (tricky case)
		{
			name:  "multiple regular parameters comma separated",
			input: "path__contains=test,path__startswith=prefix",
			expected: vast_client.Params{
				"path__contains":   "test",
				"path__startswith": "prefix",
			},
		},

		// Complex mixed cases
		{
			name:  "complex mixed parameters",
			input: "path__contains=test id__in=1,2,3 name__in=alice,bob path__endswith=suffix",
			expected: vast_client.Params{
				"path__contains": "test",
				"id__in":         []int64{1, 2, 3},
				"name__in":       []string{"alice", "bob"},
				"path__endswith": "suffix",
			},
		},

		// Error cases
		{
			name:      "invalid format no equals",
			input:     "invalid_format",
			expectErr: true,
			errMsg:    "invalid parameter format: invalid_format (expected key=value)",
		},
		{
			name:      "empty key",
			input:     "=value",
			expectErr: true,
			errMsg:    "empty key in parameter: =value",
		},
		{
			name:      "only equals",
			input:     "=",
			expectErr: true,
			errMsg:    "empty key in parameter: =",
		},
		{
			name:  "multiple equals",
			input: "key=value=extra",
			expected: vast_client.Params{
				"key": "value=extra", // SplitN(2) should handle this correctly
			},
		},

		// Edge cases with whitespace
		{
			name:     "parameter with only spaces",
			input:    "   ",
			expected: vast_client.Params{},
		},
		{
			name:  "mixed with empty parameters",
			input: "path__contains=test  id__in=1,2,3",
			expected: vast_client.Params{
				"path__contains": "test",
				"id__in":         []int64{1, 2, 3},
			},
		},

		// Bracket notation tests
		{
			name:  "__in parameter with bracket notation integers",
			input: "uid__in=[1,2,3,4,5]",
			expected: vast_client.Params{
				"uid__in": []int64{1, 2, 3, 4, 5},
			},
		},
		{
			name:  "__in parameter with bracket notation strings",
			input: "name__in=[alice,bob,charlie]",
			expected: vast_client.Params{
				"name__in": []string{"alice", "bob", "charlie"},
			},
		},
		{
			name:  "__in parameter with bracket notation and spaces",
			input: "id__in=[ 1, 2, 3 ]",
			expected: vast_client.Params{
				"id__in": []int64{1, 2, 3},
			},
		},
		{
			name:  "__in parameter with empty brackets",
			input: "id__in=[]",
			expected: vast_client.Params{
				"id__in": []string{},
			},
		},
		{
			name:  "mixed parameters with bracket notation",
			input: "path__contains=test uid__in=[1,2,3] name__in=[alice,bob]",
			expected: vast_client.Params{
				"path__contains": "test",
				"uid__in":        []int64{1, 2, 3},
				"name__in":       []string{"alice", "bob"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertServerParamsToVastParams(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errMsg != "" && err.Error() != tt.errMsg {
					t.Errorf("expected error message %q, got %q", tt.errMsg, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestConvertServerParamsToVastParams_TypeChecking(t *testing.T) {
	// Test specific type checking scenarios
	tests := []struct {
		name      string
		input     string
		key       string
		checkType func(interface{}) bool
		typeName  string
	}{
		{
			name:  "integer slice type check",
			input: "id__in=1,2,3",
			key:   "id__in",
			checkType: func(v interface{}) bool {
				_, ok := v.([]int64)
				return ok
			},
			typeName: "[]int64",
		},
		{
			name:  "string slice type check",
			input: "name__in=alice,bob",
			key:   "name__in",
			checkType: func(v interface{}) bool {
				_, ok := v.([]string)
				return ok
			},
			typeName: "[]string",
		},
		{
			name:  "mixed becomes string slice",
			input: "mixed__in=1,alice,2",
			key:   "mixed__in",
			checkType: func(v interface{}) bool {
				_, ok := v.([]string)
				return ok
			},
			typeName: "[]string",
		},
		{
			name:  "regular parameter stays string",
			input: "path__contains=test",
			key:   "path__contains",
			checkType: func(v interface{}) bool {
				_, ok := v.(string)
				return ok
			},
			typeName: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertServerParamsToVastParams(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			value, exists := result[tt.key]
			if !exists {
				t.Fatalf("key %s not found in result", tt.key)
			}

			if !tt.checkType(value) {
				t.Errorf("expected type %s for key %s, got %T", tt.typeName, tt.key, value)
			}
		})
	}
}

func TestSplitServerParams(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
		{
			name:     "single parameter",
			input:    "key=value",
			expected: []string{"key=value"},
		},
		{
			name:     "comma separated",
			input:    "key1=value1,key2=value2",
			expected: []string{"key1=value1", "key2=value2"},
		},
		{
			name:     "space separated",
			input:    "key1=value1 key2=value2",
			expected: []string{"key1=value1", "key2=value2"},
		},
		{
			name:     "comma separated with spaces",
			input:    "key1=value1, key2=value2 , key3=value3",
			expected: []string{"key1=value1", "key2=value2", "key3=value3"},
		},
		{
			name:     "space separated with extra spaces",
			input:    "  key1=value1   key2=value2  ",
			expected: []string{"key1=value1", "key2=value2"},
		},
		{
			name:     "mixed with empty entries",
			input:    "key1=value1,,key2=value2",
			expected: []string{"key1=value1", "key2=value2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SplitServerParams(tt.input)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}
