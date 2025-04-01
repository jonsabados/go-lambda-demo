package correlation

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rs/zerolog"
)

const Header = "x-correlation-id"

type correlationIDKeyType string

const correlationIDKey = correlationIDKeyType("correlationID")

type IDGenerator func() string

func Middleware(idGenerator IDGenerator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			inboundCorrelationID := request.Header.Get(Header)
			requestID := idGenerator()
			if inboundCorrelationID != "" {
				requestID = fmt.Sprintf("%s:%s", inboundCorrelationID, requestID)
			}
			writer.Header().Add(Header, requestID)
			ctx := request.Context()
			ctx = WithContext(ctx, requestID)
			request = request.WithContext(ctx)
			next.ServeHTTP(writer, request)
		})
	}
}

func FromContext(ctx context.Context) string {
	if value, ok := ctx.Value(correlationIDKey).(string); ok {
		return value
	}
	return ""
}

func WithContext(ctx context.Context, correlationID string) context.Context {
	logger := zerolog.Ctx(ctx).With().Str("correlationID", correlationID).Logger()
	ctx = logger.WithContext(ctx)
	return context.WithValue(ctx, correlationIDKey, correlationID)
}
