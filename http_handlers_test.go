package baseftrwapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/stretchr/testify/assert"
)

const knownUUID = "12345"

func TestPutHandler(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		name          string
		req           *http.Request
		dummyServices map[string]Service
		statusCode    int
		contentType   string // Contents of the Content-Type header
		body          string
	}{
		{"Success", newRequest("PUT", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID}), http.StatusOK, "", ""},
		{"ParseError", newRequest("PUT", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID, failParse: true}), http.StatusBadRequest, "", errorMessage("TEST failing to DECODE")},
		{"UUIDMisMatch", newRequest("PUT", fmt.Sprintf("/dummies/%s", "99999")), dummyServices(dummyService{uuid: knownUUID}), http.StatusBadRequest, "", errorMessage("uuid does not match: '12345' '99999'")},
		{"WriteFailed", newRequest("PUT", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID, failWrite: true}), http.StatusServiceUnavailable, "", errorMessage("TEST failing to WRITE")},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(test.dummyServices, test.name, test.name).ServeHTTP(rec, test.req)
		assert.True(test.statusCode == rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.Equal(test.body, rec.Body.String(), fmt.Sprintf("%s: Wrong body", test.name))
	}
}

func TestGetHandler(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		name          string
		req           *http.Request
		dummyServices map[string]Service
		statusCode    int
		contentType   string // Contents of the Content-Type header
		body          string
	}{
		{"Success", newRequest("GET", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID}), http.StatusOK, "", "{}\n"},
		{"NotFound", newRequest("GET", fmt.Sprintf("/dummies/%s", "99999")), dummyServices(dummyService{uuid: knownUUID}), http.StatusNotFound, "", ""},
		{"ReadError", newRequest("GET", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID, failRead: true}), http.StatusServiceUnavailable, "", errorMessage("TEST failing to READ")},
		//{"EncodeError", newRequest("GET", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID}), http.StatusServiceUnavailable, "", errorMessage("TEST failing to WRITE")},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(test.dummyServices, test.name, test.name).ServeHTTP(rec, test.req)
		assert.True(test.statusCode == rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.Equal(test.body, rec.Body.String(), fmt.Sprintf("%s: Wrong body", test.name))
	}
}

func newRequest(method, url string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	return req
}

func errorMessage(errMsg string) string {
	return fmt.Sprintf("{\"message\": \"%s\"}\n", errMsg)
}

func dummyServices(service Service) map[string]Service {
	return map[string]Service{"dummies": service}
}

type dummyService struct {
	uuid       string
	failParse  bool
	failWrite  bool
	failRead   bool
	failDelete bool
}

type dummyServiceData struct {
}

func (dS dummyService) Write(thing interface{}) error {
	if dS.failWrite {
		return errors.New("TEST failing to WRITE")
	}
	return nil
}

func (dS dummyService) Read(uuid string) (thing interface{}, found bool, err error) {
	if dS.failRead {
		return nil, false, errors.New("TEST failing to READ")
	}
	if uuid == dS.uuid {
		return dummyServiceData{}, true, nil
	}
	return nil, false, nil
}

func (dS dummyService) Delete(uuid string) (found bool, err error) {
	return true, nil
}

func (dS dummyService) DecodeJSON(*json.Decoder) (thing interface{}, identity string, err error) {
	if dS.failParse {
		return "", "", errors.New("TEST failing to DECODE")
	}
	return dummyServiceData{}, dS.uuid, nil
}

func (dS dummyService) Count() (int, error) {
	return 0, nil
}

func (dS dummyService) Check() (check v1a.Check) {
	return v1a.Check{}
}

func (dS dummyService) Initialise() error {
	return nil
}
