package service

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"git.f-i-ts.de/ize0h88/maas-service/pkg/maas"
	restful "github.com/emicklei/go-restful"
)

func init() {
	restful.Add(NewSize())
}

func TestGetImages(t *testing.T) {
	bodyReader := strings.NewReader("")
	httpRequest, _ := http.NewRequest("GET", "/size", bodyReader)
	httpRequest.Header.Set("Content-Type", restful.MIME_JSON)
	httpWriter := httptest.NewRecorder()

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)

	resp := httpWriter.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	expectedStatusCode := 200

	if resp.StatusCode != expectedStatusCode {
		t.Errorf("Status code was %d, expected: %d", resp.StatusCode, expectedStatusCode)
	}

	var result []*maas.Size

	if err := json.Unmarshal(body, &result); err != nil {
		t.Error("Response not JSON parsable", err)
	}

	if len(result) != len(dummySizes) {
		t.Errorf("Not all sizes were returned")
		t.Error(string(body))
	}
}

func TestGetSpecificImages(t *testing.T) {
	bodyReader := strings.NewReader("")
	httpRequest, _ := http.NewRequest("GET", "/size?id=t1.small.x86&id=m2.xlarge.x86", bodyReader)
	httpRequest.Header.Set("Content-Type", restful.MIME_JSON)
	httpWriter := httptest.NewRecorder()

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)

	resp := httpWriter.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	expectedStatusCode := 200

	if resp.StatusCode != expectedStatusCode {
		t.Errorf("Status code was %d, expected: %d", resp.StatusCode, expectedStatusCode)
	}

	var result []*maas.Size

	if err := json.Unmarshal(body, &result); err != nil {
		t.Error("Response not JSON parsable", err)
	}

	if len(result) != 2 {
		t.Errorf("More than two sizes were returned")
		t.Error(string(body))
	}
}
