package service

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/emicklei/go-restful"
)

func TestGetApiDocsNotFound(t *testing.T) {

	apidocsservice := NewApiDoc("/notexisting/path.html")
	container := restful.NewContainer().Add(apidocsservice)
	req := httptest.NewRequest("GET", "/apidocs.html", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
}

func TestGetApiDocsOk(t *testing.T) {

	apidocsservice := NewApiDoc("./testdata/doc.html")
	container := restful.NewContainer().Add(apidocsservice)
	req := httptest.NewRequest("GET", "/apidocs.html", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	require.Equal(t, "<html><body>Ok</body></html>", w.Body.String())
}
