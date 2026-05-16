package conversation

import (
	"errors"
	"sync"
	"testing"
	"time"

	"qim/internal/actor"
	"qim/internal/eventbus"
)

func TestConversationActorMessageFlow_BitsUT(t *testing.T) {
	store := newConversationTestStore()
	bus := &conversationTestBus{}
	engine := actor.NewEngine()
	ref, err := engine.Spawn("conv-message-test", NewConversationActor(1, store, engine, bus))
	if err != nil {
		t.Fatalf("spawn conversation actor: %v", err)
	}

	raw, err := ref.Ask(SendMessageCmd{SenderID: 1, MsgType: 1, Content: "hello", ClientID: "client-1"}, time.Second)
	if err != nil {
		t.Fatalf("send ask error: %v", err)
	}
	result := raw.(Result)
	if result.Err != nil {
		t.Fatalf("send result error: %v", result.Err)
	}
	dto := result.Data.(MessageDTO)
	if dto.ID != 1001 || dto.Seq != 1 || dto.Content != "hello" {
		t.Fatalf("dto = %+v", dto)
	}
	if got := bus.count(EventMessageSent); got != 1 {
		t.Fatalf("message sent events = %d", got)
	}

	raw, err = ref.Ask(RevokeMessageCmd{OperatorID: 1, MessageID: 1001}, time.Second)
	if err != nil {
		t.Fatalf("revoke ask error: %v", err)
	}
	result = raw.(Result)
	if result.Err != nil || result.Data != true {
		t.Fatalf("revoke result = %+v", result)
	}
	if !store.messageRevoked {
		t.Fatalf("message should be revoked in store")
	}
	if got := bus.count(EventMessageRevoked); got != 1 {
		t.Fatalf("message revoked events = %d", got)
	}
}

func TestConversationActorRealtimeGroupEvents_BitsUT(t *testing.T) {
	store := newConversationTestStore()
	bus := &conversationTestBus{}
	engine := actor.NewEngine()
	ref, err := engine.Spawn("conv-group-test", NewConversationActor(1, store, engine, bus))
	if err != nil {
		t.Fatalf("spawn conversation actor: %v", err)
	}

	cases := []struct {
		name      string
		cmd       any
		eventName string
	}{
		{"正在输入广播", TypingCmd{UID: 1}, EventTyping},
		{"更新群信息广播", UpdateConvInfoCmd{OperatorID: 1, Name: ptrString("new")}, EventConversationUpdated},
		{"邀请成员广播", AddMemberCmd{OperatorID: 1, UID: 3, Role: MemberRoleRegular}, EventMemberJoined},
		{"转让群主广播", TransferOwnerCmd{OperatorID: 1, NewOwnerID: 2}, EventOwnerTransferred},
		{"踢出成员广播", RemoveMemberCmd{OperatorID: 2, UID: 3}, EventMemberKicked},
		{"再次邀请成员广播", AddMemberCmd{OperatorID: 2, UID: 4, Role: MemberRoleRegular}, EventMemberJoined},
		{"成员退群广播", LeaveConvCmd{UID: 4}, EventMemberLeft},
		{"群解散广播", DissolveConvCmd{OperatorID: 2}, EventGroupDissolved},
	}

	for _, tt := range cases {
		raw, err := ref.Ask(tt.cmd, time.Second)
		if err != nil {
			t.Fatalf("%s ask error: %v", tt.name, err)
		}
		result := raw.(Result)
		if result.Err != nil {
			t.Fatalf("%s result error: %v", tt.name, result.Err)
		}
		waitForEvent(t, bus, tt.eventName)
	}
}

