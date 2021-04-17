package metal

import "testing"

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
