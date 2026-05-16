package friend

import (
	"errors"
	"sync"
	"testing"
	"time"

	"qim/internal/actor"
	"qim/internal/dal"
	"qim/internal/eventbus"
)

func TestManagerActorRequestFlow_BitsUT(t *testing.T) {
	store := newFriendTestStore()
	bus := &friendTestBus{}
	engine := actor.NewEngine()
	ref, err := engine.Spawn("friend-manager-test", NewManagerActor(store, engine, bus))
	if err != nil {
		t.Fatalf("spawn friend manager: %v", err)
	}

	raw, err := ref.Ask(SendRequestCmd{FromUID: 1, ToUID: 2, Message: "hi"}, time.Second)
	if err != nil {
		t.Fatalf("send request ask: %v", err)
	}
	result := raw.(Result)
	if result.Err != nil {
		t.Fatalf("send request result error: %v", result.Err)
	}
	dto := result.Data.(FriendRequestDTO)
	if dto.ID == 0 || dto.FromUID != 1 || dto.ToUID != 2 {
		t.Fatalf("dto = %+v", dto)
	}
	if bus.count(EventFriendRequestCreated) != 1 {
		t.Fatalf("request created event missing")
	}
	raw, err = ref.Ask(SendRequestCmd{FromUID: 1, ToUID: 1, Message: "self"}, time.Second)
	if err != nil {
		t.Fatalf("send self request ask: %v", err)
	}
	if !errors.Is(raw.(Result).Err, ErrCannotAddSelf) {
		t.Fatalf("err = %v, want ErrCannotAddSelf", raw.(Result).Err)
	}

	raw, err = ref.Ask(ListIncomingCmd{UID: 2}, time.Second)
	if err != nil {
		t.Fatalf("list incoming ask: %v", err)
	}
	incoming := raw.(Result).Data.([]FriendRequestDTO)
	if len(incoming) != 1 || incoming[0].ID != dto.ID {
		t.Fatalf("incoming = %+v", incoming)
	}

	raw, err = ref.Ask(HandleRequestCmd{UID: 2, ReqID: dto.ID, Accept: true}, time.Second)
	if err != nil {
		t.Fatalf("handle request ask: %v", err)
	}
	result = raw.(Result)
	if result.Err != nil || result.Data != true {
		t.Fatalf("handle result = %+v", result)
	}
	if bus.count(EventFriendRequestHandled) != 1 {
		t.Fatalf("request handled event missing")
	}
	if len(store.friends[1]) != 1 || len(store.friends[2]) != 1 {
		t.Fatalf("bidirectional friends should be created: %+v", store.friends)
	}
}

func TestManagerActorFriendAndGroupOps_BitsUT(t *testing.T) {
	store := newFriendTestStore()
	store.friends[1] = []dal.Friend{{ID: 1, UserID: 1, FriendUID: 2, CreatedAt: 1}}
	store.groups[1] = []dal.FriendGroup{{ID: 10, UserID: 1, Name: "default", SortOrder: 1}}
	engine := actor.NewEngine()
	ref, err := engine.Spawn("friend-ops-test", NewManagerActor(store, engine, nil))
	if err != nil {
		t.Fatalf("spawn friend manager: %v", err)
	}

	ops := []any{
		ListFriendsCmd{UID: 1},
		UpdateRemarkCmd{UID: 1, FriendUID: 2, Remark: "bob"},
		MoveGroupCmd{UID: 1, FriendUID: 2, GroupID: 10},
		ListGroupsCmd{UID: 1},
		CreateGroupCmd{UID: 1, Name: "work"},
		RenameGroupCmd{UID: 1, GroupID: 10, Name: "friends"},
		SortGroupsCmd{UID: 1, Groups: []struct {
			GroupID   uint64
			SortOrder int
		}{{GroupID: 10, SortOrder: 2}}},
		DeleteGroupCmd{UID: 1, GroupID: 10},
		DeleteFriendCmd{UID: 1, FriendUID: 2},
	}
	for _, cmd := range ops {
		raw, err := ref.Ask(cmd, time.Second)
		if err != nil {
			t.Fatalf("%T ask: %v", cmd, err)
		}
		if result := raw.(Result); result.Err != nil {
			t.Fatalf("%T result error: %v", cmd, result.Err)
		}
	}
}

