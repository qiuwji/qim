package actor

// ReceiveFunc 是消息处理函数的类型，用于将 Receive 方法转换为一流函数
type ReceiveFunc func(ctx Context)

// Middleware 是消息处理中间件，接收下一个处理函数并返回包装后的处理函数
type Middleware func(next ReceiveFunc) ReceiveFunc
