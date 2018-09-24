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

func (suite *SizeTestSuite) SetupTest() {
	suite.sr = sr
	for _, ds := range dummySizes {
		suite.sr.sizes[ds.ID] = ds
	}
}

func (suite *SizeTestSuite) TearDownTest() {
	for _, size := range suite.sr.sizes {
		delete(suite.sr.sizes, size.ID)
	}
}

func TestSizeTestSuite(t *testing.T) {
	suite.Run(t, new(SizeTestSuite))
}

func (suite *SizeTestSuite) TestGetSizes() {
	require := suite.Require()
	httpWriter, httpRequest := utils.HttpMock("GET", "/size", "")
	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)

	var result []*maas.Size
	resp, body, err := utils.ParseJSONResponse(httpWriter, &result)

	require.Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	require.Nil(err, "Response not JSON parsable", err)
	require.Equal(len(dummySizes), len(result), "Not all sizes were returned", string(body))
}

func (suite *SizeTestSuite) TestGetSpecificSizes() {
	require := suite.Require()
	ids := []string{"t1.small.x86", "m2.xlarge.x86"}
	httpWriter, httpRequest := utils.HttpMock("GET", fmt.Sprintf("/size?id=%s&id=%s", ids[0], ids[1]), "")
	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)

	var result []*maas.Size
	resp, body, err := utils.ParseJSONResponse(httpWriter, &result)

	require.Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	require.Nil(err, "Response not JSON parsable", err)
	require.Len(result, len(ids), "More than two sizes were returned", string(body))
	for _, size := range result {
		require.Contains(ids, size.ID, "Size not contained in result")
	}
}

func (suite *SizeTestSuite) TestDeletingSize() {
	require := suite.Require()
	sizeToDelete := "m2.xlarge.x86"
	beforeSizes := getSize(suite.sr, []string{})
	httpWriter, httpRequest := utils.HttpMock("DELETE", fmt.Sprintf("/size/%s", sizeToDelete), "")
	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)

	var result *maas.Size
	resp, body, err := utils.ParseJSONResponse(httpWriter, &result)

	require.Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	require.Nil(err, "Response not JSON parsable", err)
	require.Equal(sizeToDelete, result.ID, "Deleted size id was not returned", string(body))
	afterSizes := getSize(suite.sr, []string{})
	require.NotContains(afterSizes, sizeToDelete, "Deleted size still exists")
	require.Len(afterSizes, len(beforeSizes)-1, "Same amount of sizes before and after deletion")

}

func (suite *SizeTestSuite) TestDeletingUnexistingImage() {
	require := suite.Require()
	httpWriter, httpRequest := utils.HttpMock("DELETE", "/size/something", "")
	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)

	resp, body, err := utils.ParsePlainTextResponse(httpWriter)

	require.Equal(http.StatusNotFound, resp.StatusCode, "Wrong status code in response")
	require.Nil(err, "Response not readable", err)
	require.Contains(string(body), `id "something" not found`, "No proper error message in response", string(body))
}

func (suite *SizeTestSuite) TestCreateSize() {
	require := suite.Require()
	sizeToCreate := &maas.Size{
		ID:          "new.size.x86",
		Name:        "new.size.x86",
		Description: "A test size.",
	}
	beforeSizes := getSize(suite.sr, []string{})
	httpWriter, httpRequest := utils.HttpMock("PUT", "/size", sizeToCreate)
	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)

	var result *maas.Size
	resp, body, err := utils.ParseJSONResponse(httpWriter, &result)

	require.Equal(http.StatusCreated, resp.StatusCode, "Wrong status code in response")
	require.Nil(err, "Response not JSON parsable", err)
	afterSizes := getSize(suite.sr, []string{})
	require.Len(afterSizes, len(beforeSizes)+1, "Same amount of sizes before and after creation")
	createdSizes := getSize(suite.sr, []string{sizeToCreate.ID})
	require.Len(createdSizes, 1, "Size created more than once", string(body))
}

func (suite *SizeTestSuite) TestCreateSizeInvalidPayload() {
	require := suite.Require()
	httpWriter, httpRequest := utils.HttpMock("PUT", "/size", "something")
	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)

	resp, body, err := utils.ParsePlainTextResponse(httpWriter)

	require.Equal(http.StatusInternalServerError, resp.StatusCode, "Wrong status code in response")
	require.Nil(err, "Response not readable", err)
	require.Contains(string(body), `cannot read size from request: json`, "No proper error message in response", string(body))
}

func (suite *SizeTestSuite) TestUpdateSize() {
	require := suite.Require()
	sizeToUpdate := dummySizes[0]
	sizeToUpdate.Description = "Modified Description"
	beforeSizes := getSize(suite.sr, []string{})
	httpWriter, httpRequest := utils.HttpMock("POST", "/size", sizeToUpdate)
	restful.DefaultContainer.ServeHTTP(httpWriter, httpRequest)

	var result *maas.Size
	resp, body, err := utils.ParseJSONResponse(httpWriter, &result)

	require.Equal(http.StatusOK, resp.StatusCode, "Wrong status code in response")
	require.Nil(err, "Response not JSON parsable", err)
	afterSizes := getSize(suite.sr, []string{})
	require.Len(afterSizes, len(beforeSizes), "Different amount of sizes after update")
	updatedSizes := getSize(suite.sr, []string{sizeToUpdate.ID})
	require.Len(updatedSizes, 1, "Updated size found more than once", string(body))
	updatedSize := updatedSizes[0]
	require.Equal(updatedSize.Description, sizeToUpdate.Description, "Field was not updated properly")
}
