package service

import (
	"fmt"
	"net/http"
	"testing"

	"git.f-i-ts.de/cloud-native/maas/maas-service/cmd/maas-api/internal/utils"
	"git.f-i-ts.de/cloud-native/maas/maas-service/pkg/maas"
	restful "github.com/emicklei/go-restful"
	"github.com/stretchr/testify/suite"
)

var (
	sr sizeResource
)

func init() {
	// dummy as long we do not have a database
	sr = sizeResource{
		sizes: make(map[string]*maas.Size),
	}
	restful.Add(sr.webService())
}

type SizeTestSuite struct {
	suite.Suite
	sr sizeResource
}

func (s *SizeTestSuite) SetupTest() {
	s.sr = sr
	addDummySizes(s.sr.sizes)
}

func (s *SizeTestSuite) TearDownTest() {
	deleteSizes(s.sr.sizes)
}

func TestSizeTestSuite(t *testing.T) {
	suite.Run(t, new(SizeTestSuite))
}

func (s *SizeTestSuite) TestGetSizes() {
	httpWriter, httpRequest := utils.HttpMock("GET", "/size", "")

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)
	var result []*maas.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	s.Assert().Equal(len(dummySizes), len(result), "Not all sizes were returned")
}

func (s *SizeTestSuite) TestGetSize() {
	size := dummySizes[0]
	httpWriter, httpRequest := utils.HttpMock("GET", fmt.Sprintf("/size/%s", size.ID), "")

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)
	var result *maas.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	s.Assert().Equal(size.ID, result.ID, "Size was not returned")
}

func (s *SizeTestSuite) TestDeletingSize() {
	sizeToDelete := "m2.xlarge.x86"
	beforeSizes := getSizes(s.sr)
	httpWriter, httpRequest := utils.HttpMock("DELETE", fmt.Sprintf("/size/%s", sizeToDelete), "")

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)
	var result *maas.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	s.Assert().Equal(sizeToDelete, result.ID, "Deleted size id was not returned")
	afterSizes := getSizes(s.sr)
	s.Assert().NotContains(afterSizes, sizeToDelete, "Deleted size still exists")
	s.Assert().Len(afterSizes, len(beforeSizes)-1, "Same amount of sizes before and after deletion")
}

func (s *SizeTestSuite) TestDeletingUnexistingImage() {
	httpWriter, httpRequest := utils.HttpMock("DELETE", "/size/something", "")

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)
	resp, body, err := utils.ParseHTTPResponse(s.T(), httpWriter, nil)

	s.Assert().Equal(http.StatusNotFound, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not readable", err)
	s.Assert().Contains(string(body), `id "something" not found`, "No proper error message in response")
}

func (s *SizeTestSuite) TestCreateSize() {
	sizeToCreate := &maas.Size{
		ID:          "new.size.x86",
		Name:        "new.size.x86",
		Description: "A test size.",
	}
	beforeSizes := getSizes(s.sr)
	httpWriter, httpRequest := utils.HttpMock("PUT", "/size", sizeToCreate)

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)
	var result *maas.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Assert().Equal(http.StatusCreated, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	afterSizes := getSizes(s.sr)
	s.Assert().Len(afterSizes, len(beforeSizes)+1, "Same amount of sizes before and after creation")
	createdSize, err := getSize(s.sr, sizeToCreate.ID)
	s.Require().Nil(err, "Created size not found")
	s.Assert().Equal(createdSize.ID, sizeToCreate.ID, "Size created more than once")
}

func (s *SizeTestSuite) TestCreateSizeInvalidPayload() {
	httpWriter, httpRequest := utils.HttpMock("PUT", "/size", "something")

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)
	resp, body, err := utils.ParseHTTPResponse(s.T(), httpWriter, nil)

	s.Assert().Equal(http.StatusInternalServerError, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not readable", err)
	s.Assert().Contains(string(body), `cannot read size from request: json`, "No proper error message in response")
}

func (s *SizeTestSuite) TestUpdateSize() {
	sizeToUpdate := dummySizes[0]
	sizeToUpdate.Description = "Modified Description"
	beforeSizes := getSizes(s.sr)
	httpWriter, httpRequest := utils.HttpMock("POST", "/size", sizeToUpdate)

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)
	var result *maas.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	afterSizes := getSizes(s.sr)
	s.Assert().Len(afterSizes, len(beforeSizes), "Different amount of sizes after update")
	updatedSize, err := getSize(s.sr, sizeToUpdate.ID)
	s.Require().Nil(err, "Updated size not found")
	s.Assert().Equal(updatedSize.Description, sizeToUpdate.Description, "Field was not updated properly")
}
