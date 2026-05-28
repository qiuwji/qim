package agent

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"qim/internal/dal"
)

func (d *AgentDispatcher) validSession(sessionID string) bool {
	if sessionID == "" || d.agentStore == nil {
		return false
	}
	session, err := d.agentStore.GetSession(sessionID)
	if err != nil || session == nil {
		return false
	}
	now := time.Now().Unix()
	return session.Status == dal.AgentSessionStatusActive && session.ExpiresAt > now
}

func (d *AgentDispatcher) touchSession(sessionID string) error {
	if d.agentStore == nil || sessionID == "" {
		return nil
	}
	now := time.Now().Unix()
	return d.agentStore.TouchSession(sessionID, now, now+int64(agentDisconnectTimeout.Seconds()))
}

func newSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b[:])
}
