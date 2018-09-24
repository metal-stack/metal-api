package utils

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"

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

func ParseJSONResponse(httpWriter *httptest.ResponseRecorder, object interface{}) (*http.Response, []byte, error) {
	resp := httpWriter.Result()
	body, _ := ioutil.ReadAll(resp.Body)
	err := json.Unmarshal(body, &object)
	return resp, body, err
}

func ParsePlainTextResponse(httpWriter *httptest.ResponseRecorder) (*http.Response, []byte, error) {
	resp := httpWriter.Result()
	body, err := ioutil.ReadAll(resp.Body)
	return resp, body, err
}
