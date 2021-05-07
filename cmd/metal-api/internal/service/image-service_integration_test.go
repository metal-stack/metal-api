// +build integration

package service

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	restful "github.com/emicklei/go-restful/v3"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/datastore"
	"github.com/metal-stack/metal-api/cmd/metal-api/internal/metal"
	v1 "github.com/metal-stack/metal-api/cmd/metal-api/internal/service/v1"
	"github.com/metal-stack/metal-api/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestGetImagesIntegration(t *testing.T) {
	rethinkContainer, c, err := test.StartRethink()
	require.NoError(t, err)
	defer rethinkContainer.Terminate(context.Background())

	ds := datastore.New(zaptest.NewLogger(t), c.IP+":"+c.Port, c.DB, c.User, c.Password)
	ds.VRFPoolRangeMax = 1000
	ds.ASNPoolRangeMax = 1000

	err = ds.Connect()
	require.NoError(t, err)
	err = ds.Initialize()
	require.NoError(t, err)

	imageservice := NewImage(ds)
	container := restful.NewContainer().Add(imageservice)

	imageID := "test-image-1.0.0"
	imageName := "testimage"
	imageDesc := "Test Image"
	newImage := v1.ImageCreateRequest{
		Common: v1.Common{
			Identifiable: v1.Identifiable{
				ID: imageID,
			},
			Describable: v1.Describable{
				Name:        &imageName,
				Description: &imageDesc,
			},
		},
		URL:      "https://www.google.com", // not good to rely on this page
		Features: []string{string(metal.ImageFeatureMachine)},
	}

	ji, err := json.Marshal(newImage)
	require.NoError(t, err)
	body := ioutil.NopCloser(strings.NewReader(string(ji)))
	createReq := httptest.NewRequest(http.MethodPut, "/v1/image", body)
	createReq.Header.Set("Content-Type", "application/json")

	container = injectAdmin(container, createReq)
	w := httptest.NewRecorder()
	container.ServeHTTP(w, createReq)

	resp := w.Result()
	defer resp.Body.Close()
	require.Equal(t, http.StatusCreated, resp.StatusCode, w.Body.String())
	var result v1.ImageResponse
	err = json.NewDecoder(resp.Body).Decode(&result)
	require.NoError(t, err)
	assert.Equal(t, newImage.ID, result.ID)
	assert.Equal(t, newImage.Name, result.Name)
	assert.Equal(t, newImage.Description, result.Description)
	assert.Equal(t, newImage.URL, *result.URL)
	require.Len(t, result.Features, 1)
	assert.Equal(t, newImage.Features[0], result.Features[0])
}
