package apperr

import (
	"errors"
	"testing"
)

func TestErrorAndPayload_BitsUT(t *testing.T) {
	t.Run("nil error 返回 ok payload", func(t *testing.T) {
		payload := ToPayload(nil)
		if payload.Code != CodeOK {
			t.Fatalf("code = %s, want %s", payload.Code, CodeOK)
		}
	})

	t.Run("应用错误保留业务码和消息", func(t *testing.T) {
		cause := errors.New("db down")
		err := Wrap(CodeBadRequest, "bad input", cause)
		payload := ToPayload(err)
		if payload.Code != CodeBadRequest || payload.Message != "bad input" {
			t.Fatalf("payload = %+v", payload)
		}
		if !errors.Is(err, cause) {
			t.Fatalf("wrapped error should unwrap cause")
		}
		if !Is(err) {
			t.Fatalf("Is should recognize apperr")
		}
	})

	t.Run("普通错误转换为 internal", func(t *testing.T) {
		payload := ToPayload(errors.New("boom"))
		if payload.Code != CodeInternal || payload.Message != "boom" {
			t.Fatalf("payload = %+v", payload)
		}
	})

	t.Run("Error nil receiver 和空消息分支", func(t *testing.T) {
		var appErr *Error
		if appErr.Error() != "" || appErr.Unwrap() != nil {
			t.Fatalf("nil receiver should be safe")
		}
		err := &Error{Code: CodeUnauthorized}
		if err.Error() != string(CodeUnauthorized) {
			t.Fatalf("error = %q", err.Error())
		}
		err.Cause = errors.New("cause")
		if err.Error() != "cause" {
			t.Fatalf("error = %q", err.Error())
		}
	})
}
