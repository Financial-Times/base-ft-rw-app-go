package baseftrwapp

import (
	"fmt"
	standardLog "log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/Financial-Times/http-handlers-go/httphandlers"
	"github.com/Financial-Times/service-status-go/gtg"
	status "github.com/Financial-Times/service-status-go/httphandlers"
	graphite "github.com/cyberdelia/go-metrics-graphite"
	"github.com/gorilla/mux"
	metrics "github.com/rcrowley/go-metrics"
	log "github.com/sirupsen/logrus"
)

type RWConf struct {
	Services      map[string]Service
	HealthHandler func(http.ResponseWriter, *http.Request)
	Port          int
	ServiceName   string
	Env           string
	EnableReqLog  bool
}

// RunServer will set up GET, PUT and DELETE endpoints for the specified path,
// calling the appropriate service functions:
// PUT -> Write
// GET -> Read
// DELETE -> Delete
// It will also setup the healthcheck and ping endpoints
// Endpoints are wrapped in a metrics timer and request loggin including transactionID, which is generated
// if not found on the request as X-Request-Id header
func RunServer(services map[string]Service, healthHandler func(http.ResponseWriter, *http.Request), port int, serviceName string, env string) {
	RunServerWithConf(RWConf{
		EnableReqLog:  true,
		Services:      services,
		Env:           env,
		HealthHandler: healthHandler,
		Port:          port,
		ServiceName:   serviceName,
	})
}

func RunServerWithConf(conf RWConf) {
	for path, service := range conf.Services {
		err := service.Initialise()
		if err != nil {
			log.Fatalf("Service for path %s could not startup, err=%s", path, err)
		}
	}

	if conf.Env != "local" {
		f, err := os.OpenFile("/var/log/apps/"+conf.ServiceName+"-go-app.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
		if err == nil {
			log.SetOutput(f)
			log.SetFormatter(&log.TextFormatter{DisableColors: true})
		} else {
			log.Fatalf("Failed to initialise log file, %v", err)
		}
		defer f.Close()
	}

	var m http.Handler
	m = router(conf.Services, conf.HealthHandler)
	if conf.EnableReqLog {
		m = httphandlers.TransactionAwareRequestLoggingHandler(log.StandardLogger(), m)
	}
	m = httphandlers.HTTPMetricsHandler(metrics.DefaultRegistry, m)

	http.Handle("/", m)

	log.Printf("listening on %d", conf.Port)
	log.Println(http.ListenAndServe(fmt.Sprintf(":%d", conf.Port), nil).Error())
	log.Printf("exiting on %s", conf.ServiceName)
}

//Router sets up the Router - extracted for testability
func router(services map[string]Service, healthHandler func(http.ResponseWriter, *http.Request)) *mux.Router {
	m := mux.NewRouter()

	gtgChecker := make([]gtg.StatusChecker, 0)

	for path, service := range services {
		handlers := httpHandlers{service}
		m.HandleFunc(fmt.Sprintf("/%s/__count", path), handlers.countHandler).Methods("GET")
		m.HandleFunc(fmt.Sprintf("/%s/__ids", path), handlers.idsHandler).Methods("GET")
		m.HandleFunc(fmt.Sprintf("/%s/{uuid}", path), handlers.getHandler).Methods("GET")
		m.HandleFunc(fmt.Sprintf("/%s/{uuid}", path), handlers.putHandler).Methods("PUT")
		m.HandleFunc(fmt.Sprintf("/%s/{uuid}", path), handlers.deleteHandler).Methods("DELETE")
		gtgChecker = append(gtgChecker, func() gtg.Status {
			if err := service.Check(); err != nil {
				return gtg.Status{GoodToGo: false, Message: err.Error()}
			}

			return gtg.Status{GoodToGo: true}
		})
	}

	m.HandleFunc("/__health", healthHandler)
	// The top one of these feels more correct, but the lower one matches what we have in Dropwizard,
	// so it's what apps expect currently
	m.HandleFunc(status.PingPath, status.PingHandler)
	m.HandleFunc(status.PingPathDW, status.PingHandler)

	// The top one of these feels more correct, but the lower one matches what we have in Dropwizard,
	// so it's what apps expect currently same as ping, the content of build-info needs more definition
	m.HandleFunc(status.BuildInfoPath, status.BuildInfoHandler)
	m.HandleFunc(status.BuildInfoPathDW, status.BuildInfoHandler)

	m.HandleFunc(status.GTGPath, status.NewGoodToGoHandler(gtg.FailFastParallelCheck(gtgChecker)))

	return m
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
