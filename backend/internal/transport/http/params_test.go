package http

import (
	"errors"
	nethttp "net/http"
	"net/http/httptest"
	"testing"

	"qim/internal/pkg/apperr"

	"github.com/gin-gonic/gin"
)

func TestParamsAndErrors_BitsUT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("paramUint 成功和失败分支", func(t *testing.T) {
		router := gin.New()
		router.GET("/items/:id", func(c *gin.Context) {
			id, ok := paramUint(c, "id")
			if !ok {
				return
			}
			c.JSON(nethttp.StatusOK, gin.H{"id": id, "q": queryUint(c, "q"), "seq": queryInt64(c, "seq")})
		})

		w := httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(nethttp.MethodGet, "/items/10?q=2&seq=-1", nil))
		if w.Code != nethttp.StatusOK {
			t.Fatalf("status = %d body = %s", w.Code, w.Body.String())
		}

		w = httptest.NewRecorder()
		router.ServeHTTP(w, httptest.NewRequest(nethttp.MethodGet, "/items/0", nil))
		if w.Code != nethttp.StatusOK {
			t.Fatalf("status = %d", w.Code)
		}
	})

	t.Run("badRequest 和 internalError 支持 nil 与 wrap", func(t *testing.T) {
		if payload := apperr.ToPayload(badRequest(nil)); payload.Code != apperr.CodeBadRequest {
			t.Fatalf("badRequest nil payload = %+v", payload)
		}
		if payload := apperr.ToPayload(internalError(nil)); payload.Code != apperr.CodeInternal {
			t.Fatalf("internalError nil payload = %+v", payload)
		}
		cause := errors.New("boom")
		if !errors.Is(badRequest(cause), cause) {
			t.Fatalf("badRequest should wrap cause")
		}
		if !errors.Is(internalError(cause), cause) {
			t.Fatalf("internalError should wrap cause")
		}
	})
}