func TestManagerActor_BitsUT(t *testing.T) {
	store := newConversationTestStore()
	engine := actor.NewEngine()
	ref, err := engine.Spawn("conv-manager-test", NewManagerActor(store, engine, nil))
	if err != nil {
		t.Fatalf("spawn manager: %v", err)
	}

	raw, err := ref.Ask(ListUserConversationsCmd{UID: 1}, time.Second)
	if err != nil {
		t.Fatalf("list ask: %v", err)
	}
	list := raw.(Result).Data.([]UserConvDTO)
	if len(list) != 1 || list[0].ConversationID != 1 {
		t.Fatalf("list = %+v", list)
	}

	raw, err = ref.Ask(CreatePrivateConvCmd{UID1: 1, UID2: 2}, time.Second)
	if err != nil {
		t.Fatalf("create private ask: %v", err)
	}
	if raw.(Result).Data.(ConversationDTO).ID == 0 {
		t.Fatalf("private conversation should have id")
	}

	raw, err = ref.Ask(CreateGroupConvCmd{OwnerID: 1, Name: "g", Members: []uint64{2}}, time.Second)
	if err != nil {
		t.Fatalf("create group ask: %v", err)
	}
	if raw.(Result).Data.(ConversationDTO).Name != "g" {
		t.Fatalf("group conversation = %+v", raw.(Result).Data)
	}

	store.managerErr = errors.New("store down")
	for _, cmd := range []any{
		ListUserConversationsCmd{UID: 1},
		CreatePrivateConvCmd{UID1: 1, UID2: 2},
		CreateGroupConvCmd{OwnerID: 1, Name: "g"},
		PinConvCmd{UID: 1, ConversationID: 1, Pinned: true},
		MuteConvCmd{UID: 1, ConversationID: 1, Muted: true},
		ReadConvCmd{UID: 1, ConversationID: 1, Seq: 5},
	} {
		raw, err := ref.Ask(cmd, time.Second)
		if err != nil {
			t.Fatalf("%T ask error: %v", cmd, err)
		}
		if raw.(Result).Err == nil {
			t.Fatalf("%T should return store error", cmd)
		}
	}
}

func TestManagerActorPinMuteRead_BitsUT(t *testing.T) {
	store := newConversationTestStore()
	engine := actor.NewEngine()
	ref, err := engine.Spawn("conv-manager-pmr-test", NewManagerActor(store, engine, nil))
	if err != nil {
		t.Fatalf("spawn manager: %v", err)
	}

	pinCases := []struct {
		name string
		cmd  any
	}{
		{"置顶会话", PinConvCmd{UID: 1, ConversationID: 1, Pinned: true}},
		{"取消置顶", PinConvCmd{UID: 1, ConversationID: 1, Pinned: false}},
		{"免打扰", MuteConvCmd{UID: 1, ConversationID: 1, Muted: true}},
		{"取消免打扰", MuteConvCmd{UID: 1, ConversationID: 1, Muted: false}},
		{"标记已读", ReadConvCmd{UID: 1, ConversationID: 1, Seq: 10}},
		{"全部已读", ReadAllConvCmd{UID: 1}},
	}
	for _, tt := range pinCases {
		raw, err := ref.Ask(tt.cmd, time.Second)
		if err != nil {
			t.Fatalf("%s ask error: %v", tt.name, err)
		}
		if result := raw.(Result); result.Err != nil {
			t.Fatalf("%s result error: %v", tt.name, result.Err)
		}
	}
}

func TestConversationActorSettingsAndPermissionErrors_BitsUT(t *testing.T) {
	store := newConversationTestStore()
	store.members[5] = MemberRecord{ConversationID: 1, UserID: 5, Role: int8(MemberRoleRegular), JoinTime: 1}
	engine := actor.NewEngine()
	ref, err := engine.Spawn("conv-settings-test", NewConversationActor(1, store, engine, nil))
	if err != nil {
		t.Fatalf("spawn conversation actor: %v", err)
	}

	cases := []struct {
		name    string
		cmd     any
		wantErr error
	}{
		{"普通成员不能改群信息", UpdateConvInfoCmd{OperatorID: 5, Name: ptrString("x")}, ErrAdminRequired},
		{"重复邀请已有成员", AddMemberCmd{OperatorID: 1, UID: 2, Role: MemberRoleRegular}, ErrMemberExists},
		{"不能邀请群主角色", AddMemberCmd{OperatorID: 1, UID: 3, Role: MemberRoleOwner}, ErrInvalidRole},
		{"不能移除不存在成员", RemoveMemberCmd{OperatorID: 1, UID: 99}, ErrMemberNotFound},
		{"非群主不能转让群主", TransferOwnerCmd{OperatorID: 2, NewOwnerID: 1}, ErrOwnerRequired},
		{"非群主不能设置角色", SetRoleCmd{OperatorID: 2, UID: 1, Role: MemberRoleAdmin}, ErrOwnerRequired},
		{"群主不能直接退群", LeaveConvCmd{UID: 1}, ErrOwnerRequired},
	}

	for _, tt := range cases {
		raw, err := ref.Ask(tt.cmd, time.Second)
		if err != nil {
			t.Fatalf("%s ask error: %v", tt.name, err)
		}
		result := raw.(Result)
		if !errors.Is(result.Err, tt.wantErr) {
			t.Fatalf("%s err = %v, want %v", tt.name, result.Err, tt.wantErr)
		}
	}

	okCases := []any{
		GetConvInfoQuery{},
		ListMembersQuery{},
		SetRoleCmd{OperatorID: 1, UID: 2, Role: MemberRoleRegular},
	}
	for _, cmd := range okCases {
		raw, err := ref.Ask(cmd, time.Second)
		if err != nil {
			t.Fatalf("%T ask error: %v", cmd, err)
		}
		if result := raw.(Result); result.Err != nil {
			t.Fatalf("%T result error: %v", cmd, result.Err)
		}
	}
}

