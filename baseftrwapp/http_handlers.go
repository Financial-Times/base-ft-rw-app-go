package baseftrwapp

import (
	"encoding/json"
	"fmt"
	"net/http"

	"compress/gzip"

	"io"

	"github.com/Financial-Times/transactionid-utils-go"
	"github.com/Financial-Times/up-rw-app-api-go/rwapi"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
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
			writeJSONMessage(w, err.Error(), http.StatusBadRequest)
			return
		}
		defer unzipped.Close()
		body = unzipped
	}

	dec := json.NewDecoder(body)
	inst, docUUID, err := hh.s.DecodeJSON(dec)

	if err != nil {
		writeJSONMessage(w, err.Error(), http.StatusBadRequest)
		return
	}

	if docUUID != uuid {
		writeJSONMessage(w, fmt.Sprintf("uuid does not match: '%v' '%v'", docUUID, uuid), http.StatusBadRequest)
		return
	}
	tid := transactionidutils.GetTransactionIDFromRequest(req)

	err = hh.s.Write(inst, tid)

	if err != nil {
		switch e := err.(type) {
		case noContentReturnedError:
			writeJSONMessage(w, e.NoContentReturnedDetails(), http.StatusNoContent)
			return
		case rwapi.ConstraintOrTransactionError:
			writeJSONMessage(w, e.Error(), http.StatusConflict)
			return
		case invalidRequestError:
			writeJSONMessage(w, e.InvalidRequestDetails(), http.StatusBadRequest)
			return
		default:
			writeJSONMessage(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
	}

	w.Header().Set("X-Request-Id", tid)
	writeJSONMessage(w, "PUT successful", http.StatusOK)
}

func (hh *httpHandlers) deleteHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	tid := transactionidutils.GetTransactionIDFromRequest(req)
	deleted, err := hh.s.Delete(uuid, tid)

	if err != nil {
		writeJSONMessage(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	w.Header().Set("X-Request-Id", tid)

	if deleted {
		w.WriteHeader(http.StatusNoContent)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func (hh *httpHandlers) getHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]
	tid := transactionidutils.GetTransactionIDFromRequest(req)

	obj, found, err := hh.s.Read(uuid, tid)

	w.Header().Add("Content-Type", "application/json")
	w.Header().Set("X-Request-Id", tid)

	if err != nil {
		writeJSONMessage(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(obj); err != nil {
		writeJSONMessage(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (hh *httpHandlers) countHandler(w http.ResponseWriter, r *http.Request) {

	count, err := hh.s.Count()

	w.Header().Add("Content-Type", "application/json")

	if err != nil {
		writeJSONMessage(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	enc := json.NewEncoder(w)

	if err := enc.Encode(count); err != nil {
		writeJSONMessage(w, err.Error(), http.StatusServiceUnavailable)
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

func writeJSONMessage(w http.ResponseWriter, msg string, statusCode int) {
	w.WriteHeader(statusCode)
	fmt.Fprint(w, fmt.Sprintf("{\"message\": \"%s\"}", msg))
}
