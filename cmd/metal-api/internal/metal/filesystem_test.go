package metal

import (
	"reflect"
	"testing"
)

var (
	s1 = Size{
		Base: Base{ID: "c1-large-x86"},
	}
	s2 = Size{
		Base: Base{ID: "c1-xlarge-x86"},
	}
	s3 = Size{
		Base: Base{ID: "s3-large-x86"},
	}
	s4 = Size{
		Base: Base{ID: "s2-large-x86"},
	}

	i1 = Image{
		Base: Base{ID: "debian-10"},
	}
	i2 = Image{
		Base: Base{ID: "ubuntu-20.04"},
	}
	i3 = Image{
		Base: Base{ID: "firewall-2"},
	}
	i4 = Image{
		Base: Base{ID: "centos-7"},
	}
)

func TestFilesystemLayoutConstraint_Matches(t *testing.T) {
	type constraints struct {
		Sizes  map[string]bool
		Images []string
	}
	type args struct {
		size  Size
		image Image
	}
	tests := []struct {
		name string
		c    constraints
		args args
		want bool
	}{
		{
			name: "default layout",
			c: constraints{
				Sizes:  map[string]bool{"c1-large-x86": true, "c1-xlarge-x86": true},
				Images: []string{"ubuntu*", "debian*"},
			},
			args: args{
				size:  s1,
				image: i1,
			},
			want: true,
		},
		{
			name: "firewall layout no match",
			c: constraints{
				Sizes:  map[string]bool{"c1-large-x86": true, "c1-xlarge-x86": true},
				Images: []string{"firewall*"},
			},
			args: args{
				size:  s2,
				image: i1,
			},
			want: false,
		},
		{
			name: "firewall layout match",
			c: constraints{
				Sizes:  map[string]bool{"c1-large-x86": true, "c1-xlarge-x86": true},
				Images: []string{"firewall*"},
			},
			args: args{
				size:  s2,
				image: i3,
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &FilesystemLayoutConstraint{
				Sizes:  tt.c.Sizes,
				Images: tt.c.Images,
			}
			if got := c.Matches(tt.args.size, tt.args.image); got != tt.want {
				t.Errorf("FilesystemLayoutConstraint.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilesystemLayouts_From(t *testing.T) {
	type args struct {
		size  Size
		image Image
	}
	tests := []struct {
		name    string
		fls     FilesystemLayouts
		args    args
		want    *string
		wantErr bool
	}{
		{
			name: "simple match debian",
			fls: FilesystemLayouts{
				FilesystemLayout{
					Base: Base{ID: "default"},
					Constraint: FilesystemLayoutConstraint{
						Sizes:  map[string]bool{"c1-large-x86": true, "c1-xlarge-x86": true},
						Images: []string{"ubuntu*", "debian*"},
					},
				},
				FilesystemLayout{
					Base: Base{ID: "firewall"},
					Constraint: FilesystemLayoutConstraint{
						Sizes:  map[string]bool{"c1-large-x86": true, "c1-xlarge-x86": true},
						Images: []string{"firewall*"},
					},
				},
			},
			args: args{
				size:  s1,
				image: i1,
			},
			want:    strPtr("default"),
			wantErr: false,
		},
		{
			name: "simple match firewall",
			fls: FilesystemLayouts{
				FilesystemLayout{
					Base: Base{ID: "default"},
					Constraint: FilesystemLayoutConstraint{
						Sizes:  map[string]bool{"c1-large-x86": true, "c1-xlarge-x86": true},
						Images: []string{"ubuntu*", "debian*"},
					},
				},
				FilesystemLayout{
					Base: Base{ID: "firewall"},
					Constraint: FilesystemLayoutConstraint{
						Sizes:  map[string]bool{"c1-large-x86": true, "c1-xlarge-x86": true},
						Images: []string{"firewall*"},
					},
				},
			},
			args: args{
				size:  s1,
				image: i3,
			},
			want:    strPtr("firewall"),
			wantErr: false,
		},
		{
			name: "no match, wrong size",
			fls: FilesystemLayouts{
				FilesystemLayout{
					Base: Base{ID: "default"},
					Constraint: FilesystemLayoutConstraint{
						Sizes:  map[string]bool{"c1-large-x86": true, "c1-xlarge-x86": true},
						Images: []string{"ubuntu*", "debian*"},
					},
				},
				FilesystemLayout{
					Base: Base{ID: "firewall"},
					Constraint: FilesystemLayoutConstraint{
						Sizes:  map[string]bool{"c1-large-x86": true, "c1-xlarge-x86": true},
						Images: []string{"firewall*"},
					},
				},
			},
			args: args{
				size:  s3,
				image: i1,
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no match, wrong image",
			fls: FilesystemLayouts{
				FilesystemLayout{
					Base: Base{ID: "default"},
					Constraint: FilesystemLayoutConstraint{
						Sizes:  map[string]bool{"c1-large-x86": true, "c1-xlarge-x86": true},
						Images: []string{"ubuntu*", "debian*"},
					},
				},
				FilesystemLayout{
					Base: Base{ID: "firewall"},
					Constraint: FilesystemLayoutConstraint{
						Sizes:  map[string]bool{"c1-large-x86": true, "c1-xlarge-x86": true},
						Images: []string{"firewall*"},
					},
				},
			},
			args: args{
				size:  s1,
				image: i4,
			},
			want:    nil,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fls.From(tt.args.size, tt.args.image)
			if (err != nil) != tt.wantErr {
				t.Errorf("FilesystemLayouts.From() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				if tt.want != nil {
					t.Errorf("FilesystemLayouts.From() got nil was not expected")
				}
				return
			}
			if !reflect.DeepEqual(got.Base.ID, *tt.want) {
				t.Errorf("FilesystemLayouts.From() = %v, want %v", got, tt.want)
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}
