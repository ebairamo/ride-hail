package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Ключи контекста
type contextKey string

const (
	RequestIDKey contextKey = "request_id"
	UserIDKey    contextKey = "user_id"
	RoleKey      contextKey = "role"
)

// Logger — основной логгер
type Logger struct {
	service  string
	hostname string
	Slog     *slog.Logger
}

// NewLogger создаёт новый логгер с опциями
func NewLogger(service string, opts LoggerOptions) *Logger {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown-host"
	}

	var handler slog.Handler
	if opts.Pretty {
		handler = NewPrettyJSONHandler(opts.Output, opts.Level)
	} else {
		handler = slog.NewJSONHandler(opts.Output, &slog.HandlerOptions{
			Level: opts.Level,
		})
	}

	return &Logger{
		service:  service,
		hostname: hostname,
		Slog:     slog.New(handler),
	}
}

type LoggerOptions struct {
	Output io.Writer
	Pretty bool
	Level  slog.Level
}

// Func создаёт логгер функции
func (l *Logger) Func(prefix string) *FuncLogger {
	return &FuncLogger{
		service:  l.service,
		prefix:   prefix,
		hostname: l.hostname,
		slog:     l.Slog,
	}
}

// FuncLogger — логгер уровня функции
type FuncLogger struct {
	service  string
	prefix   string
	hostname string
	slog     *slog.Logger
}

func (f *FuncLogger) log(ctx context.Context, level slog.Level, action, message string, fields ...interface{}) {
	if len(fields)%2 != 0 {
		// Логируем предупреждение о нечётном количестве аргументов
		f.slog.Warn("odd number of fields provided to logger",
			slog.String("func", f.prefix),
			slog.Int("fields_count", len(fields)),
		)
		fields = fields[:len(fields)-1] // Удаляем последний элемент
	}

	reqID := GetRequestID(ctx)
	userID := GetUserID(ctx)

	attrs := []slog.Attr{
		slog.String("service", f.service),
		slog.String("func_name", f.prefix),
		slog.String("action", action),
		slog.String("hostname", f.hostname),
	}

	if reqID != "" {
		attrs = append(attrs, slog.String("request_id", reqID))
	}

	if userID != "" {
		attrs = append(attrs, slog.String("user_id", userID))
	}

	// Безопасная итерация по полям
	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			continue
		}
		attrs = append(attrs, slog.Any(key, fields[i+1]))
	}

	f.slog.LogAttrs(ctx, level, message, attrs...)
}

func (f *FuncLogger) Debug(ctx context.Context, action, message string, fields ...interface{}) {
	f.log(ctx, slog.LevelDebug, action, message, fields...)
}

func (f *FuncLogger) Info(ctx context.Context, action, message string, fields ...interface{}) {
	f.log(ctx, slog.LevelInfo, action, message, fields...)
}

func (f *FuncLogger) Warn(ctx context.Context, action, message string, fields ...interface{}) {
	f.log(ctx, slog.LevelWarn, action, message, fields...)
}

func (f *FuncLogger) Error(ctx context.Context, action, message string, fields ...interface{}) {
	f.log(ctx, slog.LevelError, action, message, fields...)
}

// Функции для работы с контекстом
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

func GetRequestID(ctx context.Context) string {
	if v := ctx.Value(RequestIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

func GetUserID(ctx context.Context) string {
	if v := ctx.Value(UserIDKey); v != nil {
		if id, ok := v.(string); ok {
			return id
		}
	}
	return ""
}

func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, RoleKey, role)
}

func GetRole(ctx context.Context) string {
	if v := ctx.Value(RoleKey); v != nil {
		if r, ok := v.(string); ok {
			return r
		}
	}
	return ""
}

////////////////////////////////////////////////////////////////////////////////
// PRETTY JSON HANDLER
////////////////////////////////////////////////////////////////////////////////

type PrettyJSONHandler struct {
	out   io.Writer
	level slog.Leveler
	mu    sync.Mutex // Потокобезопасность
}

func NewPrettyJSONHandler(out io.Writer, level slog.Leveler) *PrettyJSONHandler {
	return &PrettyJSONHandler{out: out, level: level}
}

func (h *PrettyJSONHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return lvl >= h.level.Level()
}

func (h *PrettyJSONHandler) Handle(_ context.Context, r slog.Record) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	data := make(map[string]interface{})
	data["time"] = r.Time.Format(time.RFC3339)
	data["level"] = r.Level.String()
	data["msg"] = r.Message

	r.Attrs(func(a slog.Attr) bool {
		data[a.Key] = a.Value.Any()
		return true
	})

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		// Логируем в stderr, чтобы не потерять ошибку
		_, _ = io.WriteString(os.Stderr, "logger error: "+err.Error()+"\n")
		return err
	}

	_, err := h.out.Write(buf.Bytes())
	return err
}

func (h *PrettyJSONHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *PrettyJSONHandler) WithGroup(name string) slog.Handler {
	return h
}
