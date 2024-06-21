package metal

import (
	"errors"
	"testing"
)

func TestNotFound(t *testing.T) {
	tests := []struct {
		name    string
		format  string
		args    []interface{}
		wantErr bool
	}{
		{
			name:    "TestNotFound 1",
			format:  "SomeFormat",
			wantErr: true,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := NotFound(tt.format, tt.args...); (err != nil) != tt.wantErr {
				t.Errorf("NotFound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Test 1",
			err:  errors.New("Some other Error"),
			want: false,
		},
		{
			name: "Test 2",
			err:  errNotFound,
			want: true,
		},
		{
			name: "Test 3",
			err:  nil,
			want: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.err); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsConflict(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Test 1",
			err:  errors.New("Some other Error"),
			want: false,
		},
		{
			name: "Test 2",
			err:  errConflict,
			want: true,
		},
		{
			name: "Test 3",
			err:  nil,
			want: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConflict(tt.err); got != tt.want {
				t.Errorf("IsConflict() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsInternal(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "Test 1",
			err:  errors.New("Some other Error"),
			want: false,
		},
		{
			name: "Test 2",
			err:  errInternal,
			want: true,
		},
		{
			name: "Test 3",
			err:  nil,
			want: false,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := IsInternal(tt.err); got != tt.want {
				t.Errorf("IsInternal() = %v, want %v", got, tt.want)
			}
		})
	}
}
