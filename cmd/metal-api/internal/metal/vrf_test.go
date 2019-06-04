package metal

import (
	"testing"
)

func TestGenerateVrfID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:    "generate vrf id",
			input:   "devops",
			want:    "43750",
			wantErr: false,
		},
		{
			name:    "generate vrf id",
			input:   "someoneelse",
			want:    "59010",
			wantErr: false,
		},
		{
			name:    "generate vrf id",
			input:   "someonewithanumber1",
			want:    "27116",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateVrfID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateVrfID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GenerateVrfID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVrf_ToUint(t *testing.T) {
	tests := []struct {
		name    string
		id  string
		want    uint
		wantErr bool
	}{
		{
			name: "vrf to uint",
			id: "42000",
			want: 42000,
			wantErr: false,
		},
		{
			name: "vrf to uint with error",
			id: "42000kl",
			want: 0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := &Vrf{
				Base:      Base{
					ID: tt.id,
				},
			}
			got, err := v.ToUint()
			if (err != nil) != tt.wantErr {
				t.Errorf("Vrf.ToUint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Vrf.ToUint() = %v, want %v", got, tt.want)
			}
		})
	}
}
