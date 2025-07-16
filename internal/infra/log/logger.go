package logger

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/httplog/v3"
	"github.com/google/uuid"
)

type ContextKey string

const RequestKey ContextKey = "request_data"

type RequestData struct {
	TraceID  string `json:"trace_id"`
	Method   string `json:"method"`
	Path     string `json:"path"`
	ClientIP string `json:"client_ip"`
}

func GetRequestContext(ctx context.Context) *RequestData {
	if v, ok := ctx.Value(RequestKey).(*RequestData); ok {
		return v
	}
	return nil
}

// Custom handler que injeta RequestData do contexto
type ContextHandler struct {
	slog.Handler
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	if req := GetRequestContext(ctx); req != nil {
		r.AddAttrs(slog.Group("request",
			slog.String("trace_id", req.TraceID),
			slog.String("method", req.Method),
			slog.String("path", req.Path),
			slog.String("client_ip", req.ClientIP),
		))
	}
	return h.Handler.Handle(ctx, r)
}

type loggingResponseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *loggingResponseWriter) WriteHeader(statusCode int) {
	w.status = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *loggingResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}


func InitLogger(out io.Writer) {
	opts := &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		ReplaceAttr: httplog.SchemaECS.Concise(true).ReplaceAttr,
	}
	base := slog.NewJSONHandler(out, opts)
	slog.SetDefault(slog.New(&ContextHandler{Handler: base}))
}

func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		traceID := uuid.New().String()

		reqData := &RequestData{
			TraceID:  traceID,
			Method:   r.Method,
			Path:     r.URL.Path,
			ClientIP: r.RemoteAddr,
		}

		ctx := context.WithValue(r.Context(), RequestKey, reqData)
		lrw := &loggingResponseWriter{ResponseWriter: w}
		nr := r.WithContext(ctx)

		next.ServeHTTP(lrw, nr)

		duration := time.Since(start)

		slog.Info("request handled",
			slog.Group("request",
				slog.String("trace_id", reqData.TraceID),
				slog.String("method", reqData.Method),
				slog.String("path", reqData.Path),
				slog.String("client_ip", reqData.ClientIP),
				slog.Int("status", lrw.status),
				slog.Int("size", lrw.size),
				slog.String("duration", duration.String()),
			),
		)
	})
}
