package ctx

import (
	"os"

	"golang.org/x/net/context"
	"gopkg.in/inconshreveable/log15.v2"
)

type key int

const (
	loggerKey key = 1
)

func Logger(ctx context.Context) log15.Logger {
	if v, ok := ctx.Value(loggerKey).(log15.Logger); ok {
		return v
	} else {
		log := log15.New()
		log.SetHandler(log15.MultiHandler(
			log15.StreamHandler(os.Stdout, log15.LogfmtFormat()),
		))
		return log
	}
}

func WithLogger(ctx context.Context, l log15.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}
