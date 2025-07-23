package vast_client

import (
	"errors"
	"testing"
)

func TestApiError_Error(t *testing.T) {
	tests := []struct {
		name     string
		apiError ApiError
		want     string
	}{
		{
			name: "complete API error",
			apiError: ApiError{
				Method:     "GET",
				URL:        "https://test.com/api/users",
				StatusCode: 404,
				Body:       "Not Found",
			},
			want: "GET request to https://test.com/api/users returned status code 404 — response body: Not Found",
		},
		{
			name: "zero status code",
			apiError: ApiError{
				Method:     "GET",
				URL:        "https://test.com/api/users",
				StatusCode: 0,
				Body:       "Connection failed",
			},
			want: "response body: Connection failed",
		},
		{
			name: "empty fields",
			apiError: ApiError{
				Method:     "",
				URL:        "",
				StatusCode: 500,
				Body:       "",
			},
			want: " request to  returned status code 500 — response body: ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.apiError.Error()
			if got != tt.want {
				t.Errorf("ApiError.Error() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsApiError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "is API error",
			err: &ApiError{
				Method:     "GET",
				URL:        "https://test.com",
				StatusCode: 404,
				Body:       "Not Found",
			},
			want: true,
		},
		{
			name: "is not API error - standard error",
			err:  errors.New("standard error"),
			want: false,
		},
		{
			name: "is not API error - nil",
			err:  nil,
			want: false,
		},
		{
			name: "is not API error - other custom error",
			err: &NotFoundError{
				Resource: "users",
				Query:    "name=test",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsApiError(tt.err)
			if got != tt.want {
				t.Errorf("IsApiError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIgnoreStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		codes      []int
		wantErrNil bool
	}{
		{
			name: "ignore matching status code",
			err: &ApiError{
				Method:     "DELETE",
				URL:        "https://test.com",
				StatusCode: 404,
				Body:       "Not Found",
			},
			codes:      []int{404, 409},
			wantErrNil: true,
		},
		{
			name: "don't ignore non-matching status code",
			err: &ApiError{
				Method:     "GET",
				URL:        "https://test.com",
				StatusCode: 500,
				Body:       "Internal Server Error",
			},
			codes:      []int{404, 409},
			wantErrNil: false,
		},
		{
			name: "ignore multiple codes - match first",
			err: &ApiError{
				Method:     "POST",
				URL:        "https://test.com",
				StatusCode: 409,
				Body:       "Conflict",
			},
			codes:      []int{404, 409},
			wantErrNil: true,
		},
		{
			name:       "non-API error unchanged",
			err:        errors.New("standard error"),
			codes:      []int{404},
			wantErrNil: false,
		},
		{
			name:       "nil error unchanged",
			err:        nil,
			codes:      []int{404},
			wantErrNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IgnoreStatusCodes(tt.err, tt.codes...)
			if (got == nil) != tt.wantErrNil {
				t.Errorf("IgnoreStatusCodes() error = %v, wantErrNil %v", got, tt.wantErrNil)
			}
		})
	}
}

func TestExpectStatusCodes(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		codes []int
		want  bool
	}{
		{
			name: "status code matches",
			err: &ApiError{
				Method:     "GET",
				URL:        "https://test.com",
				StatusCode: 404,
				Body:       "Not Found",
			},
			codes: []int{404, 409},
			want:  true,
		},
		{
			name: "status code doesn't match",
			err: &ApiError{
				Method:     "GET",
				URL:        "https://test.com",
				StatusCode: 500,
				Body:       "Internal Server Error",
			},
			codes: []int{404, 409},
			want:  false,
		},
		{
			name:  "non-API error",
			err:   errors.New("standard error"),
			codes: []int{404},
			want:  false,
		},
		{
			name:  "nil error",
			err:   nil,
			codes: []int{404},
			want:  false,
		},
		{
			name: "empty codes list",
			err: &ApiError{
				StatusCode: 404,
			},
			codes: []int{},
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpectStatusCodes(tt.err, tt.codes...)
			if got != tt.want {
				t.Errorf("ExpectStatusCodes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNotFoundError_Error(t *testing.T) {
	err := &NotFoundError{
		Resource: "users",
		Query:    "name=test&active=true",
	}

	want := "resource 'users' not found for params 'name=test&active=true'"
	got := err.Error()

	if got != want {
		t.Errorf("NotFoundError.Error() = %v, want %v", got, want)
	}
}

func TestTooManyRecordsError_Error(t *testing.T) {
	err := &TooManyRecordsError{
		ResourcePath: "/api/v5/users/",
		Params:       Params{"name": "test", "limit": 1},
	}

	want := "too many records found for resource '/api/v5/users/' with params 'map[limit:1 name:test]'"
	got := err.Error()

	if got != want {
		t.Errorf("TooManyRecordsError.Error() = %v, want %v", got, want)
	}
}

func TestIsNotFoundErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "is NotFoundError",
			err: &NotFoundError{
				Resource: "users",
				Query:    "name=test",
			},
			want: true,
		},
		{
			name: "is not NotFoundError - API error",
			err: &ApiError{
				StatusCode: 404,
			},
			want: false,
		},
		{
			name: "is not NotFoundError - standard error",
			err:  errors.New("standard error"),
			want: false,
		},
		{
			name: "is not NotFoundError - nil",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsNotFoundErr(tt.err)
			if got != tt.want {
				t.Errorf("IsNotFoundErr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIgnoreNotFound(t *testing.T) {
	testRecord := Record{"id": 123, "name": "test"}

	tests := []struct {
		name       string
		val        Record
		err        error
		wantVal    Record
		wantErrNil bool
	}{
		{
			name:       "ignore NotFoundError",
			val:        testRecord,
			err:        &NotFoundError{Resource: "users", Query: "name=test"},
			wantVal:    testRecord,
			wantErrNil: true,
		},
		{
			name:       "don't ignore other errors",
			val:        testRecord,
			err:        errors.New("other error"),
			wantVal:    testRecord,
			wantErrNil: false,
		},
		{
			name:       "no error",
			val:        testRecord,
			err:        nil,
			wantVal:    testRecord,
			wantErrNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVal, gotErr := IgnoreNotFound(tt.val, tt.err)
			if (gotErr == nil) != tt.wantErrNil {
				t.Errorf("IgnoreNotFound() error = %v, wantErrNil %v", gotErr, tt.wantErrNil)
			}
			if len(gotVal) != len(tt.wantVal) {
				t.Errorf("IgnoreNotFound() val = %v, want %v", gotVal, tt.wantVal)
			}
		})
	}
}

func TestIsTooManyRecordsErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "is TooManyRecordsError",
			err: &TooManyRecordsError{
				ResourcePath: "/api/users/",
				Params:       Params{"name": "test"},
			},
			want: true,
		},
		{
			name: "is not TooManyRecordsError - NotFoundError",
			err: &NotFoundError{
				Resource: "users",
				Query:    "name=test",
			},
			want: false,
		},
		{
			name: "is not TooManyRecordsError - standard error",
			err:  errors.New("standard error"),
			want: false,
		},
		{
			name: "is not TooManyRecordsError - nil",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isTooManyRecordsErr(tt.err)
			if got != tt.want {
				t.Errorf("isTooManyRecordsErr() = %v, want %v", got, tt.want)
			}
		})
	}
}
