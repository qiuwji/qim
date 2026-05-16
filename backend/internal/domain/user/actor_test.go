package user

import (
	"errors"
	"sync"
	"testing"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/eventbus"
)

func TestManagerAndSessionActor_BitsUT(t *testing.T) {
	store := newUserTestStore()
	engine := actor.NewEngine()
	managerRef, err := engine.Spawn("user-manager-test", NewManagerActor(store, engine))
	if err != nil {
		t.Fatalf("spawn manager: %v", err)
	}

	raw, err := managerRef.Ask(RegisterCmd{Username: "10001", Password: "Secret!1", Nickname: "Alice"}, time.Second)
	if err != nil {
		t.Fatalf("register ask: %v", err)
	}
	result := raw.(Result)
	if result.Err != nil {
		t.Fatalf("register result error: %v", result.Err)
	}
	dto := result.Data.(UserDTO)
	if dto.ID == 0 || dto.Username != "10001" {
		t.Fatalf("register dto = %+v", dto)
	}

	raw, err = managerRef.Ask(LoginCmd{Username: "10001", Password: "Secret!1"}, time.Second)
	if err != nil {
		t.Fatalf("login ask: %v", err)
	}
	result = raw.(Result)
	if result.Err != nil {
		t.Fatalf("login result error: %v", result.Err)
	}
	if _, ok := engine.Lookup("session:1"); !ok {
		t.Fatalf("login should create session actor")
	}

	sessionRef, _ := engine.Lookup("session:1")
	raw, err = sessionRef.Ask(GetProfileQuery{}, time.Second)
	if err != nil {
		t.Fatalf("profile ask: %v", err)
	}
	if raw.(Result).Data.(UserDTO).Nickname != "Alice" {
		t.Fatalf("unexpected profile = %+v", raw)
	}

	raw, err = sessionRef.Ask(UpdateProfileCmd{Nickname: "A", Avatar: "/a.png", Sign: "hi"}, time.Second)
	if err != nil || raw.(Result).Err != nil {
		t.Fatalf("update profile raw=%+v err=%v", raw, err)
	}
	raw, err = sessionRef.Ask(ChangePasswordCmd{OldPassword: "Secret!1", NewPassword: "NewSecret!1"}, time.Second)
	if err != nil || raw.(Result).Err != nil {
		t.Fatalf("change password raw=%+v err=%v", raw, err)
	}
	if !checkPassword(store.users[1].Password, "NewSecret!1") {
		t.Fatalf("password should be updated")
	}

	managerRef.Tell(eventbus.EventEnvelope{Event: UserOnlineEvent{UID: 1, At: 123}})
	time.Sleep(50 * time.Millisecond)
	if store.users[1].LastOnlineAt != 123 {
		t.Fatalf("last online = %d", store.users[1].LastOnlineAt)
	}
}

func TestManagerActorQueryErrors_BitsUT(t *testing.T) {
	store := newUserTestStore()
	engine := actor.NewEngine()
	managerRef, err := engine.Spawn("user-manager-error-test", NewManagerActor(store, engine))
	if err != nil {
		t.Fatalf("spawn manager: %v", err)
	}

	raw, err := managerRef.Ask(LoginCmd{Username: "missing", Password: "bad"}, time.Second)
	if err != nil {
		t.Fatalf("login ask: %v", err)
	}
	if !errors.Is(raw.(Result).Err, ErrInvalidCredentials) {
		t.Fatalf("login err = %v", raw.(Result).Err)
	}

	store.users[1] = &dal.User{ID: 1, Username: "alice", Nickname: "Alice"}
	raw, err = managerRef.Ask(SearchUsersCmd{Keyword: "ali"}, time.Second)
	if err != nil || raw.(Result).Err != nil {
		t.Fatalf("search raw=%+v err=%v", raw, err)
	}
	users := raw.(Result).Data.([]UserDTO)
	if len(users) != 1 {
		t.Fatalf("users = %+v", users)
	}

	raw, err = managerRef.Ask(GetUserCmd{UID: 1}, time.Second)
	if err != nil || raw.(Result).Err != nil {
		t.Fatalf("get user raw=%+v err=%v", raw, err)
	}
	if raw.(Result).Data.(UserDTO).Username != "alice" {
		t.Fatalf("get user = %+v", raw.(Result).Data)
	}
}

