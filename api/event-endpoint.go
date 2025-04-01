package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/rs/zerolog"
)

type InboundEventPayload struct {
	Message string `json:"message"`
	Source  string `json:"source"`
}

type EventReceiver interface {
	ReceiveInboundEvent(ctx context.Context, source, message string) error
}

func NewEventReceiverEndpoint(er EventReceiver) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		ctx := request.Context()

		body, err := io.ReadAll(request.Body)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("error reading request body while receiving event")
			DoErrorResponse(ctx, writer)
			return
		}

		var payload InboundEventPayload
		err = json.Unmarshal(body, &payload)
		if err != nil {
			zerolog.Ctx(ctx).Warn().Err(err).Msg("error unmarshalling request")
			DoBadRequestResponse(ctx, []string{fmt.Sprintf("Unable to process request body: %s", err)}, make([]FieldError, 0), writer)
			return
		}

		err = er.ReceiveInboundEvent(ctx, payload.Source, payload.Message)
		if err != nil {
			zerolog.Ctx(ctx).Err(err).Msg("error receiving event")
			DoErrorResponse(ctx, writer)
			return
		}
		DoAcceptedResponse(ctx, "event received and accepted", writer)
	})
}
