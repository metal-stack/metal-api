package datastore

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"

	"github.com/stretchr/testify/assert"

	r "gopkg.in/rethinkdb/rethinkdb-go.v6"
)

func Test_makeRange(t *testing.T) {
	type args struct {
		min uint32
		max uint32
	}
	tests := []struct {
		name string
		args args
		want []integer
	}{
		{
			name: "verify minimum range 1-1",
			args: args{min: 1, max: 1},
			want: []integer{{"vrf", 1}},
		},
		{
			name: "verify range 1-5",
			args: args{min: 1, max: 5},
			want: []integer{{"vrf", 1}, {"vrf", 2}, {"vrf", 3}, {"vrf", 4}, {"vrf", 5}},
		},
		{
			name: "verify empty range on max less than min",
			args: args{min: 1, max: 0},
			want: []integer{},
		},
		{
			name: "verify zero result",
			args: args{min: 0, max: 0},
			want: []integer{{"vrf", 0}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := makeRange("vrf", tt.args.min, tt.args.max); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("makeRange() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_ReleaseUniqueInteger(t *testing.T) {

	tests := []struct {
		name         string
		value        uint32
		err          error
		requiresMock bool
	}{
		{
			name:         "verify deletion succeeds",
			value:        10000,
			err:          nil,
			requiresMock: true,
		},
		{
			name:         "verify deletion fails",
			value:        10000,
			err:          errors.New("any error message indicating insert failed"),
			requiresMock: true,
		},
		{
			name:         "verify validation of input fails",
			value:        524288,
			err:          fmt.Errorf("value '524288' is outside of the allowed range '%d - %d'", IntegerPoolRangeMin, IntegerPoolRangeMax),
			requiresMock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs, mock := InitMockDB()
			ip := rs.IntegerPools["vrf"]

			if tt.requiresMock {
				if tt.err != nil {
					mock.On(r.DB("mockdb").Table("integerpool").Insert(integer{Name: "vrf", ID: tt.value}, r.InsertOpts{Conflict: "replace"})).Return(nil, tt.err)
				} else {
					mock.On(r.DB("mockdb").Table("integerpool").Insert(integer{Name: "vrf", ID: tt.value}, r.InsertOpts{Conflict: "replace"})).Return(r.
						WriteResponse{Changes: []r.ChangeResponse{{OldValue: map[string]interface{}{"id": float64(
						tt.value)}}}}, tt.err)
				}
			}

			got := ip.ReleaseUniqueInteger(tt.value)
			if tt.err != nil {
				assert.EqualError(t, got, tt.err.Error())
			} else {
				assert.NoError(t, got)
			}

			if tt.requiresMock {
				mock.AssertExpectations(t)
			}
		})
	}
}

func TestRethinkStore_AcquireRandomUniqueInteger(t *testing.T) {
	rs, mock := InitMockDB()
	ip := rs.IntegerPools["vrf"]

	changes := []r.ChangeResponse{{OldValue: map[string]interface{}{"name": "vrf", "id": float64(IntegerPoolRangeMin)}}}
	mock.On(r.DB("mockdb").Table("integerpool").Get(map[string]interface{}{"name": "vrf"}).Limit(1).Delete(r.
		DeleteOpts{ReturnChanges: true})).Return(r.WriteResponse{Changes: changes}, nil)

	got, err := ip.AcquireRandomUniqueInteger()
	assert.NoError(t, err)
	assert.EqualValues(t, IntegerPoolRangeMin, got)

	mock.AssertExpectations(t)
}

func TestRethinkStore_AcquireUniqueInteger(t *testing.T) {
	tests := []struct {
		name         string
		value        uint32
		err          error
		requiresMock bool
	}{
		{
			name:         "verify rethinkdb term is called as expected",
			value:        10000,
			err:          nil,
			requiresMock: true,
		},
		{
			name:         "verify validation of input fails",
			value:        524288,
			err:          fmt.Errorf("value '524288' is outside of the allowed range '%d - %d'", IntegerPoolRangeMin, IntegerPoolRangeMax),
			requiresMock: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs, mock := InitMockDB()
			ip := rs.IntegerPools["vrf"]

			if tt.requiresMock {
				changes := []r.ChangeResponse{{OldValue: map[string]interface{}{"name": "vrf", "id": float64(
					tt.value)}}}
				mock.On(r.DB("mockdb").Table("integerpool").Get(map[string]interface{}{"id": tt.value, "name": "vrf"}).Delete(r.
					DeleteOpts{ReturnChanges: true})).Return(r.WriteResponse{Changes: changes}, tt.err)
			}

			got, err := ip.AcquireUniqueInteger(tt.value)
			if tt.err != nil {
				assert.EqualError(t, err, tt.err.Error())
			} else {
				assert.NoError(t, err)
				assert.EqualValues(t, tt.value, got)
			}

			if tt.requiresMock {
				mock.AssertExpectations(t)
			}
		})
	}
}

func TestRethinkStore_genericAcquire(t *testing.T) {
	tests := []struct {
		name              string
		value             uint32
		runWriteErr       error
		expectedErr       error
		requiresMock      bool
		requiresCountMock bool
		tableChanges      bool
	}{
		{
			name:              "verify acquire succeeds",
			value:             10000,
			runWriteErr:       nil,
			expectedErr:       nil,
			requiresMock:      true,
			requiresCountMock: false,
			tableChanges:      true,
		},
		{
			name:              "verify fails for unable to execute deletion",
			value:             10000,
			runWriteErr:       errors.New("any error message indicating delete failed"),
			expectedErr:       errors.New("any error message indicating delete failed"),
			requiresMock:      true,
			requiresCountMock: false,
			tableChanges:      true,
		},
		{
			name:              "verify fails for unavailability error",
			value:             10000,
			runWriteErr:       nil,
			expectedErr:       metal.Internal(errors.New("acquisition of a value failed for exhausted pool"), ""),
			requiresMock:      true,
			requiresCountMock: true,
			tableChanges:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs, mock := InitMockDB()
			ip := rs.IntegerPools["vrf"]

			term := rs.integerTable().Get(tt.value)
			if tt.requiresMock {
				var changes []r.ChangeResponse
				if tt.tableChanges {
					changes = []r.ChangeResponse{{OldValue: map[string]interface{}{"id": float64(
						tt.value)}}}
				}
				mock.On(term.Delete(r.DeleteOpts{ReturnChanges: true})).Return(r.WriteResponse{Changes: changes},
					tt.runWriteErr)
				if tt.requiresCountMock {
					mock.On(rs.integerTable().Count()).Return(int64(0), nil)
				}
			}

			got, err := ip.genericAcquire(&term)
			if tt.expectedErr != nil {
				assert.EqualError(t, err, tt.expectedErr.Error())
			} else {
				assert.NoError(t, err)
				assert.EqualValues(t, tt.value, got)
			}

			if tt.requiresMock {
				mock.AssertExpectations(t)
			}
		})
	}
}