func TestManagerActorRejectsNonNumericUsername_BitsUT(t *testing.T) {
	store := newUserTestStore()
	engine := actor.NewEngine()
	managerRef, err := engine.Spawn("user-manager-username-test", NewManagerActor(store, engine))
	if err != nil {
		t.Fatalf("spawn manager: %v", err)
	}

	raw, err := managerRef.Ask(RegisterCmd{Username: "alice", Password: "secret", Nickname: "Alice"}, time.Second)
	if err != nil {
		t.Fatalf("register ask: %v", err)
	}
	if !errors.Is(raw.(Result).Err, ErrInvalidUsername) {
		t.Fatalf("register err = %v", raw.(Result).Err)
	}
}

func TestSessionActorErrorBranches_BitsUT(t *testing.T) {
	store := newUserTestStore()
	hash, err := hashPassword("secret")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	store.users[1] = &dal.User{ID: 1, Username: "alice", Password: hash, Nickname: "Alice"}
	engine := actor.NewEngine()
	ref, err := engine.Spawn("user-session-branches-test", NewSessionActor(1, store, engine))
	if err != nil {
		t.Fatalf("spawn session: %v", err)
	}

	raw, err := ref.Ask(UpdateProfileCmd{}, time.Second)
	if err != nil || raw.(Result).Err != nil {
		t.Fatalf("empty update raw=%+v err=%v", raw, err)
	}

	raw, err = ref.Ask(ChangePasswordCmd{OldPassword: "bad", NewPassword: "new"}, time.Second)
	if err != nil {
		t.Fatalf("change bad password ask: %v", err)
	}
	if !errors.Is(raw.(Result).Err, ErrIncorrectPassword) {
		t.Fatalf("err = %v, want ErrIncorrectPassword", raw.(Result).Err)
	}

	NewSessionActor(1, store, engine).OnStart(nil)
	NewSessionActor(1, store, engine).OnStop(nil)
	if (UserOnlineEvent{}).Name() != EventUserOnline {
		t.Fatalf("unexpected user online event name")
	}
}

type userTestStore struct {
	mu     sync.Mutex
	nextID uint64
	users  map[uint64]*dal.User
}

func newUserTestStore() *userTestStore {
	return &userTestStore{nextID: 1, users: make(map[uint64]*dal.User)}
}

func (s *userTestStore) GetUser(id uint64) (*dal.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return nil, errors.New("not found")
	}
	cp := *user
	return &cp, nil
}
func (s *userTestStore) GetUserByUsername(username string) (*dal.User, error) {
	for _, user := range s.users {
		if user.Username == username {
			cp := *user
			return &cp, nil
		}
	}
	return nil, errors.New("not found")
}
func (s *userTestStore) CreateUser(user *dal.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user.ID = s.nextID
	s.nextID++
	cp := *user
	s.users[user.ID] = &cp
	return nil
}
func (s *userTestStore) UpdateUser(id uint64, updates map[string]any) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	user := s.users[id]
	if user == nil {
		return errors.New("not found")
	}
	if v, ok := updates["nickname"].(string); ok {
		user.Nickname = v
	}
	if v, ok := updates["avatar"].(string); ok {
		user.Avatar = v
	}
	if v, ok := updates["sign"].(string); ok {
		user.Sign = v
	}
	if v, ok := updates["password"].(string); ok {
		user.Password = v
	}
	if v, ok := updates["last_online_at"].(int64); ok {
		user.LastOnlineAt = v
	}
	return nil
}
func (s *userTestStore) SearchUsers(keyword string, limit int) ([]dal.User, error) {
	var out []dal.User
	for _, user := range s.users {
		if user.Username == keyword || user.Nickname == keyword || keyword == "ali" {
			out = append(out, *user)
		}
	}
	return out, nil
}
func (s *userTestStore) Authenticate(username, password string) (*dal.User, error) {
	user, err := s.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}
	if !checkPassword(user.Password, password) {
		return nil, errors.New("bad credentials")
	}
	return user, nil
}
