package message

import (
	"testing"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
)

func TestMessageStoreActor_BitsUT(t *testing.T) {
	store := &messageTestStore{}
	engine := actor.NewEngine()
	ref, err := engine.Spawn("message-store-test", NewMessageStoreActor(store, engine))
	if err != nil {
		t.Fatalf("spawn message store: %v", err)
	}

	raw, err := ref.Ask(StoreMsgCmd{ConversationID: 1, SenderID: 2, MsgType: MsgTypeText, Content: "hello", ClientID: "c1"}, time.Second)
	if err != nil {
		t.Fatalf("store ask: %v", err)
	}
	result := raw.(Result)
	if result.Err != nil {
		t.Fatalf("store result error: %v", result.Err)
	}
	dto := result.Data.(MessageDTO)
	if dto.ID != 1 || dto.Content != "hello" || dto.ClientID != "c1" {
		t.Fatalf("dto = %+v", dto)
	}

	raw, err = ref.Ask(ListMessagesCmd{ConversationID: 1, BeforeSeq: 10, Limit: 20}, time.Second)
	if err != nil || raw.(Result).Err != nil {
		t.Fatalf("list raw=%+v err=%v", raw, err)
	}
	if len(raw.(Result).Data.([]MessageDTO)) != 1 {
		t.Fatalf("list data = %+v", raw.(Result).Data)
	}

	raw, err = ref.Ask(SearchMessagesCmd{ConversationID: 1, Keyword: "hello", Limit: 20}, time.Second)
	if err != nil || raw.(Result).Err != nil {
		t.Fatalf("search raw=%+v err=%v", raw, err)
	}
	if len(raw.(Result).Data.([]MessageDTO)) != 1 {
		t.Fatalf("search data = %+v", raw.(Result).Data)
	}
}

type messageTestStore struct {
	messages []dal.Message
}

func (s *messageTestStore) CreateMessage(msg *dal.Message) error {
	msg.ID = uint64(len(s.messages) + 1)
	msg.Seq = int64(len(s.messages) + 1)
	s.messages = append(s.messages, *msg)
	return nil
}
func (s *messageTestStore) ListMessages(convID uint64, beforeSeq int64, limit int) ([]dal.Message, error) {
	return s.messages, nil
}
func (s *messageTestStore) SearchMessages(convID uint64, keyword string, limit int) ([]dal.Message, error) {
	return s.messages, nil
}
