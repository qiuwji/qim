package service

import (
	"fmt"

	"qim/internal/actor"
	"qim/internal/domain/friend"
)

type FriendService struct {
	engine *actor.Engine
}

func NewFriendService(engine *actor.Engine) *FriendService {
	return &FriendService{engine: engine}
}

func (s *FriendService) Ref() (*actor.ActorRef, error) {
	ref, ok := s.engine.Lookup("friend-manager")
	if !ok {
		return nil, fmt.Errorf("friend-manager unavailable")
	}
	return ref, nil
}

func (s *FriendService) Ask(cmd any) (friend.Result, error) {
	ref, err := s.Ref()
	if err != nil {
		return friend.Result{}, err
	}
	raw, err := ref.Ask(cmd, askTimeout)
	if err != nil {
		return friend.Result{}, err
	}
	r, ok := raw.(friend.Result)
	if !ok {
		return friend.Result{}, fmt.Errorf("unexpected result type")
	}
	return r, nil
}

func (s *FriendService) Tell(cmd any) error {
	ref, err := s.Ref()
	if err != nil {
		return err
	}
	return ref.Tell(cmd)
}
