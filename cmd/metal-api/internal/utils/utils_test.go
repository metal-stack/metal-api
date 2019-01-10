package utils

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"git.f-i-ts.de/cloud-native/metal/metal-api/cmd/metal-api/internal/metal"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// A very simple health check.
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")

	// In the future we could report back on the status of our DB, or our cache
	// (e.g. Redis) by performing a simple PING, and include them in the response.
	io.WriteString(w, `{"alive": true}`)
}

func Test_loggingResponseWriter_Header(t *testing.T) {

	js, _ := json.Marshal(metal.RegisterSwitch{
		ID:     metal.Switch1.ID,
		SiteID: metal.Switch1.SiteID,
		RackID: metal.Switch1.RackID,
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
		ID:     metal.Switch1.ID,
		SiteID: metal.Switch1.SiteID,
		RackID: metal.Switch1.RackID,
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
		// TODO: Add test cases.
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
		ID:     metal.Switch1.ID,
		SiteID: metal.Switch1.SiteID,
		RackID: metal.Switch1.RackID,
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
		// TODO: Add test cases.
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
		ID:     metal.Switch1.ID,
		SiteID: metal.Switch1.SiteID,
		RackID: metal.Switch1.RackID,
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
		// TODO: Add test cases.
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

func TestRestfulLogger(t *testing.T) {

	z := zap.NewNop()
	X := RestfulLogger(z, false)

	// Only Pointer Comparison.
	require.Equal(t, reflect.ValueOf(X).Pointer(), reflect.ValueOf(RestfulLogger(z, false)).Pointer())
}
