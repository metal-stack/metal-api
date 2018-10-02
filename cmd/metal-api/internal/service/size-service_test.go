package service

import (
	"fmt"
	"net/http"
	"testing"

	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/datastore/hashmapstore"
	"git.f-i-ts.de/cloud-native/maas/metal-api/cmd/metal-api/internal/utils"
	"git.f-i-ts.de/cloud-native/maas/metal-api/pkg/metal"
	restful "github.com/emicklei/go-restful"
	"github.com/stretchr/testify/suite"
)

var (
	store *hashmapstore.HashmapStore
	sr    sizeResource
)

type SizeTestSuite struct {
	suite.Suite
	sr sizeResource
	ws *restful.Container
}

func (s *SizeTestSuite) SetupTest() {
	store := hashmapstore.NewHashmapStore()
	store.AddMockData()
	s.sr = sizeResource{ds: store}
	s.ws = restful.NewContainer()
	s.ws.Add(s.sr.webService())
}

func TestSizeTestSuite(t *testing.T) {
	suite.Run(t, new(SizeTestSuite))
}

func (s *SizeTestSuite) TestGetSizes() {
	httpWriter, httpRequest := utils.HttpMock("GET", "/size", "")

	s.ws.ServeHTTP(httpWriter, httpRequest)
	var result []*metal.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	s.Assert().Equal(len(hashmapstore.DummySizes), len(result), "Not all sizes were returned")
}

func (s *SizeTestSuite) TestGetSize() {
	size := hashmapstore.DummySizes[0]
	httpWriter, httpRequest := utils.HttpMock("GET", fmt.Sprintf("/size/%s", size.ID), "")

	s.ws.ServeHTTP(httpWriter, httpRequest)
	var result *metal.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	s.Assert().Equal(size.ID, result.ID, "Size was not returned")
}

func (s *SizeTestSuite) TestDeletingSize() {
	sizeToDelete := "m2.xlarge.x86"
	beforeSizes, err := s.sr.ds.ListSizes()
	s.Require().NoError(err)
	httpWriter, httpRequest := utils.HttpMock("DELETE", fmt.Sprintf("/size/%s", sizeToDelete), "")

	s.ws.ServeHTTP(httpWriter, httpRequest)
	var result *metal.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Require().Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	s.Assert().Equal(sizeToDelete, result.ID, "Deleted size id was not returned")
	afterSizes, err := s.sr.ds.ListSizes()
	s.NoError(err)
	s.Assert().NotContains(afterSizes, sizeToDelete, "Deleted size still exists")
	s.Assert().Len(afterSizes, len(beforeSizes)-1, "Same amount of sizes before and after deletion")
}

func (s *SizeTestSuite) TestDeletingUnexistingImage() {
	httpWriter, httpRequest := utils.HttpMock("DELETE", "/size/something", "")

	s.ws.ServeHTTP(httpWriter, httpRequest)
	resp, body, err := utils.ParseHTTPResponse(s.T(), httpWriter, nil)

	s.Assert().Equal(http.StatusNotFound, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not readable", err)
	s.Assert().Contains(string(body), `id "something" not found`, "No proper error message in response")
}

func (s *SizeTestSuite) TestCreateSize() {
	sizeToCreate := &metal.Size{
		ID:          "new.size.x86",
		Name:        "new.size.x86",
		Description: "A test size.",
	}
	beforeSizes, err := s.sr.ds.ListSizes()
	s.NoError(err)
	httpWriter, httpRequest := utils.HttpMock("PUT", "/size", sizeToCreate)

	s.ws.ServeHTTP(httpWriter, httpRequest)
	var result *metal.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Require().Equal(http.StatusCreated, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	afterSizes, err := s.sr.ds.ListSizes()
	s.NoError(err)
	s.Assert().Len(afterSizes, len(beforeSizes)+1, "Same amount of sizes before and after creation")
	createdSize, err := s.sr.ds.FindSize(sizeToCreate.ID)
	s.Require().Nil(err, "Created size not found")
	s.Assert().Equal(createdSize.ID, sizeToCreate.ID, "Size created more than once")
}

func (s *SizeTestSuite) TestCreateSizeInvalidPayload() {
	httpWriter, httpRequest := utils.HttpMock("PUT", "/size", "something")

	s.ws.ServeHTTP(httpWriter, httpRequest)
	resp, body, err := utils.ParseHTTPResponse(s.T(), httpWriter, nil)

	s.Assert().Equal(http.StatusInternalServerError, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not readable", err)
	s.Assert().Contains(string(body), `cannot read size from request: json`, "No proper error message in response")
}

func (s *SizeTestSuite) TestUpdateSize() {
	sizeToUpdate := hashmapstore.DummySizes[0]
	sizeToUpdate.Description = "Modified Description"
	beforeSizes, err := s.sr.ds.ListSizes()
	s.NoError(err)
	httpWriter, httpRequest := utils.HttpMock("POST", "/size", sizeToUpdate)

	s.ws.ServeHTTP(httpWriter, httpRequest)
	var result *metal.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	afterSizes, err := s.sr.ds.ListSizes()
	s.NoError(err)
	s.Assert().Len(afterSizes, len(beforeSizes), "Different amount of sizes after update")
	updatedSize, err := s.sr.ds.FindSize(sizeToUpdate.ID)
	s.Require().Nil(err, "Updated size not found")
	s.Assert().Equal(updatedSize.Description, sizeToUpdate.Description, "Field was not updated properly")
}
