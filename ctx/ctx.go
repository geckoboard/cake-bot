package ctx

import (
	"context"

	"github.com/geckoboard/cake-bot/log"
)

type key int

const (
	loggerKey  key = 1
	bugsnagKey key = 2
)

func Logger(ctx context.Context) log.LeveledLogger {
	if v := ctx.Value(loggerKey); v != nil {
		if l, ok := v.(log.LeveledLogger); ok {
			return l
		}
	}

	return log.New()
}

func WithLogger(ctx context.Context, l log.LeveledLogger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}
