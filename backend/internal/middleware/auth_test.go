package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	jwtpkg "qim/internal/pkg/jwt"
	"qim/internal/pkg/logx"

	"github.com/gin-gonic/gin"
)

func TestAuth_BitsUT(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager := jwtpkg.NewManager("secret", time.Hour)
	token, err := manager.Generate(1001, "alice")
	if err != nil {
		t.Fatalf("Generate token: %v", err)
	}

	t.Run("Authorization Bearer 鉴权成功并注入用户信息", func(t *testing.T) {
		router := gin.New()
		router.Use(Auth(manager))
		router.GET("/secure", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"uid": c.GetUint64("uid"), "username": c.GetString("username")})
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/secure", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
		}
		var body map[string]any
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("unmarshal body: %v", err)
		}
		if body["uid"].(float64) != 1001 || body["username"] != "alice" {
			t.Fatalf("body = %+v", body)
		}
	})

	t.Run("query token 鉴权成功", func(t *testing.T) {
		router := gin.New()
		router.Use(Auth(manager))
		router.GET("/secure", func(c *gin.Context) { c.Status(http.StatusNoContent) })

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/secure?token="+token, nil)
		router.ServeHTTP(w, req)
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
		}
	})

	t.Run("缺失或错误 token 会中止请求", func(t *testing.T) {
		for _, header := range []string{"", "bad", "Bearer bad.token"} {
			router := gin.New()
			router.Use(Auth(manager))
			router.GET("/secure", func(c *gin.Context) { t.Fatalf("handler should not run") })

			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/secure", nil)
			if header != "" {
				req.Header.Set("Authorization", header)
			}
			router.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d", w.Code)
			}
			if !jsonContainsCode(w.Body.Bytes(), "unauthorized") {
				t.Fatalf("body should contain unauthorized: %s", w.Body.String())
			}
		}
	})
}

func TestCORSAndLogID_BitsUT(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CORS options 直接返回 204 并写入跨域头", func(t *testing.T) {
		router := gin.New()
		router.Use(CORS())
		router.GET("/resource", func(c *gin.Context) { c.Status(http.StatusOK) })

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodOptions, "/resource", nil)
		req.Header.Set("Origin", "http://example.com")
		router.ServeHTTP(w, req)

		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d", w.Code)
		}
		if w.Header().Get("Access-Control-Allow-Origin") != "http://example.com" {
			t.Fatalf("allow origin = %q", w.Header().Get("Access-Control-Allow-Origin"))
		}
	})

	t.Run("LogID 生成或透传 X-Log-ID 响应头和 context", func(t *testing.T) {
		router := gin.New()
		router.Use(LogID())
		router.GET("/ping", func(c *gin.Context) {
			if c.GetString(logx.LogIDKey) == "" {
				t.Fatalf("log id should be stored in context")
			}
			c.Status(http.StatusNoContent)
		})

		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.Header.Set(logx.LogIDHeader, "client-log-id")
		router.ServeHTTP(w, req)

		if w.Header().Get(logx.LogIDHeader) != "client-log-id" {
			t.Fatalf("log header = %q", w.Header().Get(logx.LogIDHeader))
		}
	})
}

func jsonContainsCode(raw []byte, code string) bool {
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		return false
	}
	return body["code"] == code
}
