package datastore

import (
	"reflect"
	"testing"
)

func TestMigrations_Between(t *testing.T) {
	type args struct {
		current int
		target  *int
	}
	tests := []struct {
		name    string
		ms      Migrations
		args    args
		want    Migrations
		wantErr bool
	}{
		{
			name: "no migrations is fine",
			ms:   []Migration{},
			args: args{
				current: 0,
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "get all migrations from 0, sorted",
			ms: []Migration{
				{
					Name:    "migration 4",
					Version: 4,
				},
				{
					Name:    "migration 2",
					Version: 2,
				},
				{
					Name:    "migration 1",
					Version: 1,
				},
			},
			args: args{
				current: 0,
			},
			want: []Migration{
				{
					Name:    "migration 1",
					Version: 1,
				},
				{
					Name:    "migration 2",
					Version: 2,
				},
				{
					Name:    "migration 4",
					Version: 4,
				},
			},
			wantErr: false,
		},
		{
			name: "get all migrations from 1, sorted",
			ms: []Migration{
				{
					Name:    "migration 4",
					Version: 4,
				},
				{
					Name:    "migration 2",
					Version: 2,
				},
				{
					Name:    "migration 1",
					Version: 1,
				},
			},
			args: args{
				current: 1,
			},
			want: []Migration{
				{
					Name:    "migration 2",
					Version: 2,
				},
				{
					Name:    "migration 4",
					Version: 4,
				},
			},
			wantErr: false,
		},
		{
			name: "get migrations up to target version, sorted",
			ms: []Migration{
				{
					Name:    "migration 4",
					Version: 4,
				},
				{
					Name:    "migration 2",
					Version: 2,
				},
				{
					Name:    "migration 1",
					Version: 1,
				},
			},
			args: args{
				current: 0,
				target:  intPtr(2),
			},
			want: []Migration{
				{
					Name:    "migration 1",
					Version: 1,
				},
				{
					Name:    "migration 2",
					Version: 2,
				},
			},
			wantErr: false,
		},
		{
			name: "error on unknown target version",
			ms: []Migration{
				{
					Name:    "migration 4",
					Version: 4,
				},
				{
					Name:    "migration 2",
					Version: 2,
				},
				{
					Name:    "migration 1",
					Version: 1,
				},
			},
			args: args{
				current: 0,
				target:  intPtr(3),
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.ms.Between(tt.args.current, tt.args.target)
			if (err != nil) != tt.wantErr {
				t.Errorf("Migrations.Between() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Migrations.Between() = %v, want %v", got, tt.want)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
