package utils

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/testdata"
)

func Test_loggingResponseWriter_Header(t *testing.T) {

	js, _ := json.Marshal(metal.RegisterSwitch{
		ID:          testdata.Switch1.ID,
		PartitionID: testdata.Switch1.PartitionID,
		RackID:      testdata.Switch1.RackID,
	})

	recorder1 := httptest.NewRecorder()
	recorder1.Header().Set("Content-Type", "application/json")

	recorder2 := httptest.NewRecorder()
	recorder2.Header().Set("Content-Type", "application/json")

	type fields struct {
		w      http.ResponseWriter
		buf    bytes.Buffer
		header int
	}
	tests := []struct {
		name   string
		fields fields
		want   http.Header
	}{
		{
			name: "Test 1",
			fields: fields{
				w:      recorder1,
				buf:    *bytes.NewBuffer(js),
				header: http.StatusOK,
			},
			want: recorder2.Header(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &loggingResponseWriter{
				w:      tt.fields.w,
				buf:    tt.fields.buf,
				header: tt.fields.header,
			}

			if got := w.Header(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loggingResponseWriter.Header() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_loggingResponseWriter_Write(t *testing.T) {

	js, _ := json.Marshal(metal.RegisterSwitch{
		ID:          testdata.Switch1.ID,
		PartitionID: testdata.Switch1.PartitionID,
		RackID:      testdata.Switch1.RackID,
	})

	recorder1 := httptest.NewRecorder()
	recorder1.Header().Set("Content-Type", "application/json")

	recorder2 := httptest.NewRecorder()
	recorder2.Header().Set("Content-Type", "application/json")

	type fields struct {
		w      http.ResponseWriter
		buf    bytes.Buffer
		header int
	}
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		// Test-Data List / Test Cases:
		{
			name: "Test 1",
			fields: fields{
				w:      recorder1,
				buf:    *bytes.NewBuffer(js),
				header: http.StatusOK,
			},
			args: args{
				b: js,
			},
			want: len(js),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &loggingResponseWriter{
				w:      tt.fields.w,
				buf:    tt.fields.buf,
				header: tt.fields.header,
			}
			got, err := w.Write(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("loggingResponseWriter.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("loggingResponseWriter.Write() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_loggingResponseWriter_WriteHeader(t *testing.T) {

	js, _ := json.Marshal(metal.RegisterSwitch{
		ID:          testdata.Switch1.ID,
		PartitionID: testdata.Switch1.PartitionID,
		RackID:      testdata.Switch1.RackID,
	})

	recorder1 := httptest.NewRecorder()
	recorder1.Header().Set("Content-Type", "application/json")

	recorder2 := httptest.NewRecorder()
	recorder2.Header().Set("Content-Type", "application/json")

	type fields struct {
		w      http.ResponseWriter
		buf    bytes.Buffer
		header int
	}
	type args struct {
		h int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// Test-Data List / Test Cases:
		{
			name: "Test 1",
			fields: fields{
				w:      recorder1,
				buf:    *bytes.NewBuffer(js),
				header: http.StatusOK,
			},
			args: args{
				h: http.StatusOK,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &loggingResponseWriter{
				w:      tt.fields.w,
				buf:    tt.fields.buf,
				header: tt.fields.header,
			}
			w.WriteHeader(tt.args.h)
		})
	}
}

func Test_loggingResponseWriter_Content(t *testing.T) {

	js, _ := json.Marshal(metal.RegisterSwitch{
		ID:          testdata.Switch1.ID,
		PartitionID: testdata.Switch1.PartitionID,
		RackID:      testdata.Switch1.RackID,
	})

	recorder1 := httptest.NewRecorder()
	recorder1.Header().Set("Content-Type", "application/json")

	recorder2 := httptest.NewRecorder()
	recorder2.Header().Set("Content-Type", "application/json")

	type fields struct {
		w      http.ResponseWriter
		buf    bytes.Buffer
		header int
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// Test-Data List / Test Cases:
		{
			name: "Test 1",
			fields: fields{
				w:      recorder1,
				buf:    *bytes.NewBuffer(js),
				header: http.StatusOK,
			},
			want: string(js),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &loggingResponseWriter{
				w:      tt.fields.w,
				buf:    tt.fields.buf,
				header: tt.fields.header,
			}
			if got := w.Content(); got != tt.want {
				t.Errorf("loggingResponseWriter.Content() = %v, want %v", got, tt.want)
			}
		})
	}
}
