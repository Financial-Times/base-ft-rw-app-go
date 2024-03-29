package baseftrwapp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Financial-Times/up-rw-app-api-go/rwapi"
	"github.com/stretchr/testify/assert"
)

const knownUUID = "12345"

var testAPIData = []byte(`swagger: "2.0"`)

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
		{"Success", newRequest("PUT", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID}), http.StatusOK, "", jsonMessage("PUT successful")},
		{"ParseError", newRequest("PUT", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID, failParse: true}), http.StatusBadRequest, "", jsonMessage("TEST failing to DECODE")},
		{"UUIDMisMatch", newRequest("PUT", fmt.Sprintf("/dummies/%s", "99999")), dummyServices(dummyService{uuid: knownUUID}), http.StatusBadRequest, "", jsonMessage("uuid does not match: '12345' '99999'")},
		{"WriteFailed", newRequest("PUT", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID, failWrite: true}), http.StatusServiceUnavailable, "", jsonMessage("TEST failing to WRITE")},
		{"WriteFailedDueToConflict", newRequest("PUT", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID, failConflict: true}), http.StatusConflict, "", jsonMessage("TEST failing to WRITE due to CONFLICT")},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(testAPIData, test.dummyServices, healthHandler).ServeHTTP(rec, test.req)
		assert.Equal(test.statusCode, rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
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
		{"ReadError", newRequest("GET", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID, failRead: true}), http.StatusServiceUnavailable, "", jsonMessage("TEST failing to READ")},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(testAPIData, test.dummyServices, healthHandler).ServeHTTP(rec, test.req)
		assert.Equal(test.statusCode, rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.Equal(test.body, rec.Body.String(), fmt.Sprintf("%s: Wrong body", test.name))
	}
}

func TestDeleteHandler(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		name          string
		req           *http.Request
		dummyServices map[string]Service
		statusCode    int
		contentType   string // Contents of the Content-Type header
		body          string
	}{
		{"Success", newRequest("DELETE", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID}), http.StatusNoContent, "", ""},
		{"NotFound", newRequest("DELETE", fmt.Sprintf("/dummies/%s", "99999")), dummyServices(dummyService{uuid: knownUUID}), http.StatusNotFound, "", ""},
		{"DeleteError", newRequest("DELETE", fmt.Sprintf("/dummies/%s", knownUUID)), dummyServices(dummyService{uuid: knownUUID, failDelete: true}), http.StatusServiceUnavailable, "", jsonMessage("TEST failing to DELETE")},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(testAPIData, test.dummyServices, healthHandler).ServeHTTP(rec, test.req)
		assert.Equal(test.statusCode, rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.Equal(test.body, rec.Body.String(), fmt.Sprintf("%s: Wrong body", test.name))
	}
}

func TestCountHandler(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		name          string
		req           *http.Request
		dummyServices map[string]Service
		statusCode    int
		contentType   string // Contents of the Content-Type header
		body          string
	}{
		{"Success", newRequest("GET", "/dummies/__count"), dummyServices(dummyService{uuid: knownUUID}), http.StatusOK, "", "2\n"},
		{"CountError", newRequest("GET", "/dummies/__count"), dummyServices(dummyService{uuid: knownUUID, failCount: true}), http.StatusServiceUnavailable, "", jsonMessage("TEST failing to COUNT")},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(testAPIData, test.dummyServices, healthHandler).ServeHTTP(rec, test.req)
		assert.Equal(test.statusCode, rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.Equal(test.body, rec.Body.String(), fmt.Sprintf("%s: Wrong body", test.name))
	}
}

