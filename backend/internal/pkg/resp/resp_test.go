package resp

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"qim/internal/pkg/apperr"

	"github.com/gin-gonic/gin"
)

func TestOKAndFail_BitsUT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("OK 不在响应体返回 log_id", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		OK(c, map[string]string{"hello": "world"})

		var body Response
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if body.Code != apperr.CodeOK {
			t.Fatalf("code = %s", body.Code)
		}
		if _, ok := mapFromJSON(t, w.Body.Bytes())["log_id"]; ok {
			t.Fatalf("response body should not contain log_id: %s", w.Body.String())
		}
	})

	t.Run("Fail 使用应用错误 payload", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		Fail(c, apperr.New(apperr.CodeUnauthorized, "missing token"))

		var body Response
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if body.Code != apperr.CodeUnauthorized || body.Message != "missing token" {
			t.Fatalf("body = %+v", body)
		}
	})

	t.Run("Fail 普通错误转换 internal", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
		Fail(c, errors.New("boom"))

		var body Response
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if body.Code != apperr.CodeInternal || body.Message != "boom" {
			t.Fatalf("body = %+v", body)
		}
	})
}

func mapFromJSON(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal map: %v", err)
	}
	return m
}
