package skop

import (
	"context"

	"github.com/go-kit/kit/log"
)

type loggerKey int

const contextLoggerKey = loggerKey(0)

func Logger(ctx context.Context) log.Logger {
	logger, _ := ctx.Value(contextLoggerKey).(log.Logger)
	return logger
}

func ContextWithLogger(ctx context.Context, logger log.Logger) context.Context {
	return context.WithValue(ctx, contextLoggerKey, logger)
}
