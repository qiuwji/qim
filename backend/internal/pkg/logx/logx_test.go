package logx

import (
	"context"
	"strings"
	"testing"
)

func TestLogIDContext_BitsUT(t *testing.T) {
	t.Run("NewLogID 包含时间强相关前缀和序号", func(t *testing.T) {
		id := NewLogID()
		parts := strings.Split(id, "-")
		if len(parts) < 2 {
			t.Fatalf("log id format = %q", id)
		}
		if len(parts[0]) != len("20060102150405000") {
			t.Fatalf("timestamp prefix = %q", parts[0])
		}
	})

	t.Run("context 写入和空值保护", func(t *testing.T) {
		if got := LogIDFromContext(nil); got != "" {
			t.Fatalf("nil context log id = %q", got)
		}
		base := context.Background()
		if WithLogID(base, "") != base {
			t.Fatalf("empty log id should return original context")
		}
		ctx := WithLogID(base, "log-1")
		if got := LogIDFromContext(ctx); got != "log-1" {
			t.Fatalf("log id = %q", got)
		}
	})

	t.Run("logger helpers 不 panic", func(t *testing.T) {
		Init()
		WithLogIDField("")
		FromContext(WithLogID(context.Background(), "log-2"))
		NewActorLogger().Printf("hello %s", "world")
	})
}
