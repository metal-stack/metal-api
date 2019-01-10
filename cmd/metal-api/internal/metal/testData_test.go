package metal

import (
	"testing"

	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func TestPrepareTests(t *testing.T) {
	PrepareTests()
	tests := []struct {
		name string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			PrepareTests()
		})
	}
}

func TestInitMockDBData(t *testing.T) {
	// mock the DB
	mock := r.NewMock()
	InitMockDBData(mock)

	type args struct {
		mock *r.Mock
	}
	tests := []struct {
		name string
		args args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			InitMockDBData(tt.args.mock)
		})
	}
}
