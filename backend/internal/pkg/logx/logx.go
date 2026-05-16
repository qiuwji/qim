package logx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

const (
	LogIDKey    = "log_id"
	LogIDHeader = "X-Log-ID"
)

type contextKey struct{}

var logSeq uint64

func Init() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(err)
	}
	zap.ReplaceGlobals(logger)
}

func NewLogID() string {
	now := time.Now()
	seq := atomic.AddUint64(&logSeq, 1) % 1000000
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%s-%06d", now.Format("20060102150405000"), seq)
	}
	return fmt.Sprintf("%s-%06d-%s", now.Format("20060102150405000"), seq, hex.EncodeToString(b[:]))
}

func WithLogID(ctx context.Context, logID string) context.Context {
	if logID == "" {
		return ctx
	}
	return context.WithValue(ctx, contextKey{}, logID)
}

func LogIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	logID, _ := ctx.Value(contextKey{}).(string)
	return logID
}

func FromContext(ctx context.Context) *zap.Logger {
	logID := LogIDFromContext(ctx)
	return WithLogIDField(logID)
}

func WithLogIDField(logID string) *zap.Logger {
	if logID == "" {
		return zap.L()
	}
	return zap.L().With(zap.String(LogIDKey, logID))
}

type ActorLogger struct{}

func NewActorLogger() *ActorLogger {
	return &ActorLogger{}
}

func (*ActorLogger) Printf(format string, args ...any) {
	zap.L().Error(fmt.Sprintf(format, args...))
}
