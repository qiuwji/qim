package jwt

import "time"

type Claims struct {
	UID      uint64 `json:"uid"`
	Username string `json:"username"`
}

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

func (m *Manager) Generate(uid uint64, username string) (string, error) {
	return "", nil
}

func (m *Manager) Parse(token string) (*Claims, error) {
	return nil, nil
}
