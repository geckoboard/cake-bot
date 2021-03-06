package log

import (
	"os"

	gklog "github.com/go-kit/kit/log"
	gklevels "github.com/go-kit/kit/log/deprecated_levels"
)

type LeveledLogger interface {
	Debug(keyvals ...interface{})
	Info(keyvals ...interface{})
	Error(keyvals ...interface{})
	Warn(keyvals ...interface{})
	Crit(keyvals ...interface{})

	With(keyvals ...interface{}) LeveledLogger
}

func New() LeveledLogger {
	l := gklog.NewLogfmtLogger(os.Stdout)
	kitlevels := gklevels.New(
		l,

		// Fudge values so that switching between debug/info levels does not
		// mess with the log justification
		gklevels.DebugValue("dbug"),
		gklevels.ErrorValue("errr"),
	)

	if os.Getenv("SUPPRESS_TIMESTAMP") == "" {
		kitlevels = kitlevels.With("ts", gklog.DefaultTimestampUTC)
	}

	return levels{kitlevels}
}

type levels struct {
	kit gklevels.Levels
}

func (l levels) Debug(keyvals ...interface{}) {
	if err := l.kit.Debug().Log(keyvals...); err != nil {
		panic(err)
	}
}

func (l levels) Info(keyvals ...interface{}) {
	if err := l.kit.Info().Log(keyvals...); err != nil {
		panic(err)
	}
}

func (l levels) Error(keyvals ...interface{}) {
	if err := l.kit.Error().Log(keyvals...); err != nil {
		panic(err)
	}
}
func (l levels) Warn(keyvals ...interface{}) {
	if err := l.kit.Warn().Log(keyvals...); err != nil {
		panic(err)
	}
}
func (l levels) Crit(keyvals ...interface{}) {
	if err := l.kit.Crit().Log(keyvals...); err != nil {
		panic(err)
	}
}

func (l levels) With(keyvals ...interface{}) LeveledLogger {
	return levels{l.kit.With(keyvals...)}
}
