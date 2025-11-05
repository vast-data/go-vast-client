package core

import (
	"errors"
	"testing"
)

func TestNotFoundError_Error(t *testing.T) {
	err := &NotFoundError{
		Resource: "User",
		Query:    "name=testuser",
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Error("NotFoundError.Error() returned empty string")
	}
}

func TestIsNotFoundErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "NotFoundError",
			err:  &NotFoundError{Resource: "User"},
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundErr(tt.err); got != tt.want {
				t.Errorf("IsNotFoundErr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIgnoreNotFound(t *testing.T) {
	tests := []struct {
		name      string
		val       Record
		err       error
		wantErr   bool
		wantValue bool
	}{
		{
			name:      "NotFoundError should be ignored",
			val:       Record{"test": "value"},
			err:       &NotFoundError{Resource: "User"},
			wantErr:   false,
			wantValue: true,
		},
		{
			name:      "regular error should not be ignored",
			val:       Record{},
			err:       errors.New("some error"),
			wantErr:   true,
			wantValue: false,
		},
		{
			name:      "nil error should return value",
			val:       Record{"test": "value"},
			err:       nil,
			wantErr:   false,
			wantValue: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, err := IgnoreNotFound(tt.val, tt.err)
			if (err != nil) != tt.wantErr {
				t.Errorf("IgnoreNotFound() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantValue && val == nil {
				t.Error("IgnoreNotFound() returned nil value when value expected")
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
			name: "TooManyRecordsError",
			err:  &TooManyRecordsError{ResourcePath: "/users"},
			want: true,
		},
		{
			name: "regular error",
			err:  errors.New("some error"),
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTooManyRecordsErr(tt.err); got != tt.want {
				t.Errorf("IsTooManyRecordsErr() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTooManyRecordsError_Error(t *testing.T) {
	err := &TooManyRecordsError{
		ResourcePath: "/users",
		Params:       Params{"name": "test"},
	}

	errMsg := err.Error()
	if errMsg == "" {
		t.Error("TooManyRecordsError.Error() returned empty string")
	}
}
