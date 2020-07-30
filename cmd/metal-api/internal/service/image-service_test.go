package service

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/testdata"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-lib/httperrors"
	"github.com/stretchr/testify/require"
)

func TestGetImages(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	imageservice := NewImage(ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("GET", "/v1/image", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result []v1.ImageResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Len(t, result, 4)
	require.Equal(t, testdata.Img1.ID, result[0].ID)
	require.Equal(t, testdata.Img1.Name, *result[0].Name)
	require.Equal(t, testdata.Img2.ID, result[1].ID)
	require.Equal(t, testdata.Img2.Name, *result[1].Name)
	require.Equal(t, testdata.Img3.ID, result[2].ID)
	require.Equal(t, testdata.Img3.Name, *result[2].Name)
}

func TestGetImage(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	imageservice := NewImage(ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("GET", "/v1/image/image-1", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.ImageResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Img1.ID, result.ID)
	require.Equal(t, testdata.Img1.Name, *result.Name)
}

func TestGetImageNotFound(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	imageservice := NewImage(ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("GET", "/v1/image/image-999", nil)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusNotFound, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Contains(t, result.Message, "999")
	require.Equal(t, 404, result.StatusCode)
}

func TestDeleteImage(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	imageservice := NewImage(ds)
	container := restful.NewContainer().Add(imageservice)
	req := httptest.NewRequest("DELETE", "/v1/image/image-3", nil)
	container = injectAdmin(container, req)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.ImageResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Img3.ID, result.ID)
	require.Equal(t, testdata.Img3.Name, *result.Name)
}

func TestCreateImage(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	createRequest := v1.ImageCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: testdata.Img1.ID,
			},
			Describable: v1.Describable{
				Name:        &testdata.Img1.Name,
				Description: &testdata.Img1.Description,
			},
		},
		URL: testdata.Img1.URL,
	}
	js, _ := json.Marshal(createRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/image", body)
	container := injectAdmin(restful.NewContainer().Add(NewImage(ds)), req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.ImageResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Img1.ID, result.ID)
	require.Equal(t, testdata.Img1.Name, *result.Name)
	require.Equal(t, testdata.Img1.Description, *result.Description)
	require.Equal(t, testdata.Img1.URL, *result.URL)
	require.Equal(t, string(testdata.Img1.Classification), result.Classification)
	require.False(t, result.ExpirationDate.IsZero())

}

func TestCreateImageWithBrokenURL(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	createRequest := v1.ImageCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: testdata.Img1.ID,
			},
			Describable: v1.Describable{
				Name:        &testdata.Img1.Name,
				Description: &testdata.Img1.Description,
			},
		},
		URL: "http://this.domain.does.not.exist/",
	}
	js, _ := json.Marshal(createRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/image", body)
	container := injectAdmin(restful.NewContainer().Add(NewImage(ds)), req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, w.Body.String())
	var result httperrors.HTTPErrorResponse
	err := json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, "image:image-1 is not accessible under:http://this.domain.does.not.exist/ error:Head \"http://this.domain.does.not.exist/\": dial tcp: lookup this.domain.does.not.exist: no such host", result.Message)

	createRequest.URL = "http://images.metal-stack.io/this-file-does-not-exist"

	js, _ = json.Marshal(createRequest)
	body = bytes.NewBuffer(js)
	req = httptest.NewRequest("PUT", "/v1/image", body)
	container = injectAdmin(restful.NewContainer().Add(NewImage(ds)), req)
	req.Header.Add("Content-Type", "application/json")
	w = httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp = w.Result()
	require.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode, w.Body.String())

	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	require.Equal(t, "image:image-1 is not accessible under:http://images.metal-stack.io/this-file-does-not-exist status:404 Not Found", result.Message)
}

func TestCreateImageWithClassification(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)
	vc := string(testdata.Img1.Classification)

	createRequest := v1.ImageCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: testdata.Img1.ID,
			},
			Describable: v1.Describable{
				Name:        &testdata.Img1.Name,
				Description: &testdata.Img1.Description,
			},
		},
		URL:            testdata.Img1.URL,
		Classification: &vc,
	}
	js, _ := json.Marshal(createRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("PUT", "/v1/image", body)
	container := injectAdmin(restful.NewContainer().Add(NewImage(ds)), req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.ImageResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Img1.ID, result.ID)
	require.Equal(t, testdata.Img1.Name, *result.Name)
	require.Equal(t, testdata.Img1.Description, *result.Description)
	require.Equal(t, testdata.Img1.URL, *result.URL)
	require.Equal(t, string(testdata.Img1.Classification), result.Classification)
	require.False(t, result.ExpirationDate.IsZero())

}

func TestUpdateImage(t *testing.T) {
	ds, mock := datastore.InitMockDB()
	testdata.InitMockDBData(mock)

	imageservice := NewImage(ds)
	container := restful.NewContainer().Add(imageservice)

	updateRequest := v1.ImageUpdateRequest{
		Common: v1.Common{
			Describable: v1.Describable{
				Name:        &testdata.Img2.Name,
				Description: &testdata.Img2.Description,
			},
			Identifiable: v1.Identifiable{
				ID: testdata.Img1.ID,
			},
		},
		ImageBase: v1.ImageBase{
			URL: &testdata.Img2.URL,
		},
	}
	js, _ := json.Marshal(updateRequest)
	body := bytes.NewBuffer(js)
	req := httptest.NewRequest("POST", "/v1/image", body)
	container = injectAdmin(container, req)
	req.Header.Add("Content-Type", "application/json")
	w := httptest.NewRecorder()
	container.ServeHTTP(w, req)

	resp := w.Result()
	require.Equal(t, http.StatusOK, resp.StatusCode, w.Body.String())
	var result v1.ImageResponse
	err := json.NewDecoder(resp.Body).Decode(&result)

	require.Nil(t, err)
	require.Equal(t, testdata.Img1.ID, result.ID)
	require.Equal(t, testdata.Img2.Name, *result.Name)
	require.Equal(t, testdata.Img2.Description, *result.Description)
	require.Equal(t, testdata.Img2.URL, *result.URL)
}