func TestConversationActorMemberLimitZeroMeansUnlimited_BitsUT(t *testing.T) {
	store := newConversationTestStore()
	store.conv.MemberLimit = 0
	engine := actor.NewEngine()
	ref, err := engine.Spawn("conv-limit-zero-test", NewConversationActor(1, store, engine, nil))
	if err != nil {
		t.Fatalf("spawn conversation actor: %v", err)
	}
	raw, err := ref.Ask(AddMemberCmd{OperatorID: 1, UID: 3, Role: MemberRoleRegular}, time.Second)
	if err != nil {
		t.Fatalf("add member ask error: %v", err)
	}
	if result := raw.(Result); result.Err != nil {
		t.Fatalf("add member with unlimited limit error: %v", result.Err)
	}
}

func TestConversationActorDuplicateMessageAndStoreErrors_BitsUT(t *testing.T) {
	store := newConversationTestStore()
	store.duplicateMessage = true
	engine := actor.NewEngine()
	ref, err := engine.Spawn("conv-duplicate-test", NewConversationActor(1, store, engine, nil))
	if err != nil {
		t.Fatalf("spawn conversation actor: %v", err)
	}
	raw, err := ref.Ask(SendMessageCmd{SenderID: 1, MsgType: 1, Content: "hello", ClientID: "dup"}, time.Second)
	if err != nil {
		t.Fatalf("duplicate send ask: %v", err)
	}
	result := raw.(Result)
	if result.Err != nil {
		t.Fatalf("duplicate send result error: %v", result.Err)
	}
	dto := result.Data.(MessageDTO)
	if dto.Seq != 7 || dto.Content != "old" {
		t.Fatalf("duplicated dto = %+v", dto)
	}

	errorStore := newConversationTestStore()
	errorStore.commitErr = errors.New("commit failed")
	errorRef, err := engine.Spawn("conv-error-test", NewConversationActor(1, errorStore, engine, nil))
	if err != nil {
		t.Fatalf("spawn error conversation actor: %v", err)
	}
	raw, err = errorRef.Ask(SendMessageCmd{SenderID: 1, MsgType: 1, Content: "hello"}, time.Second)
	if err != nil {
		t.Fatalf("error send ask: %v", err)
	}
	if raw.(Result).Err == nil {
		t.Fatalf("commit error should be returned")
	}
}

