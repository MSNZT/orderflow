package httpmw

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

const unmatchedRoute = "unmatched"

type RequestMetricsRecorder interface {
	RequestStarted()
	RequestFinished(method string, route string, status int, duration time.Duration)
}

func RequestMetrics(recorder RequestMetricsRecorder) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			writer := newResponseWriter(w)

			recorder.RequestStarted()

			defer func() {
				route := chi.RouteContext(r.Context()).RoutePattern()

				if route == "" {
					route = unmatchedRoute
				}

				recorder.RequestFinished(r.Method, route, writer.statusCode, time.Since(start))

			}()

			next.ServeHTTP(writer, r)
		})
	}
}
