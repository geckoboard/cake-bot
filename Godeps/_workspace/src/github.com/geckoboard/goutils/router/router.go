package router

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/julienschmidt/httprouter"
)

// App Handler
type AppHandler func(http.ResponseWriter, *http.Request, httprouter.Params)

// App Router logging requests
type AppRouter struct {
	httprouter.Router
}

func New() *AppRouter {
	panicHandler := func(w http.ResponseWriter, _ *http.Request, _ interface{}) {
		w.WriteHeader(500)
	}

	return &AppRouter{
		Router: httprouter.Router{
			RedirectTrailingSlash: true,
			RedirectFixedPath:     true,
			PanicHandler:          panicHandler,
		},
	}
}

func (router *AppRouter) Handler(handler AppHandler) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		handler(w, r, p)
	}
}

func (router *AppRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Process request
	tw := &AppResponseWriter{w, -1, 0}
	router.Router.ServeHTTP(tw, r)

	// Stop timer
	latency := time.Since(start)

	log.Printf("| %3d | %12v | %s | %s %-7s\n",
		tw.StatusCode, latency, ClientIP(r),
		r.Method, r.URL.Path,
	)
}

func ClientIP(r *http.Request) string {
	clientIP := r.Header.Get("X-Real-IP")
	if len(clientIP) == 0 {
		clientIP = r.Header.Get("X-Forwarded-For")
	}
	if len(clientIP) == 0 {
		clientIP = r.RemoteAddr
	}
	return clientIP
}

// Logging Response Writer that saves http status codes
type AppResponseWriter struct {
	http.ResponseWriter
	StatusCode int
	size       int
}

func (rw *AppResponseWriter) Written() bool {
	return rw.StatusCode != -1
}

func (rw *AppResponseWriter) WriteHeader(status int) {
	rw.StatusCode = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *AppResponseWriter) Write(b []byte) (int, error) {
	if !rw.Written() {
		rw.WriteHeader(http.StatusOK)
	}
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// Logs are printed to STDOUT.
func setLogger() {
	log.SetOutput(os.Stdout)
}

func init() {
	setLogger()
}
