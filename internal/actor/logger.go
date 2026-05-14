package actor

import "fmt"

// Logger 定义日志输出接口
type Logger interface {
	Printf(format string, args ...any)
}

type defaultLogger struct{}

func (*defaultLogger) Printf(format string, args ...any) {
	fmt.Printf(format+"\n", args...)
}