func TestManagerActorRequestRejectAndErrors_BitsUT(t *testing.T) {
	store := newFriendTestStore()
	bus := &friendTestBus{}
	engine := actor.NewEngine()
	ref, err := engine.Spawn("friend-request-branches-test", NewManagerActor(store, engine, bus))
	if err != nil {
		t.Fatalf("spawn friend manager: %v", err)
	}

	raw, err := ref.Ask(SendRequestCmd{FromUID: 1, ToUID: 2, Message: "hi"}, time.Second)
	if err != nil || raw.(Result).Err != nil {
		t.Fatalf("send raw=%+v err=%v", raw, err)
	}
	reqID := raw.(Result).Data.(FriendRequestDTO).ID

	raw, err = ref.Ask(ListOutgoingCmd{UID: 1}, time.Second)
	if err != nil || raw.(Result).Err != nil {
		t.Fatalf("outgoing raw=%+v err=%v", raw, err)
	}
	outgoing := raw.(Result).Data.([]FriendRequestDTO)
	if len(outgoing) != 1 || outgoing[0].ID != reqID {
		t.Fatalf("outgoing = %+v", outgoing)
	}

	raw, err = ref.Ask(HandleRequestCmd{UID: 99, ReqID: reqID, Accept: true}, time.Second)
	if err != nil {
		t.Fatalf("not your request ask: %v", err)
	}
	if !errors.Is(raw.(Result).Err, ErrNotYourRequest) {
		t.Fatalf("err = %v, want ErrNotYourRequest", raw.(Result).Err)
	}

	raw, err = ref.Ask(HandleRequestCmd{UID: 2, ReqID: reqID, Accept: false}, time.Second)
	if err != nil {
		t.Fatalf("reject ask: %v", err)
	}
	if raw.(Result).Err != nil || raw.(Result).Data != true {
		t.Fatalf("reject result = %+v", raw)
	}
	if bus.count(EventFriendRequestHandled) != 1 {
		t.Fatalf("reject should publish handled event")
	}

	NewManagerActor(store, engine, nil).publishFriendRequestCreated(nil)
	NewManagerActor(store, engine, nil).publishFriendRequestHandled(nil, false)
}

func TestManagerActorStoreErrors_BitsUT(t *testing.T) {
	store := newFriendTestStore()
	store.err = errors.New("store down")
	engine := actor.NewEngine()
	ref, err := engine.Spawn("friend-store-errors-test", NewManagerActor(store, engine, nil))
	if err != nil {
		t.Fatalf("spawn friend manager: %v", err)
	}

	commands := []any{
		SendRequestCmd{FromUID: 1, ToUID: 2},
		ListIncomingCmd{UID: 1},
		ListOutgoingCmd{UID: 1},
		HandleRequestCmd{UID: 1, ReqID: 1, Accept: true},
		DeleteFriendCmd{UID: 1, FriendUID: 2},
		ListFriendsCmd{UID: 1},
		UpdateRemarkCmd{UID: 1, FriendUID: 2, Remark: "x"},
		MoveGroupCmd{UID: 1, FriendUID: 2, GroupID: 1},
		ListGroupsCmd{UID: 1},
		CreateGroupCmd{UID: 1, Name: "x"},
		RenameGroupCmd{UID: 1, GroupID: 1, Name: "x"},
		DeleteGroupCmd{UID: 1, GroupID: 1},
	}
	for _, cmd := range commands {
		raw, err := ref.Ask(cmd, time.Second)
		if err != nil {
			t.Fatalf("%T ask error: %v", cmd, err)
		}
		if raw.(Result).Err == nil {
			t.Fatalf("%T should return store error", cmd)
		}
	}
}

