package utils

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	restful "github.com/emicklei/go-restful"
)

func HttpMock(method string, url string, payload interface{}) (*httptest.ResponseRecorder, *http.Request) {
	json, err := json.Marshal(payload)
	if err != nil {
		panic("Unable to marshal JSON payload")
	}
	bodyReader := strings.NewReader(string(json))
	httpRequest, _ := http.NewRequest(method, url, bodyReader)
	httpRequest.Header.Set("Content-Type", restful.MIME_JSON)
	httpWriter := httptest.NewRecorder()
	return httpWriter, httpRequest
}

func ParseHTTPResponse(t *testing.T, httpWriter *httptest.ResponseRecorder, object interface{}) (*http.Response, []byte, error) {
	resp := httpWriter.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	var err error
	if object != nil && httpWriter.Header().Get("Content-Type") == restful.MIME_JSON {
		err = json.Unmarshal(body, &object)
	}
	t.Log("Response body:", string(body))
	return resp, body, err
}
