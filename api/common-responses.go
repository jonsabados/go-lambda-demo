package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/jonsabados/go-lambda-demo/correlation"
)

type ErrorResponse struct {
	Message       string `json:"message"`
	CorrelationID string `json:"correlationId"`
}

func DoErrorResponse(ctx context.Context, writer http.ResponseWriter) {
	writer.Header().Add("content-type", "application/json")
	writer.WriteHeader(http.StatusInternalServerError)
	bytes, err := json.Marshal(ErrorResponse{
		Message:       "An unexpected error has been encountered. Please reference the included correlation id in any support inquires.",
		CorrelationID: correlation.FromContext(ctx),
	})
	if err != nil {
		panic(fmt.Errorf("error marshalling ErrorResponse, this should not happen: %w", err))
	}
	_, _ = writer.Write(bytes)
}

type FieldError struct {
	Field         string `json:"field"`
	Error         string `json:"error"`
	CorrelationID string `json:"correlationId"`
}

type BadRequestResponse struct {
	Errors        []string     `json:"errors"`
	FieldErrors   []FieldError `json:"fieldErrors"`
	CorrelationID string       `json:"correlationId"`
}

func DoBadRequestResponse(ctx context.Context, errors []string, fieldErrors []FieldError, writer http.ResponseWriter) {
	writer.Header().Add("content-type", "application/json")
	writer.WriteHeader(http.StatusBadRequest)
	bytes, err := json.Marshal(BadRequestResponse{
		Errors:        errors,
		FieldErrors:   fieldErrors,
		CorrelationID: correlation.FromContext(ctx),
	})
	if err != nil {
		panic(fmt.Errorf("error marshalling BadRequestResponse, this should not happen: %w", err))
	}
	_, _ = writer.Write(bytes)
}

type AcceptedResponse struct {
	Response      interface{} `json:"response"`
	CorrelationID string      `json:"correlationId"`
}

func DoAcceptedResponse(ctx context.Context, Response interface{}, writer http.ResponseWriter) {
	writer.Header().Add("content-type", "application/json")
	writer.WriteHeader(http.StatusAccepted)
	bytes, err := json.Marshal(AcceptedResponse{
		Response:      Response,
		CorrelationID: correlation.FromContext(ctx),
	})
	if err != nil {
		panic(fmt.Errorf("error marshalling AcceptedResponse, this should not happen: %w", err))
	}
	_, _ = writer.Write(bytes)
}
