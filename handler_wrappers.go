package baseftrwapp

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/url"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/rcrowley/go-metrics"
)

// HTTPMetricsHandler records metrics for each request
func HTTPMetricsHandler(h http.Handler) http.Handler {
	return httpMetricsHandler{h}
}

type httpMetricsHandler struct {
	handler http.Handler
}

func (h httpMetricsHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	t := metrics.GetOrRegisterTimer(req.Method, metrics.DefaultRegistry)
	t.Time(func() { h.handler.ServeHTTP(w, req) })
}

// TransactionAwareRequestLoggingHandler finds a transactionID on a request header or generates one, and makes sure it gets logged
func TransactionAwareRequestLoggingHandler(out io.Writer, h http.Handler) http.Handler {
	return transactionAwareRequestLoggingHandler{out, h}
}

type transactionAwareRequestLoggingHandler struct {
	writer  io.Writer
	handler http.Handler
}

func (h transactionAwareRequestLoggingHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	transactionID := GetTransactionIdFromRequest(req)
	t := time.Now()
	logger := makeLogger(w)
	url := *req.URL
	h.handler.ServeHTTP(logger, req)
	writeRequestLog(h.writer, req, transactionID, url, t, logger.Status(), logger.Size())
}

func makeLogger(w http.ResponseWriter) loggingResponseWriter {
	var logger loggingResponseWriter = &responseLogger{w: w}
	if _, ok := w.(http.Hijacker); ok {
		logger = &hijackLogger{responseLogger{w: w}}
	}
	h, ok1 := logger.(http.Hijacker)
	c, ok2 := w.(http.CloseNotifier)
	if ok1 && ok2 {
		return hijackCloseNotifier{logger, h, c}
	}
	if ok2 {
		return &closeNotifyWriter{logger, c}
	}
	return logger
}

type loggingResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	Status() int
	Size() int
}

// responseLogger is wrapper of http.ResponseWriter that keeps track of its HTTP
// status code and body size
type responseLogger struct {
	w      http.ResponseWriter
	status int
	size   int
}

func (l *responseLogger) Header() http.Header {
	return l.w.Header()
}

func (l *responseLogger) Write(b []byte) (int, error) {
	if l.status == 0 {
		// The status will be StatusOK if WriteHeader has not been called yet
		l.status = http.StatusOK
	}
	size, err := l.w.Write(b)
	l.size += size
	return size, err
}

func (l *responseLogger) WriteHeader(s int) {
	l.w.WriteHeader(s)
	l.status = s
}

func (l *responseLogger) Status() int {
	return l.status
}

func (l *responseLogger) Size() int {
	return l.size
}

func (l *responseLogger) Flush() {
	f, ok := l.w.(http.Flusher)
	if ok {
		f.Flush()
	}
}

type hijackLogger struct {
	responseLogger
}

func (l *hijackLogger) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h := l.responseLogger.w.(http.Hijacker)
	conn, rw, err := h.Hijack()
	if err == nil && l.responseLogger.status == 0 {
		// The status will be StatusSwitchingProtocols if there was no error and
		// WriteHeader has not been called yet
		l.responseLogger.status = http.StatusSwitchingProtocols
	}
	return conn, rw, err
}

type closeNotifyWriter struct {
	loggingResponseWriter
	http.CloseNotifier
}

type hijackCloseNotifier struct {
	loggingResponseWriter
	http.Hijacker
	http.CloseNotifier
}

// writeCombinedLog writes a log entry for req to w in Apache Combined Log Format.
// ts is the timestamp with which the entry should be logged.
// status and size are used to provide the response HTTP status and size.
func writeRequestLog(w io.Writer, req *http.Request, transactionID string, url url.URL, ts time.Time, status, size int) {
	username := "-"
	if url.User != nil {
		if name := url.User.Username(); name != "" {
			username = name
		}
	}

	host, _, err := net.SplitHostPort(req.RemoteAddr)

	if err != nil {
		host = req.RemoteAddr
	}

	uri := req.RequestURI

	// Requests using the CONNECT method over HTTP/2.0 must use
	// the authority field (aka r.Host) to identify the target.
	// Refer: https://httpwg.github.io/specs/rfc7540.html#CONNECT
	if req.ProtoMajor == 2 && req.Method == "CONNECT" {
		uri = req.Host
	}
	if uri == "" {
		uri = url.RequestURI()
	}

	log.WithFields(log.Fields{
		"time":           ts,
		"host":           host,
		"username":       username,
		"method":         req.Method,
		"transaction_id": transactionID,
		"uri":            uri,
		"protocol":       req.Proto,
		"status":         status,
		"size":           size,
		"referer":        req.Referer(),
		"userAgent":      req.UserAgent(),
	}).Info("")

}