func TestGtgHandler(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		name          string
		req           *http.Request
		dummyServices map[string]Service
		statusCode    int
		contentType   string // Contents of the Content-Type header
		body          string
	}{
		{"Success", newRequest("GET", "/__gtg"), dummyServices(dummyService{failCheck: false}), http.StatusOK, "", "OK"},
		{"GTGError", newRequest("GET", "/__gtg"), dummyServices(dummyService{failCheck: true}), http.StatusServiceUnavailable, "", "TEST failing to CHECK"},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(testAPIData, test.dummyServices, healthHandler).ServeHTTP(rec, test.req)
		assert.Equal(test.statusCode, rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.Equal(test.body, rec.Body.String(), fmt.Sprintf("%s: Wrong body", test.name))
	}
}

func TestAPIHandler(t *testing.T) {
	assert := assert.New(t)
	tests := []struct {
		name          string
		req           *http.Request
		dummyServices map[string]Service
		apiData       []byte
		statusCode    int
		contentType   string // Contents of the Content-Type header
		body          string
	}{
		{"Success", newRequest("GET", "/__api"), dummyServices(dummyService{failCheck: false}), testAPIData, http.StatusOK, "", `swagger: "2.0"`},
		{"EmptyAPIArray", newRequest("GET", "/__api"), dummyServices(dummyService{failCheck: false}), []byte(``), http.StatusNotFound, "", "404 page not found"},
		{"NilAPIArray", newRequest("GET", "/__api"), dummyServices(dummyService{failCheck: false}), nil, http.StatusNotFound, "", "404 page not found"},
	}

	for _, test := range tests {
		rec := httptest.NewRecorder()
		router(test.apiData, test.dummyServices, healthHandler).ServeHTTP(rec, test.req)
		assert.Equal(test.statusCode, rec.Code, fmt.Sprintf("%s: Wrong response code, was %d, should be %d", test.name, rec.Code, test.statusCode))
		assert.Equal(test.body, strings.TrimSpace(rec.Body.String()), fmt.Sprintf("%s: Wrong body", test.name))
	}
}

func newRequest(method, url string) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	return req
}

func jsonMessage(msg string) string {
	return fmt.Sprintf("{\"message\": \"%s\"}", msg)
}

func dummyServices(service Service) map[string]Service {
	return map[string]Service{"dummies": service}
}

type dummyService struct {
	uuid         string
	transId      string
	failParse    bool
	failWrite    bool
	failRead     bool
	failDelete   bool
	failCount    bool
	failConflict bool
	failCheck    bool
}

type dummyServiceData struct {
}

func (dS dummyService) Write(thing interface{}, transId string) error {
	if dS.failWrite {
		return errors.New("TEST failing to WRITE")
	}
	if dS.failConflict {
		return rwapi.ConstraintOrTransactionError{Message: "TEST failing to WRITE due to CONFLICT"}
	}
	dS.transId = transId
	return nil
}

func (dS dummyService) Read(uuid string, transId string) (thing interface{}, found bool, err error) {
	if dS.failRead {
		return nil, false, errors.New("TEST failing to READ")
	}
	if uuid == dS.uuid {
		return dummyServiceData{}, true, nil
	}
	dS.transId = transId
	return nil, false, nil
}

func (dS dummyService) Delete(uuid string, transId string) (found bool, err error) {
	if dS.failDelete {
		return false, errors.New("TEST failing to DELETE")
	}
	if uuid == dS.uuid {
		return true, nil
	}
	dS.transId = transId
	return false, nil
}

func (dS dummyService) DecodeJSON(*json.Decoder) (thing interface{}, identity string, err error) {
	if dS.failParse {
		return "", "", errors.New("TEST failing to DECODE")
	}
	return dummyServiceData{}, dS.uuid, nil
}

func (dS dummyService) Count() (int, error) {
	if dS.failCount {
		return 0, errors.New("TEST failing to COUNT")
	}
	return 2, nil
}

func (dS dummyService) Check() error {
	if dS.failCheck {
		return errors.New("TEST failing to CHECK")
	}
	return nil
}

func (dS dummyService) Initialise() error {
	return nil
}

func healthHandler(http.ResponseWriter, *http.Request) {
}