type friendTestStore struct {
	mu       sync.Mutex
	nextID   uint64
	requests map[uint64]dal.FriendRequest
	friends  map[uint64][]dal.Friend
	groups   map[uint64][]dal.FriendGroup
	err      error
}

func newFriendTestStore() *friendTestStore {
	return &friendTestStore{
		nextID:   1,
		requests: make(map[uint64]dal.FriendRequest),
		friends:  make(map[uint64][]dal.Friend),
		groups:   make(map[uint64][]dal.FriendGroup),
	}
}

func (s *friendTestStore) CreateRequest(req *dal.FriendRequest) error {
	if s.err != nil {
		return s.err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	req.ID = s.nextID
	s.nextID++
	s.requests[req.ID] = *req
	return nil
}
func (s *friendTestStore) ListIncomingRequests(uid uint64) ([]dal.FriendRequest, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []dal.FriendRequest
	for _, req := range s.requests {
		if req.ToUID == uid {
			out = append(out, req)
		}
	}
	return out, nil
}
func (s *friendTestStore) ListOutgoingRequests(uid uint64) ([]dal.FriendRequest, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []dal.FriendRequest
	for _, req := range s.requests {
		if req.FromUID == uid {
			out = append(out, req)
		}
	}
	return out, nil
}
func (s *friendTestStore) GetRequest(id uint64) (*dal.FriendRequest, error) {
	if s.err != nil {
		return nil, s.err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	req := s.requests[id]
	return &req, nil
}
func (s *friendTestStore) AcceptFriendRequest(reqID uint64, fromUID, toUID uint64) error {
	if s.err != nil {
		return s.err
	}
	s.friends[fromUID] = append(s.friends[fromUID], dal.Friend{UserID: fromUID, FriendUID: toUID})
	s.friends[toUID] = append(s.friends[toUID], dal.Friend{UserID: toUID, FriendUID: fromUID})
	return nil
}
func (s *friendTestStore) RejectFriendRequest(reqID uint64) error { return s.err }
func (s *friendTestStore) CreateFriend(friend *dal.Friend) error  { return s.err }
func (s *friendTestStore) DeleteFriendBidirectional(uid, friendUID uint64) error {
	if s.err != nil {
		return s.err
	}
	s.friends[uid] = nil
	s.friends[friendUID] = nil
	return nil
}
func (s *friendTestStore) ListFriends(uid uint64) ([]dal.Friend, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.friends[uid], nil
}
func (s *friendTestStore) UpdateFriend(uid, friendUID uint64, updates map[string]any) error {
	return s.err
}
func (s *friendTestStore) CreateGroup(group *dal.FriendGroup) error {
	if s.err != nil {
		return s.err
	}
	group.ID = s.nextID
	s.nextID++
	s.groups[group.UserID] = append(s.groups[group.UserID], *group)
	return nil
}
func (s *friendTestStore) ListGroups(uid uint64) ([]dal.FriendGroup, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.groups[uid], nil
}
func (s *friendTestStore) UpdateGroup(id, uid uint64, updates map[string]any) error {
	return s.err
}
func (s *friendTestStore) DeleteGroup(id, uid uint64) error { return s.err }

type friendTestBus struct {
	mu     sync.Mutex
	events []eventbus.Event
}

func (b *friendTestBus) Publish(event eventbus.Event) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.events = append(b.events, event)
	return nil
}
func (b *friendTestBus) Subscribe(eventName string, subscriber *actor.ActorRef) error {
	return nil
}
func (b *friendTestBus) Unsubscribe(eventName string, subscriber *actor.ActorRef) error {
	return nil
}
func (b *friendTestBus) count(name string) int {
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
