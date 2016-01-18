package baseftrwapp

import (
	"encoding/json"
	"fmt"
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/mux"
)

type httpHandlers struct {
	s Service
}

func (hh *httpHandlers) putHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	w.Header().Add("Content-Type", "application/json")

	dec := json.NewDecoder(req.Body)
	inst, docUUID, err := hh.s.DecodeJSON(dec)
	if err != nil {
		log.Errorf("Error on parse=%v\n", err)
		writeJsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if docUUID != uuid {
		writeJsonError(w, fmt.Sprintf("uuid does not match: '%v' '%v'", docUUID, uuid), http.StatusBadRequest)
		return
	}

	err = hh.s.Write(inst)
	if err != nil {
		log.Errorf("Error on write=%v\n", err)
		writeJsonError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	//Not necessary for a 200 to be returned, but for PUT requests, if don't specify, don't see 200 status logged in request logs
	w.WriteHeader(http.StatusOK)
}

func (hh *httpHandlers) deleteHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)
	uuid := vars["uuid"]

	deleted, err := hh.s.Delete(uuid)

	if err != nil {
		writeJsonError(w, err.Error(), http.StatusServiceUnavailable)
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
		log.Errorf("Error on read=%v\n", err)
		writeJsonError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	if !found {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	enc := json.NewEncoder(w)
	if err := enc.Encode(obj); err != nil {
		log.Errorf("Error on json encoding=%v\n", err)
		writeJsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (hh *httpHandlers) countHandler(w http.ResponseWriter, r *http.Request) {

	count, err := hh.s.Count()

	w.Header().Add("Content-Type", "application/json")

	if err != nil {
		log.Errorf("Error on read=%v\n", err)
		writeJsonError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}

	enc := json.NewEncoder(w)

	if err := enc.Encode(count); err != nil {
		log.Errorf("Error on json encoding=%v\n", err)
		writeJsonError(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
}

func pingHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Fprintf(w, "pong")
}

func writeJsonError(w http.ResponseWriter, errorMsg string, statusCode int) {
	w.WriteHeader(statusCode)
	fmt.Fprintln(w, fmt.Sprintf("{\"message\": \"%s\"}", errorMsg))
}
