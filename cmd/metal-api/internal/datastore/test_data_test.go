package datastore

import (
	"reflect"
	"testing"

	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func TestInitMockDB(t *testing.T) {
	tests := []struct {
		name  string
		want  *RethinkStore
		want1 *r.Mock
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := InitMockDB()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InitMockDB() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("InitMockDB() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
