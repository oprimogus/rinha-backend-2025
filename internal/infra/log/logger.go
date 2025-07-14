package logger

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"time"
)

type ContextKey string

const (
	RequestKey ContextKey = "request_data"
)

type RequestData struct {
	TraceID  string `json:"trace_id"`
	Method   string `json:"method"`
	Path     string `json:"path"`
	ClientIP string `json:"client_ip"`
}

func GetRequestContext(ctx context.Context) *RequestData {
	return ctx.Value(string(RequestKey)).(*RequestData)
}

type ContextualHandler struct {
	out  io.Writer
	opts slog.HandlerOptions
}

func NewContextualHandler(out io.Writer, opts *slog.HandlerOptions) *ContextualHandler {
	if opts == nil {
		opts = &slog.HandlerOptions{}
	}
	return &ContextualHandler{
		out:  out,
		opts: *opts,
	}
}

func (h *ContextualHandler) Handle(ctx context.Context, r slog.Record) error {
	m := make(map[string]any)

	// Adiciona campos padrão
	m["time"] = r.Time.Format(time.RFC3339)
	m["level"] = r.Level.String()
	m["message"] = r.Message

	if reqData, ok := ctx.Value(RequestKey).(*RequestData); ok {
		m["request"] = reqData
	}

	attrs := make(map[string]any)
	r.Attrs(func(a slog.Attr) bool {
		attrs[a.Key] = a.Value.Any()
		return true
	})

	if len(attrs) > 0 {
		m["attributes"] = attrs
	}

	encoded, err := json.Marshal(m)
	if err != nil {
		return err
	}

	_, err = h.out.Write(append(encoded, '\n'))
	return err
}

func (h *ContextualHandler) Enabled(context.Context, slog.Level) bool {
	return true
}

func (h *ContextualHandler) WithAttrs([]slog.Attr) slog.Handler {
	return h
}

func (h *ContextualHandler) WithGroup(string) slog.Handler {
	return h
}

func InitLogger(out io.Writer, level slog.Level) {
	handler := NewContextualHandler(out, &slog.HandlerOptions{
		Level:     level,
		AddSource: true,
	})
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
