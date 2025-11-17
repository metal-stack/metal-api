package service

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/security"
	"github.com/stretchr/testify/require"
)

//nolint:deadcode,unused
type emptyBody struct{}

func webRequestPut(t require.TestingT, service *restful.WebService, user *security.User, request interface{}, path string, response interface{}) int {
	return webRequest(t, http.MethodPut, service, user, request, path, response)
}

func webRequestPost(t require.TestingT, service *restful.WebService, user *security.User, request interface{}, path string, response interface{}) int {
	return webRequest(t, http.MethodPost, service, user, request, path, response)
}

func webRequestDelete(t require.TestingT, service *restful.WebService, user *security.User, request interface{}, path string, response interface{}) int {
	return webRequest(t, http.MethodDelete, service, user, request, path, response)
}

func webRequestGet(t require.TestingT, service *restful.WebService, user *security.User, request interface{}, path string, response interface{}) int {
	return webRequest(t, http.MethodGet, service, user, request, path, response)
}

func webRequest(t require.TestingT, method string, service *restful.WebService, user *security.User, request interface{}, path string, response interface{}) int {
	container := restful.NewContainer().Add(service)

	jsonBody, err := json.Marshal(request)
	require.NoError(t, err)
	body := io.NopCloser(strings.NewReader(string(jsonBody)))
	createReq := httptest.NewRequest(method, path, body)
	createReq.Header.Set("Content-Type", "application/json")

	container.Filter(MockAuth(user))

	w := httptest.NewRecorder()
	container.ServeHTTP(w, createReq)

	resp := w.Result()
	defer resp.Body.Close()

	err = json.NewDecoder(resp.Body).Decode(response)
	require.NoError(t, err)
	return resp.StatusCode
}

func genericWebRequest[E any](t *testing.T, service *restful.WebService, user *security.User, body any, method string, path string) (int, E) {
	var encoded []byte

	if body != nil {
		var err error
		encoded, err = json.Marshal(body)
		require.NoError(t, err)
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(encoded))

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept", "application/json")

	recorder := httptest.NewRecorder()

	container := restful.NewContainer().Add(service)
	container.Filter(MockAuth(user))
	container.ServeHTTP(recorder, req)

	res := recorder.Result()
	defer res.Body.Close()

	var got E
	err := json.Unmarshal(recorder.Body.Bytes(), &got)
	require.NoError(t, err, "unable to parse response into %T: %s", got, recorder.Body.String())

	return recorder.Code, got
}

func MockAuth(user *security.User) restful.FilterFunction {
	return func(req *restful.Request, resp *restful.Response, chain *restful.FilterChain) {
		rq := req.Request
		ctx := security.PutUserInContext(context.Background(), user)
		req.Request = rq.WithContext(ctx)
		chain.ProcessFilter(req, resp)
	}
}

type NopPublisher struct {
}

func (p NopPublisher) Publish(topic string, data interface{}) error {
	return nil
}

func (p NopPublisher) CreateTopic(topic string) error {
	return nil
}

func (p NopPublisher) Stop() {}