func TestConversationActorPrivateConversationRejectsGroupOps_BitsUT(t *testing.T) {
	store := newConversationTestStore()
	store.conv.Type = int8(ConvTypePrivate)
	engine := actor.NewEngine()
	ref, err := engine.Spawn("conv-private-branches-test", NewConversationActor(1, store, engine, nil))
	if err != nil {
		t.Fatalf("spawn private conversation actor: %v", err)
	}

	commands := []any{
		UpdateConvInfoCmd{OperatorID: 1, Name: ptrString("x")},
		AddMemberCmd{OperatorID: 1, UID: 3, Role: MemberRoleRegular},
		RemoveMemberCmd{OperatorID: 1, UID: 2},
		LeaveConvCmd{UID: 1},
		SetRoleCmd{OperatorID: 1, UID: 2, Role: MemberRoleAdmin},
		TransferOwnerCmd{OperatorID: 1, NewOwnerID: 2},
		DissolveConvCmd{OperatorID: 1},
	}
	for _, cmd := range commands {
		raw, err := ref.Ask(cmd, time.Second)
		if err != nil {
			t.Fatalf("%T ask error: %v", cmd, err)
		}
		if !errors.Is(raw.(Result).Err, ErrGroupRequired) {
			t.Fatalf("%T err=%v, want ErrGroupRequired", cmd, raw.(Result).Err)
		}
	}
}

func TestConversationActorPublishNilEvents_BitsUT(t *testing.T) {
	a := NewConversationActor(1, nil, nil, nil)
	a.members[1] = &MemberState{UID: 1}
	a.publishMessageSent(1, 1, SendMessageCmd{SenderID: 1}, []uint64{1}, 1)
	a.publishMessageRevoked(&MessageRecord{ID: 1, Seq: 1, SenderID: 1}, 1)
	a.publishConversationUpdated()
	a.publishMemberJoined(2, MemberRoleRegular, 1)
	a.publishMemberLeft(2)
	a.publishMemberKicked(2, 1)
	a.publishOwnerTransferred(1, 2)
	a.publishGroupDissolved(1, []uint64{1})
}

func TestConversationErrorPayload_BitsUT(t *testing.T) {
	payload, ok := ToErrorPayload(ErrNotMember)
	if !ok || payload.Code != ErrCodeNotMember {
		t.Fatalf("payload=%+v ok=%v", payload, ok)
	}
	if _, ok := ToErrorPayload(errors.New("plain")); ok {
		t.Fatalf("plain error should not be recognized as domain payload")
	}
}

func ptrString(s string) *string { return &s }

type conversationTestStore struct {
	mu               sync.Mutex
	conv             *ConversationRecord
	members          map[uint64]MemberRecord
	messageRevoked   bool
	duplicateMessage bool
	commitErr        error
	managerErr       error
}

func newConversationTestStore() *conversationTestStore {
	return &conversationTestStore{
		conv: &ConversationRecord{
			ID:          1,
			Type:        int8(ConvTypeGroup),
			Name:        "group",
			OwnerID:     1,
			MemberLimit: 10,
			CreatedAt:   1,
		},
		members: map[uint64]MemberRecord{
			1: {ConversationID: 1, UserID: 1, Role: int8(MemberRoleOwner), JoinTime: 1},
			2: {ConversationID: 1, UserID: 2, Role: int8(MemberRoleAdmin), JoinTime: 1},
		},
	}
}

func (s *conversationTestStore) GetConversation(id uint64) (*ConversationRecord, error) {
	if id != s.conv.ID {
		return nil, errors.New("not found")
	}
	cp := *s.conv
	return &cp, nil
}

