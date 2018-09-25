package service

import (
	"fmt"
	"net/http"
	"testing"

	"git.f-i-ts.de/ize0h88/maas-service/cmd/maas-api/internal/utils"
	"git.f-i-ts.de/ize0h88/maas-service/pkg/maas"
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

func (s *SizeTestSuite) TestGetSpecificSizes() {
	ids := []string{"t1.small.x86", "m2.xlarge.x86"}
	httpWriter, httpRequest := utils.HttpMock("GET", fmt.Sprintf("/size?id=%s&id=%s", ids[0], ids[1]), "")

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)
	var result []*maas.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	s.Assert().Len(result, len(ids), "More than two sizes were returned")
	for _, size := range result {
		s.Assert().Contains(ids, size.ID, "Size not contained in result")
	}
}

func (s *SizeTestSuite) TestDeletingSize() {
	sizeToDelete := "m2.xlarge.x86"
	beforeSizes := getSize(s.sr, []string{})
	httpWriter, httpRequest := utils.HttpMock("DELETE", fmt.Sprintf("/size/%s", sizeToDelete), "")

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)
	var result *maas.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	s.Assert().Equal(sizeToDelete, result.ID, "Deleted size id was not returned")
	afterSizes := getSize(s.sr, []string{})
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
	beforeSizes := getSize(s.sr, []string{})
	httpWriter, httpRequest := utils.HttpMock("PUT", "/size", sizeToCreate)

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)
	var result *maas.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Assert().Equal(http.StatusCreated, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	afterSizes := getSize(s.sr, []string{})
	s.Assert().Len(afterSizes, len(beforeSizes)+1, "Same amount of sizes before and after creation")
	createdSizes := getSize(s.sr, []string{sizeToCreate.ID})
	s.Assert().Len(createdSizes, 1, "Size created more than once")
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
	beforeSizes := getSize(s.sr, []string{})
	httpWriter, httpRequest := utils.HttpMock("POST", "/size", sizeToUpdate)

	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)
	var result *maas.Size
	resp, _, err := utils.ParseHTTPResponse(s.T(), httpWriter, &result)

	s.Assert().Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	s.Require().Nil(err, "Response not JSON parsable", err)
	afterSizes := getSize(s.sr, []string{})
	s.Assert().Len(afterSizes, len(beforeSizes), "Different amount of sizes after update")
	updatedSizes := getSize(s.sr, []string{sizeToUpdate.ID})
	s.Require().Len(updatedSizes, 1, "Updated size found more than once")
	updatedSize := updatedSizes[0]
	s.Assert().Equal(updatedSize.Description, sizeToUpdate.Description, "Field was not updated properly")
}
