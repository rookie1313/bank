package gapi

import (
	"context"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"time"
)

func GRPCLogger(ctx context.Context,
	req any,
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler) (resp any, err error) {
	startTime := time.Now()
	result, err := handler(ctx, req)
	duration := time.Since(startTime)
	statusCode := codes.Unknown
	if st, ok := status.FromError(err); ok {
		statusCode = st.Code()
	}

	logger := log.Info()
	if err != nil {
		logger = log.Error().Err(err)
	}

	logger.
		Dur("duration", duration).
		Int("status_code", int(statusCode)).
		Str("status_text", statusCode.String()).
		Str("protocol", "grpc").
		Str("method", info.FullMethod).
		Msg("received a gRPC request")
	return result, err
}

type ResponseRecorder struct {
	http.ResponseWriter
	statusCode int
	Body       []byte
}

func (receiver *ResponseRecorder) WriteHeader(statusCode int) {
	receiver.statusCode = statusCode
	receiver.ResponseWriter.WriteHeader(statusCode)
}

func (receiver *ResponseRecorder) Write(body []byte) (int, error) {
	receiver.Body = body
	return receiver.ResponseWriter.Write(body)
}

func HttpLogger(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		recorder := &ResponseRecorder{ResponseWriter: writer, statusCode: http.StatusOK}
		startTime := time.Now()
		handler.ServeHTTP(recorder, request)
		duration := time.Since(startTime)

		logger := log.Info()
		if recorder.statusCode != http.StatusOK {
			logger = log.Error().Bytes("body", recorder.Body)
		}

		logger.
			Dur("duration", duration).
			Int("status_code", recorder.statusCode).
			Str("status_text", http.StatusText(recorder.statusCode)).
			Str("protocol", "http").
			Str("method", request.Method).
			Str("path", request.RequestURI).
			Msg("received a HTTP request")
	})
}
