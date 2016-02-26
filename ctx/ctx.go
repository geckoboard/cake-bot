package ctx

import (
	"github.com/geckoboard/cake-bot/log"
	"golang.org/x/net/context"
)

type key int

const (
	loggerKey  key = 1
	bugsnagKey key = 2
)

func Logger(ctx context.Context) log.LeveledLogger {
	if v, ok := ctx.Value(loggerKey).(log.LeveledLogger); ok {
		return v
	} else {
		return log.New()
	}
}

func WithLogger(ctx context.Context, l log.LeveledLogger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}
