package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/rs/zerolog"

	"github.com/jonsabados/go-lambda-demo/correlation"
)

func NewRestAPI(logger zerolog.Logger, correlationIDGenerator correlation.IDGenerator, eventEndpoint http.Handler) http.Handler {
	r := chi.NewRouter()
	// let's use a decent set of base middlewares
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)
	// no canned zerolog based middlewares that I'm aware of, so lets use our own
	r.Use(ZerologLogAttachMiddleware(logger))
	// correlation ids are incredibly useful for debugging things in a distributed env so lets suck in our middleware for that
	r.Use(correlation.Middleware(correlationIDGenerator))
	r.Use(RequestLoggingMiddleware())
	// now for our example endpoint
	r.Post("/event", eventEndpoint.ServeHTTP)
	return r
}

// ZerologLogAttachMiddleware attaches the provided logger to all request contexts
func ZerologLogAttachMiddleware(logger zerolog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ctx := request.Context()
			ctx = logger.WithContext(ctx)
			next.ServeHTTP(writer, request.WithContext(ctx))
		})
	}
}

// RequestLoggingMiddleware logs request/response details. Note, this relies upon a logger being in the context and should be added after a ZerologLogAttachMiddleware
func RequestLoggingMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			ww := middleware.NewWrapResponseWriter(writer, request.ProtoMajor)

			t1 := time.Now()
			defer func() {
				zerolog.Ctx(request.Context()).Info().
					Int("status", ww.Status()).
					Int("bytesWritten", ww.BytesWritten()).
					Dur("duration", time.Since(t1)).
					Msg("request processed")
			}()

			next.ServeHTTP(ww, request)
		})
	}
}