func (s *conversationTestStore) CreateConversation(conv *ConversationRecord) error { return nil }
func (s *conversationTestStore) CreatePrivateConversation(input CreatePrivateConversationInput) (*ConversationRecord, error) {
	if s.managerErr != nil {
		return nil, s.managerErr
	}
	return &ConversationRecord{ID: 10, Type: int8(ConvTypePrivate), OwnerID: input.UID1, MemberLimit: 2, CreatedAt: 1}, nil
}
func (s *conversationTestStore) CreateGroupConversation(input CreateGroupConversationInput) (*ConversationRecord, error) {
	if s.managerErr != nil {
		return nil, s.managerErr
	}
	return &ConversationRecord{ID: 11, Type: int8(ConvTypeGroup), Name: input.Name, Avatar: input.Avatar, OwnerID: input.OwnerID, MemberLimit: 500, CreatedAt: 1}, nil
}
func (s *conversationTestStore) UpdateConversation(id uint64, updates map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := updates["name"].(string); ok {
		s.conv.Name = v
	}
	return nil
}
func (s *conversationTestStore) DissolveConversation(id uint64) error { return nil }
func (s *conversationTestStore) FindPrivateConversation(uid1, uid2 uint64) (*ConversationRecord, error) {
	return nil, nil
}
func (s *conversationTestStore) GetMembers(convID uint64) ([]MemberRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]MemberRecord, 0, len(s.members))
	for _, m := range s.members {
		out = append(out, m)
	}
	return out, nil
}
func (s *conversationTestStore) CreateMember(member *MemberRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.members[member.UserID] = *member
	return nil
}
func (s *conversationTestStore) CreateMembers(members []MemberRecord) error { return nil }
func (s *conversationTestStore) DeleteMember(convID, uid uint64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.members, uid)
	return nil
}
func (s *conversationTestStore) UpdateMember(convID, uid uint64, updates map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	member := s.members[uid]
	if v, ok := updates["role"].(MemberRole); ok {
		member.Role = int8(v)
	}
	if v, ok := updates["role"].(int8); ok {
		member.Role = v
	}
	if v, ok := updates["last_read_seq"].(int64); ok {
		member.LastReadSeq = v
	}
	s.members[uid] = member
	return nil
}
func (s *conversationTestStore) TransferOwner(convID, oldOwnerUID, newOwnerUID uint64) error {
	return nil
}
func (s *conversationTestStore) GetUserConversations(uid uint64) ([]UserConversationRecord, error) {
	if s.managerErr != nil {
		return nil, s.managerErr
	}
	return []UserConversationRecord{{ConversationID: 1, IsPinned: true, UnreadCount: 2, LastMsgAt: 3}}, nil
}
func (s *conversationTestStore) UpdateUserConversation(uid, convID uint64, updates map[string]any) error {
	if s.managerErr != nil {
		return s.managerErr
	}
	return nil
}
func (s *conversationTestStore) MarkConversationRead(uid, convID uint64, seq int64) error {
	if s.managerErr != nil {
		return s.managerErr
	}
	return nil
}
func (s *conversationTestStore) CommitMessage(input MessageCommitInput) (*MessageCommitResult, error) {
	if s.commitErr != nil {
		return nil, s.commitErr
	}
	if s.duplicateMessage {
		return &MessageCommitResult{
			MessageID:  9001,
			Seq:        7,
			SenderID:   input.Message.SenderID,
			MsgType:    input.Message.MsgType,
			Content:    "old",
			ReplyTo:    input.Message.ReplyTo,
			ClientID:   input.Message.ClientID,
			CreatedAt:  99,
			Duplicated: true,
		}, nil
	}
	return &MessageCommitResult{
		MessageID: 1001,
		Seq:       input.Message.Seq,
		SenderID:  input.Message.SenderID,
		MsgType:   input.Message.MsgType,
		Content:   input.Message.Content,
		ReplyTo:   input.Message.ReplyTo,
		ClientID:  input.Message.ClientID,
		CreatedAt: input.Message.CreatedAt,
	}, nil
}
func (s *conversationTestStore) GetMessage(convID, messageID uint64) (*MessageRecord, error) {
	return &MessageRecord{ID: messageID, ConversationID: convID, Seq: 1, SenderID: 1, Revoked: s.messageRevoked}, nil
}
func (s *conversationTestStore) RevokeMessage(convID, messageID uint64) error {
	s.messageRevoked = true
	return nil
}

type conversationTestBus struct {
	mu     sync.Mutex
	events []eventbus.Event
}

func (b *conversationTestBus) Publish(event eventbus.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
	return nil
}
func (b *conversationTestBus) Subscribe(eventName string, subscriber *actor.ActorRef) error {
	return nil
}
func (b *conversationTestBus) Unsubscribe(eventName string, subscriber *actor.ActorRef) error {
	return nil
}
func (b *conversationTestBus) count(name string) int {
	b.mu.Lock()
	defer b.mu.Unlock()
	n := 0
	for _, event := range b.events {
		if event.Name() == name {
			n++
		}
	}
	return n
}

func waitForEvent(t *testing.T, bus *conversationTestBus, name string) {
	t.Helper()
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		if bus.count(name) > 0 {
			return
		}
		time.Sleep(time.Millisecond * 10)
	}
	t.Fatalf("event %s was not published", name)
}
