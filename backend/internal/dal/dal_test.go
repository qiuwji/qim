package dal

import (
	"strings"
	"testing"
	"time"

	convdomain "qim/internal/domain/conversation"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestUserStore_BitsUT(t *testing.T) {
	db := newTestDB(t)
	store := NewUserStore(db)
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}

	user := &User{Username: "alice", Password: string(hash), Nickname: "Alice", CreatedAt: 1, UpdatedAt: 1, LastOnlineAt: 1}
	if err := store.CreateUser(user); err != nil {
		t.Fatalf("CreateUser error: %v", err)
	}
	if user.ID == 0 {
		t.Fatalf("expected auto id")
	}
	if gotUser, err := store.GetUser(user.ID); err != nil || gotUser.Username != "alice" {
		t.Fatalf("GetUser got=%+v err=%v", gotUser, err)
	}
	if gotUser, err := store.GetUserByUsername("alice"); err != nil || gotUser.ID != user.ID {
		t.Fatalf("GetUserByUsername got=%+v err=%v", gotUser, err)
	}

	got, err := store.Authenticate("alice", "secret")
	if err != nil {
		t.Fatalf("Authenticate error: %v", err)
	}
	if got.ID != user.ID {
		t.Fatalf("got id = %d, want %d", got.ID, user.ID)
	}
	if _, err := store.Authenticate("alice", "bad"); err == nil {
		t.Fatalf("wrong password should fail")
	}
	if err := store.UpdateUser(user.ID, map[string]any{"nickname": "A"}); err != nil {
		t.Fatalf("UpdateUser error: %v", err)
	}
	users, err := store.SearchUsers("ali", 10)
	if err != nil || len(users) != 1 {
		t.Fatalf("SearchUsers len = %d err = %v", len(users), err)
	}
}

func TestConvStoreCommitMessageIdempotent_BitsUT(t *testing.T) {
	db := newTestDB(t)
	store := NewConvStore(db)

	conv := &convdomain.ConversationRecord{
		Type:        int8(convdomain.ConvTypeGroup),
		OwnerID:     1,
		MaxSeq:      0,
		MemberLimit: 500,
		CreatedAt:   time.Now().Unix(),
		UpdatedAt:   time.Now().Unix(),
	}
	if err := store.CreateConversation(conv); err != nil {
		t.Fatalf("CreateConversation error: %v", err)
	}
	for _, uid := range []uint64{1, 2} {
		if err := store.CreateMember(&convdomain.MemberRecord{ConversationID: conv.ID, UserID: uid, Role: int8(convdomain.MemberRoleRegular), JoinTime: 1}); err != nil {
			t.Fatalf("CreateMember error: %v", err)
		}
	}

	input := convdomain.MessageCommitInput{
		Message: convdomain.MessageAppendInput{
			ConversationID: conv.ID,
			Seq:            1,
			SenderID:       1,
			MsgType:        1,
			Content:        "hello",
			ClientID:       "client-1",
			CreatedAt:      10,
		},
		UnreadProjection: convdomain.UnreadProjectionInput{
			ConversationID: conv.ID,
			SenderID:       1,
			MemberUIDs:     []uint64{1, 2},
			LastMsgAt:      10,
		},
	}
	first, err := store.CommitMessage(input)
	if err != nil {
		t.Fatalf("first CommitMessage error: %v", err)
	}
	input.Message.Seq = 2
	second, err := store.CommitMessage(input)
	if err != nil {
		t.Fatalf("second CommitMessage error: %v", err)
	}
	if !second.Duplicated || first.MessageID != second.MessageID || second.Seq != 1 {
		t.Fatalf("idempotent result first=%+v second=%+v", first, second)
	}

	record, err := store.GetMessage(conv.ID, first.MessageID)
	if err != nil {
		t.Fatalf("GetMessage error: %v", err)
	}
	if record.SenderID != 1 || record.Revoked {
		t.Fatalf("record = %+v", record)
	}
	var beforeRevoke UserConversation
	if err := db.Where("user_id = ? AND conversation_id = ?", 2, conv.ID).First(&beforeRevoke).Error; err != nil {
		t.Fatalf("query unread before revoke: %v", err)
	}
	if beforeRevoke.UnreadCount != 1 {
		t.Fatalf("unread before revoke = %d, want 1", beforeRevoke.UnreadCount)
	}
	if err := store.RevokeMessage(conv.ID, first.MessageID); err != nil {
		t.Fatalf("RevokeMessage error: %v", err)
	}
	revoked, err := store.GetMessage(conv.ID, first.MessageID)
	if err != nil {
		t.Fatalf("GetMessage revoked error: %v", err)
	}
	if !revoked.Revoked {
		t.Fatalf("message should be revoked")
	}
	var afterRevoke UserConversation
	if err := db.Where("user_id = ? AND conversation_id = ?", 2, conv.ID).First(&afterRevoke).Error; err != nil {
		t.Fatalf("query unread after revoke: %v", err)
	}
	if afterRevoke.UnreadCount != 0 {
		t.Fatalf("unread after revoke = %d, want 0", afterRevoke.UnreadCount)
	}
}

