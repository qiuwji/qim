package jwt

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestManagerGenerateParse_BitsUT(t *testing.T) {
	t.Run("生成并解析有效 token", func(t *testing.T) {
		manager := NewManager("secret", time.Hour)
		token, err := manager.Generate(123, "alice")
		if err != nil {
			t.Fatalf("Generate error: %v", err)
		}
		claims, err := manager.Parse(token)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}
		if claims.UID != 123 || claims.Username != "alice" {
			t.Fatalf("claims = %+v", claims)
		}
		if claims.ExpiresAt <= claims.IssuedAt {
			t.Fatalf("expires_at should be after issued_at: %+v", claims)
		}
	})

	t.Run("签名错误返回 invalid token", func(t *testing.T) {
		manager := NewManager("secret", time.Hour)
		token, err := manager.Generate(1, "alice")
		if err != nil {
			t.Fatalf("Generate error: %v", err)
		}
		parts := strings.Split(token, ".")
		parts[2] = "bad-signature"
		_, err = manager.Parse(strings.Join(parts, "."))
		if !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("err = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("格式错误和 payload 错误返回 invalid token", func(t *testing.T) {
		manager := NewManager("secret", time.Hour)
		for _, token := range []string{"", "a.b", "a.b.c", "a.@@@.c"} {
			_, err := manager.Parse(token)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("token %q err = %v, want ErrInvalidToken", token, err)
			}
		}
	})

	t.Run("过期 token 返回 expired token", func(t *testing.T) {
		manager := NewManager("secret", -time.Hour)
		token, err := manager.Generate(1, "alice")
		if err != nil {
			t.Fatalf("Generate error: %v", err)
		}
		_, err = manager.Parse(token)
		if !errors.Is(err, ErrExpiredToken) {
			t.Fatalf("err = %v, want ErrExpiredToken", err)
		}
	})

	t.Run("缺少 uid 或 exp 返回 invalid token", func(t *testing.T) {
		manager := NewManager("secret", time.Hour)
		header, _ := json.Marshal(map[string]string{"alg": "HS256", "typ": "JWT"})
		payload, _ := json.Marshal(Claims{Username: "alice", ExpiresAt: time.Now().Add(time.Hour).Unix()})
		unsigned := base64.RawURLEncoding.EncodeToString(header) + "." + base64.RawURLEncoding.EncodeToString(payload)
		token := unsigned + "." + manager.sign(unsigned)
		_, err := manager.Parse(token)
		if !errors.Is(err, ErrInvalidToken) {
			t.Fatalf("err = %v, want ErrInvalidToken", err)
		}
	})
}
