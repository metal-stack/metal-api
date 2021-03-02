package metal

import (
	"testing"

	"github.com/pkg/errors"
)

func TestNotFound(t *testing.T) {
	type args struct {
		format string
		args   []interface{}
	}

	theargs := args{
		format: "SomeFormat",
	}

	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		// Test Data
		{
			name:    "TestNotFound 1",
			args:    theargs,
			wantErr: true,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if err := NotFound(tt.args.format, tt.args.args...); (err != nil) != tt.wantErr {
				t.Errorf("NotFound() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIsNotFound(t *testing.T) {
	type args struct {
		e error
	}

	theargs := args{
		e: errors.New("Some other Error"),
	}

	theargs2 := args{
		e: errNotFound,
	}

	tests := []struct {
		name string
		args args
		want bool
	}{
		// Test Data Array:
		{
			name: "Test 1",
			args: theargs,
			want: false,
		},
		{
			name: "Test 2",
			args: theargs2,
			want: true,
		},
	}
	for i := range tests {
		tt := tests[i]
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.args.e); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}
