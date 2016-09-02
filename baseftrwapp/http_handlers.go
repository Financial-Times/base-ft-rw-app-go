package baseftrwapp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"compress/gzip"

	"io"

	"github.com/Financial-Times/neo-utils-go/neoutils"
	"github.com/Financial-Times/up-rw-app-api-go/rwapi"
	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

type httpHandlers struct {
	s rwapi.Service
}

func (hh *httpHandlers) putHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	w.Header().Add("Content-Type", "application/json")

	var body io.Reader = req.Body
	if req.Header.Get("Content-Encoding") == "gzip" {
		unzipped, err := gzip.NewReader(req.Body)
		if err != nil {
			writeJSONError(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer unzipped.Close()
		body = unzipped
	}

	dec := json.NewDecoder(body)
	inst, docUUID, err := hh.s.DecodeJSON(dec)

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if docUUID != uuid {
		writeJSONError(w, fmt.Sprintf("uuid does not match: '%v' '%v'", docUUID, uuid), http.StatusBadRequest)
		return
	}

	err = hh.s.Write(inst)

	if err != nil {
		switch e := err.(type) {
		case noContentReturnedError:
			writeJSONError(w, e.NoContentReturnedDetails(), http.StatusNoContent)
			return
		case *neoutils.ConstraintViolationError:
			// TODO: remove neo specific error check once all apps are
			// updated to use neoutils.Connect() because that maps errors
			// to rwapi.ConstraintOrTransactionError
			writeJSONError(w, e.Error(), http.StatusConflict)
			return
		case rwapi.ConstraintOrTransactionError:
			writeJSONError(w, e.Error(), http.StatusConflict)
			return 
		case invalidRequestError:
			writeJSONError(w, e.InvalidRequestDetails(), http.StatusBadRequest)
			return
		default:
			writeJSONError(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}
	//Not necessary for a 200 to be returned, but for PUT requests, if don't specify, don't see 200 status logged in request logs
	w.WriteHeader(http.StatusOK)
}

func (hh *httpHandlers) deleteHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	deleted, err := hh.s.Delete(uuid)

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	if deleted {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (hh *httpHandlers) getHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	obj, found, err := hh.s.Read(uuid)

	w.Header().Add("Content-Type", "application/json")

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(obj); err != nil {
		writeJSONError(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (hh *httpHandlers) countHandler(w http.ResponseWriter, r *http.Request) {

	count, err := hh.s.Count()

	w.Header().Add("Content-Type", "application/json")

	if err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	enc := json.NewEncoder(w)

	if err := enc.Encode(count); err != nil {
		writeJSONError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
}

func (hh *httpHandlers) idsHandler(w http.ResponseWriter, r *http.Request) {

	idService, ok := hh.s.(rwapi.IDService)
	if !ok {
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	w.Header().Add("Content-Type", "application/json")

	enc := json.NewEncoder(w)

	err := idService.IDs(func(id rwapi.IDEntry) (bool, error) {
		if err := enc.Encode(id); err != nil {
			return false, err
		}
		return true, nil
	})

	if err != nil {
		log.Errorf(err.Error())

		// at this point, the best we can do is close the connection to inform
		// the client of the error, because we've already said "200"
		conn, _, err := w.(http.Hijacker).Hijack()
		if err != nil {
			// there is very little we can do here. Since we're in an http
			// handler, panic is okay. It will be recovered from and will
			// not crash the application, but at least we will capture the
			// cause.
			log.Panicf(err.Error())
		}
		conn.Close()
	}

	return
}

func pingHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "pong")
}

// buildInfoHandler - This is a stop gap and will be added to when we can define what we should display here
func buildInfoHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "build-info")
}

func writeJSONError(w http.ResponseWriter, errorMsg string, statusCode int) {
	w.WriteHeader(statusCode)
	fmt.Fprintln(w, fmt.Sprintf("{\"message\": \"%s\"}", errorMsg))
}
