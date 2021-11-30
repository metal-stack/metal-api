package metal

import (
	"errors"
	"testing"
)

var (
	n1Medium          = Size{Base: Base{ID: "n1-medium-x86"}}
	c1Xlarge          = Size{Base: Base{ID: "c1-xlarge-x86"}}
	s1Xlarge          = Size{Base: Base{ID: "s1-xlarge-x86"}}
	newFirewall       = Image{OS: "firewall", Version: "2.0.20211101"}
	oldFirewall       = Image{OS: "firewall", Version: "2.0.20201101"}
	onlyMajorFirewall = Image{OS: "firewall", Version: "2"}
	debian            = Image{OS: "debian", Version: "10.0.20201101"}
)

func TestSizeImageConstraint_matches(t *testing.T) {
	type fields struct {
		Base   Base
		Images map[string]string
	}
	type args struct {
		size  Size
		image Image
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr error
	}{
		{
			name: "n1 with matching firewall is allowed",
			fields: fields{
				Base: Base{ID: n1Medium.ID},
				Images: map[string]string{
					"firewall": ">= 2.0.20211001",
				},
			},
			args:    args{size: n1Medium, image: newFirewall},
			wantErr: nil,
		},
		{
			name: "n1 with to old firewall is not allowed",
			fields: fields{
				Base: Base{ID: n1Medium.ID},
				Images: map[string]string{
					"firewall": ">= 2.0.20211001",
					"ubuntu":   ">= 2.0.20211001",
				},
			},
			args:    args{size: n1Medium, image: oldFirewall},
			wantErr: errors.New("given size:n1-medium-x86 with image:firewall-2.0.20201101 does violate image constraint:firewall >=2.0.20211001"),
		},
		{
			name: "c1 has no restrictins",
			fields: fields{
				Base: Base{ID: n1Medium.ID},
				Images: map[string]string{
					"firewall": ">= 2.0.20211001",
				},
			},
			args:    args{size: c1Xlarge, image: oldFirewall},
			wantErr: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			sc := &SizeImageConstraint{
				Base:   tt.fields.Base,
				Images: tt.fields.Images,
			}
			got := sc.Matches(tt.args.size, tt.args.image)
			if tt.wantErr == nil && got != nil || got == nil && tt.wantErr != nil {
				t.Errorf("SizeImageConstraint.matches() = %v, want %v", got, tt.wantErr)
			}

			if tt.wantErr != nil && got != nil && got.Error() != tt.wantErr.Error() {
				t.Errorf("SizeImageConstraint.matches() got:%q want:%q", got, tt.wantErr)
			}
		})
	}
}

func TestSizeImageConstraints_Matches(t *testing.T) {
	type args struct {
		size  Size
		image Image
	}
	tests := []struct {
		name    string
		scs     *SizeImageConstraints
		args    args
		wantErr error
	}{
		{
			name:    "no constraints",
			scs:     &SizeImageConstraints{},
			args:    args{size: n1Medium, image: newFirewall},
			wantErr: nil,
		},
		{
			name: "new enough image",
			scs: &SizeImageConstraints{
				{
					Base: Base{ID: n1Medium.ID},
					Images: map[string]string{
						"firewall": ">= 2.0.20211001",
					},
				},
				{
					Base: Base{ID: c1Xlarge.ID},
					Images: map[string]string{
						"firewall": ">= 2.0.20211001",
					},
				},
			},
			args:    args{size: n1Medium, image: newFirewall},
			wantErr: nil,
		},
		{
			name: "only major version given",
			scs: &SizeImageConstraints{
				{
					Base: Base{ID: n1Medium.ID},
					Images: map[string]string{
						"firewall": ">= 2.0.20211001",
					},
				},
				{
					Base: Base{ID: c1Xlarge.ID},
					Images: map[string]string{
						"firewall": ">= 2.0.20211001",
					},
				},
			},
			args:    args{size: n1Medium, image: onlyMajorFirewall},
			wantErr: errors.New("no patch version given"),
		},
		{
			name: "no constraints for this image",
			scs: &SizeImageConstraints{
				{
					Base: Base{ID: n1Medium.ID},
					Images: map[string]string{
						"firewall": ">= 2.0.20211001",
					},
				},
				{
					Base: Base{ID: c1Xlarge.ID},
					Images: map[string]string{
						"firewall": ">= 2.0.20211001",
					},
				},
			},
			args:    args{size: s1Xlarge, image: debian},
			wantErr: nil,
		},
		{
			name: "to old image",
			scs: &SizeImageConstraints{
				{
					Base: Base{ID: n1Medium.ID},
					Images: map[string]string{
						"firewall": ">= 2.0.20211001",
					},
				},
				{
					Base: Base{ID: c1Xlarge.ID},
					Images: map[string]string{
						"firewall": ">= 2.0.20211001",
					},
				},
			},
			args:    args{size: n1Medium, image: oldFirewall},
			wantErr: errors.New("given size:n1-medium-x86 with image:firewall-2.0.20201101 does violate image constraint:firewall >=2.0.20211001"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := tt.scs.Matches(tt.args.size, tt.args.image)
			if tt.wantErr == nil && got != nil || got == nil && tt.wantErr != nil {
				t.Errorf("SizeImageConstraint.matches() = %v, want %v", got, tt.wantErr)
			}

			if tt.wantErr != nil && got != nil && got.Error() != tt.wantErr.Error() {
				t.Errorf("SizeImageConstraint.matches() got:%q want:%q", got, tt.wantErr)
			}
		})
	}
}

func TestSizeImageConstraints_Validate(t *testing.T) {
	tests := []struct {
		name    string
		scs     *SizeImageConstraints
		wantErr bool
	}{
		{
			name: "valid",
			scs: &SizeImageConstraints{
				{
					Base: Base{ID: n1Medium.ID},
					Images: map[string]string{
						"firewall": ">= 2.0.20211001",
					},
				},
				{
					Base: Base{ID: c1Xlarge.ID},
					Images: map[string]string{
						"debian": ">= 10.0",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no wildcard os",
			scs: &SizeImageConstraints{
				{
					Base: Base{ID: n1Medium.ID},
					Images: map[string]string{
						"*": "",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "wildcard version is allowed",
			scs: &SizeImageConstraints{
				{
					Base: Base{ID: n1Medium.ID},
					Images: map[string]string{
						"debian": "*",
					},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid op",
			scs: &SizeImageConstraints{
				{
					Base: Base{ID: n1Medium.ID},
					Images: map[string]string{
						"debian": "% 2",
					},
				},
			},
			wantErr: true,
		},
		{
			name: "invalid op and not seperated by space",
			scs: &SizeImageConstraints{
				{
					Base: Base{ID: n1Medium.ID},
					Images: map[string]string{
						"debian": "%2",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.scs.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("SizeImageConstraints.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
