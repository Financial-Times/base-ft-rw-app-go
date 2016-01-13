package baseftrwapp

import (
	"fmt"
	standardLog "log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Financial-Times/go-fthealth/v1a"
	"github.com/Financial-Times/http-handlers-go"
	log "github.com/Sirupsen/logrus"
	"github.com/cyberdelia/go-metrics-graphite"
	"github.com/gorilla/mux"
	"github.com/rcrowley/go-metrics"
)

// RunServer will set up GET, PUT and DELETE endpoints for the specified path,
// calling the appropriate service functions:
// PUT -> Write
// GET -> Read
// DELETE -> Delete
// It will also setup the healthcheck and ping endpoints
// Endpoints are wrapped in a metrics timer and request loggin including transactionID, which is generated
// if not found on the request as X-Request-Id header
func RunServer(engs map[string]Service, serviceName string, serviceDescription string, port int) {
	for path, eng := range engs {
		err := eng.Initialise()
		if err != nil {
			log.Fatalf("Eng for path %s could not startup, err=%s", path, err)
		}
	}

	m := mux.NewRouter()
	http.Handle("/", m)

	for path, eng := range engs {
		handlers := httpHandlers{eng}
		m.HandleFunc(fmt.Sprintf("/%s/{uuid}", path), handlers.getHandler).Methods("GET")
		m.HandleFunc(fmt.Sprintf("/%s/{uuid}", path), handlers.putHandler).Methods("PUT")
		m.HandleFunc(fmt.Sprintf("/%s/{uuid}", path), handlers.deleteHandler).Methods("DELETE")
	}

	var checks []v1a.Check

	for _, eng := range engs {
		checks = append(checks, eng.Check())
	}

	m.HandleFunc("/__health", v1a.Handler(serviceName, serviceDescription, checks...))
	// The top one of these feels more correct, but the lower one matches what we have in Dropwizard,
	// so it's what apps expect currently
	m.HandleFunc("/__ping", pingHandler)
	m.HandleFunc("/ping", pingHandler)

	log.Printf("listening on %d", port)
	http.ListenAndServe(fmt.Sprintf(":%d", port),
		httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry,
			httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), m)))

	log.Println("exiting")
}

//OutputMetricsIfRequired will send metrics to Graphite if a non-empty graphiteTCPAddress is passed in, or to the standard log if logMetrics is true.
// Make sure a sensible graphitePrefix that will uniquely identify your service is passed in, e.g. "content.test.people.rw.neo4j.ftaps58938-law1a-eu-t
func OutputMetricsIfRequired(graphiteTCPAddress string, graphitePrefix string, logMetrics bool) {
	if graphiteTCPAddress != "" {
		addr, _ := net.ResolveTCPAddr("tcp", graphiteTCPAddress)
		go graphite.Graphite(metrics.DefaultRegistry, 5*time.Second, graphitePrefix, addr)
	}
	if logMetrics { //useful locally
		//messy use of the 'standard' log package here as this method takes the log struct, not an interface, so can't use logrus.Logger
		go metrics.Log(metrics.DefaultRegistry, 60*time.Second, standardLog.New(os.Stdout, "metrics", standardLog.Lmicroseconds))
	}
}

// Healthcheck defines the information needed to set up a healthcheck
type Healthcheck struct {
	Name        string
	Description string
	Checks      []v1a.Check
	Parallel    bool
}
