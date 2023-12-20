package templates

import "context"

// Logger is a logger interface for this package.
type Logger interface {
	DebugCtx(ctx context.Context, msg string, fields map[string]any)
	ErrorCtx(ctx context.Context, msg string, err error, fields map[string]any)
	Debug(msg string, fields map[string]any)
}

type noopLogger struct{}

func (l *noopLogger) DebugCtx(ctx context.Context, msg string, fields map[string]any)            {}
func (l *noopLogger) ErrorCtx(ctx context.Context, msg string, err error, fields map[string]any) {}
func (l *noopLogger) Debug(msg string, fields map[string]any)                                    {}
