package actor

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrActorExists    = errors.New("actor already exists")
	ErrEngineShutdown = errors.New("engine is shut down")
	ErrReplyNoAsk     = errors.New("Reply called on a non-Ask message")
	ErrReplyDuplicate = errors.New("Reply already called for current message")
	ErrNoReply        = errors.New("actor did not reply to Ask message")
)

type ActorStoppedError struct {
	Name string
}

func (e *ActorStoppedError) Error() string {
	return "actor " + e.Name + " is stopped"
}

type MailboxFullError struct {
	Name string
}

func (e *MailboxFullError) Error() string {
	return "actor " + e.Name + " mailbox is full"
}

type MailboxRejectedError struct {
	Name   string
	Reason MailboxDropReason
}

func (e *MailboxRejectedError) Error() string {
	return "actor " + e.Name + " mailbox rejected message: " + string(e.Reason)
}

type AskTimeoutError struct {
	Timeout time.Duration
}

func (e *AskTimeoutError) Error() string {
	if e.Timeout == 0 {
		return "ask timeout: immediate mode"
	}
	return "ask timeout after " + e.Timeout.String()
}

type ShutdownTimeoutError struct {
	RunningCount int
}

func (e *ShutdownTimeoutError) Error() string {
	return fmt.Sprintf("engine shutdown timed out, %d actors still running", e.RunningCount)
}