func TestConvStoreCRUD_BitsUT(t *testing.T) {
	db := newTestDB(t)
	store := NewConvStore(db)

	privateConv, err := store.CreatePrivateConversation(convdomain.CreatePrivateConversationInput{
		UID1:      1,
		UID2:      2,
		CreatedAt: 10,
	})
	if err != nil {
		t.Fatalf("CreatePrivateConversation error: %v", err)
	}
	if privateConv.ID == 0 || privateConv.Type != int8(convdomain.ConvTypePrivate) {
		t.Fatalf("private conv = %+v", privateConv)
	}
	foundPrivate, err := store.FindPrivateConversation(2, 1)
	if err != nil {
		t.Fatalf("FindPrivateConversation error: %v", err)
	}
	if foundPrivate.ID != privateConv.ID {
		t.Fatalf("found private id = %d, want %d", foundPrivate.ID, privateConv.ID)
	}

	groupConv, err := store.CreateGroupConversation(convdomain.CreateGroupConversationInput{
		OwnerID:    1,
		Name:       "group",
		Avatar:     "/g.png",
		MemberUIDs: []uint64{2, 3},
		CreatedAt:  20,
	})
	if err != nil {
		t.Fatalf("CreateGroupConversation error: %v", err)
	}
	if groupConv.ID == 0 || groupConv.Type != int8(convdomain.ConvTypeGroup) {
		t.Fatalf("group conv = %+v", groupConv)
	}
	if err := store.UpdateConversation(groupConv.ID, map[string]any{"name": "new", "avatar": "/new.png"}); err != nil {
		t.Fatalf("UpdateConversation error: %v", err)
	}
	updated, err := store.GetConversation(groupConv.ID)
	if err != nil {
		t.Fatalf("GetConversation error: %v", err)
	}
	if updated.Name != "new" || updated.Avatar != "/new.png" {
		t.Fatalf("updated conv = %+v", updated)
	}

	members, err := store.GetMembers(groupConv.ID)
	if err != nil {
		t.Fatalf("GetMembers error: %v", err)
	}
	if len(members) != 3 {
		t.Fatalf("members len = %d, want 3", len(members))
	}
	if err := store.CreateMembers([]convdomain.MemberRecord{{ConversationID: groupConv.ID, UserID: 4, Role: int8(convdomain.MemberRoleRegular), JoinTime: 21}}); err != nil {
		t.Fatalf("CreateMembers error: %v", err)
	}
	if err := store.UpdateMember(groupConv.ID, 4, map[string]any{"role": int8(convdomain.MemberRoleAdmin), "last_read_seq": int64(5)}); err != nil {
		t.Fatalf("UpdateMember error: %v", err)
	}
	if err := store.TransferOwner(groupConv.ID, 1, 2); err != nil {
		t.Fatalf("TransferOwner error: %v", err)
	}
	if err := store.DeleteMember(groupConv.ID, 4); err != nil {
		t.Fatalf("DeleteMember error: %v", err)
	}
	if err := db.Create(&UserConversation{UserID: 2, ConversationID: groupConv.ID, UnreadCount: 3}).Error; err != nil {
		t.Fatalf("create user conversation error: %v", err)
	}
	if err := store.UpdateUserConversation(2, groupConv.ID, map[string]any{"is_pinned": true, "is_muted": true}); err != nil {
		t.Fatalf("UpdateUserConversation error: %v", err)
	}
	userConvs, err := store.GetUserConversations(2)
	if err != nil {
		t.Fatalf("GetUserConversations error: %v", err)
	}
	if len(userConvs) == 0 || !userConvs[0].IsPinned || !userConvs[0].IsMuted {
		t.Fatalf("user conversations = %+v", userConvs)
	}
	if err := store.MarkConversationRead(2, groupConv.ID, 5); err != nil {
		t.Fatalf("MarkConversationRead error: %v", err)
	}
	var uc UserConversation
	if err := db.Where("user_id = ? AND conversation_id = ?", 2, groupConv.ID).First(&uc).Error; err != nil {
		t.Fatalf("query user conversation error: %v", err)
	}
	var member Member
	if err := db.Where("user_id = ? AND conversation_id = ?", 2, groupConv.ID).First(&member).Error; err != nil {
		t.Fatalf("query member error: %v", err)
	}
	if uc.UnreadCount != 0 || member.LastReadSeq != 5 {
		t.Fatalf("read state not updated, uc=%+v member=%+v", uc, member)
	}
	if err := store.DissolveConversation(groupConv.ID); err != nil {
		t.Fatalf("DissolveConversation error: %v", err)
	}
}

