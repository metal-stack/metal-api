package datastore

import (
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	r "gopkg.in/rethinkdb/rethinkdb-go.v5"
)

func TestRethinkStore_FindSite(t *testing.T) {

	// mock the DB
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("site").Get("1")).Return(site1, nil)
	mock.On(r.DB("mockdb").Table("site").Get("2")).Return(site2, nil)

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Site
		wantErr bool
	}{
		// Test Data Array:
		{
			name: "TestRethinkStore_FindSite Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &site1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_FindSite Test 2",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &site2,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.FindSite(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.FindSite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.FindSite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_ListSites(t *testing.T) {

	// mock the DBs
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("site")).Return([]metal.Site{
		site1, site2,
	}, nil)
	ds2, mock2 := initMockDB()
	mock2.On(r.DB("mockdb").Table("site")).Return([]metal.Site{
		site1,
	}, nil)

	tests := []struct {
		name    string
		rs      *RethinkStore
		want    []metal.Site
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_ListSites Test 1",
			rs:   ds,
			want: []metal.Site{
				site1, site2,
			},
			wantErr: false,
		},
		{
			name: "TestRethinkStore_ListSites Test 2",
			rs:   ds2,
			want: []metal.Site{
				site1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.ListSites()
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.ListSites() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.ListSites() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_CreateSite(t *testing.T) {

	// mock the DB
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("site").Insert(site1)).Return(emptyResult, nil)

	type args struct {
		site *metal.Site
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_CreateSite Test 1",
			rs:   ds,
			args: args{
				site: &site1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.CreateSite(tt.args.site); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.CreateSite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRethinkStore_DeleteSite(t *testing.T) {

	// mock the DBs
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("site").Get("1")).Return(site1, nil)
	mock.On(r.DB("mockdb").Table("site").Get("1").Delete()).Return(emptyResult, nil)
	mock.On(r.DB("mockdb").Table("site").Get("2")).Return(site2, nil)
	mock.On(r.DB("mockdb").Table("site").Get("2").Delete()).Return(emptyResult, nil)
	mock.On(r.DB("mockdb").Table("site").Get("3")).Return(emptyResult, nil)
	mock.On(r.DB("mockdb").Table("site").Get("3").Delete()).Return(emptyResult, r.ErrEmptyResult)

	type args struct {
		id string
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		want    *metal.Site
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_DeleteSite Test 1",
			rs:   ds,
			args: args{
				id: "1",
			},
			want:    &site1,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_DeleteSite Test 2",
			rs:   ds,
			args: args{
				id: "2",
			},
			want:    &site2,
			wantErr: false,
		},
		{
			name: "TestRethinkStore_DeleteSite Test 3",
			rs:   ds,
			args: args{
				id: "3",
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.rs.DeleteSite(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.DeleteSite() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RethinkStore.DeleteSite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRethinkStore_UpdateSite(t *testing.T) {

	// mock the DBs
	ds, mock := initMockDB()
	mock.On(r.DB("mockdb").Table("site").Get("1").Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(site1.Changed)), site2, r.Error("the Site was changed from another, please retry"))
	})).Return(emptyResult, nil)
	mock.On(r.DB("mockdb").Table("site").Get("2").Replace(func(row r.Term) r.Term {
		return r.Branch(row.Field("changed").Eq(r.Expr(site2.Changed)), site1, r.Error("the Site was changed from another, please retry"))
	})).Return(emptyResult, nil)

	type args struct {
		oldF *metal.Site
		newF *metal.Site
	}
	tests := []struct {
		name    string
		rs      *RethinkStore
		args    args
		wantErr bool
	}{
		// Test Data Array / Test Cases:
		{
			name: "TestRethinkStore_UpdateSite Test 1",
			rs:   ds,
			args: args{
				&site1, &site2,
			},
			wantErr: false,
		},
		{
			name: "TestRethinkStore_UpdateSite Test 2",
			rs:   ds,
			args: args{
				&site2, &site1,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.rs.UpdateSite(tt.args.oldF, tt.args.newF); (err != nil) != tt.wantErr {
				t.Errorf("RethinkStore.UpdateSite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