func TestFriendAndMsgStore_BitsUT(t *testing.T) {
	db := newTestDB(t)
	friendStore := NewFriendStore(db)
	msgStore := NewMsgStore(db)

	req := &FriendRequest{FromUID: 1, ToUID: 2, Message: "hi", Status: 0, CreatedAt: 1, UpdatedAt: 1}
	if err := friendStore.CreateRequest(req); err != nil {
		t.Fatalf("CreateRequest error: %v", err)
	}
	if incoming, err := friendStore.ListIncomingRequests(2); err != nil || len(incoming) != 1 {
		t.Fatalf("incoming len=%d err=%v", len(incoming), err)
	}
	if outgoing, err := friendStore.ListOutgoingRequests(1); err != nil || len(outgoing) != 1 {
		t.Fatalf("outgoing len=%d err=%v", len(outgoing), err)
	}
	gotReq, err := friendStore.GetRequest(req.ID)
	if err != nil || gotReq.ID != req.ID {
		t.Fatalf("GetRequest got=%+v err=%v", gotReq, err)
	}
	if err := friendStore.AcceptFriendRequest(req.ID, 1, 2); err != nil {
		t.Fatalf("AcceptFriendRequest error: %v", err)
	}
	if err := friendStore.AcceptFriendRequest(req.ID, 1, 2); err != nil {
		t.Fatalf("AcceptFriendRequest should be idempotent: %v", err)
	}
	if friends, err := friendStore.ListFriends(1); err != nil || len(friends) != 1 || friends[0].FriendUID != 2 {
		t.Fatalf("friends after duplicate accept = %+v err=%v", friends, err)
	}
	if err := friendStore.CreateFriend(&Friend{UserID: 1, FriendUID: 5, CreatedAt: 1}); err != nil {
		t.Fatalf("CreateFriend error: %v", err)
	}
	if friends, err := friendStore.ListFriends(1); err != nil || len(friends) != 2 {
		t.Fatalf("friends len=%d err=%v", len(friends), err)
	}
	if err := friendStore.UpdateFriend(1, 2, map[string]any{"remark": "bob", "group_id": 1}); err != nil {
		t.Fatalf("UpdateFriend error: %v", err)
	}
	group := &FriendGroup{UserID: 1, Name: "work", SortOrder: 1}
	if err := friendStore.CreateGroup(group); err != nil {
		t.Fatalf("CreateGroup error: %v", err)
	}
	if groups, err := friendStore.ListGroups(1); err != nil || len(groups) != 1 {
		t.Fatalf("groups len=%d err=%v", len(groups), err)
	}
	if err := friendStore.UpdateGroup(group.ID, 1, map[string]any{"name": "friends"}); err != nil {
		t.Fatalf("UpdateGroup error: %v", err)
	}
	if err := friendStore.DeleteGroup(group.ID, 1); err != nil {
		t.Fatalf("DeleteGroup error: %v", err)
	}
	if err := friendStore.DeleteFriendBidirectional(1, 2); err != nil {
		t.Fatalf("DeleteFriendBidirectional error: %v", err)
	}

	pending := &FriendRequest{FromUID: 3, ToUID: 4, Message: "reject", Status: 0, CreatedAt: 1, UpdatedAt: 1}
	if err := friendStore.CreateRequest(pending); err != nil {
		t.Fatalf("CreateRequest pending error: %v", err)
	}
	if err := friendStore.RejectFriendRequest(pending.ID); err != nil {
		t.Fatalf("RejectFriendRequest error: %v", err)
	}

	messages := []Message{
		{ConversationID: 1, Seq: 1, SenderID: 1, MsgType: 1, Content: "hello", CreatedAt: 1},
		{ConversationID: 1, Seq: 2, SenderID: 2, MsgType: 1, Content: "world hello", CreatedAt: 2},
	}
	for i := range messages {
		if err := msgStore.CreateMessage(&messages[i]); err != nil {
			t.Fatalf("CreateMessage error: %v", err)
		}
	}
	if listed, err := msgStore.ListMessages(1, 2, 0); err != nil || len(listed) != 1 {
		t.Fatalf("ListMessages len=%d err=%v", len(listed), err)
	}
	if found, err := msgStore.SearchMessages(1, "hello", 0); err != nil || len(found) != 2 {
		t.Fatalf("SearchMessages len=%d err=%v", len(found), err)
	}
	if err := db.Model(&Message{}).Where("id = ?", messages[0].ID).Update("revoked", true).Error; err != nil {
		t.Fatalf("revoke message error: %v", err)
	}
	if found, err := msgStore.SearchMessages(1, "hello", 0); err != nil || len(found) != 1 {
		t.Fatalf("SearchMessages after revoke len=%d err=%v", len(found), err)
	}
}

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := strings.NewReplacer("/", "_", " ", "_").Replace(t.Name())
	db, err := gorm.Open(sqlite.Open("file:"+name+"?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&Conversation{}, &Member{}, &UserConversation{}, &User{}, &Message{}, &FriendRequest{}, &FriendGroup{}, &Friend{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if err := db.Exec(`
CREATE UNIQUE INDEX IF NOT EXISTS idx_msg_client
ON messages(conversation_id, sender_id, client_id)
WHERE client_id <> ''
`).Error; err != nil {
		t.Fatalf("create msg client index: %v", err)
	}
	return db
}
