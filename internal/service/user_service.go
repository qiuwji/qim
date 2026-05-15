package service

import (
	"fmt"

	"qim/internal/actor"
	"qim/internal/domain/user"
)

type UserService struct {
	engine       *actor.Engine
	newSessionFn func(uid uint64) actor.Actor
}

func NewUserService(engine *actor.Engine, newSessionFn func(uid uint64) actor.Actor) *UserService {
	return &UserService{engine: engine, newSessionFn: newSessionFn}
}

func (s *UserService) ManagerRef() (*actor.ActorRef, error) {
	ref, ok := s.engine.Lookup("user-manager")
	if !ok {
		return nil, fmt.Errorf("user-manager unavailable")
	}
	return ref, nil
}

func (s *UserService) SessionRef(uid uint64) (*actor.ActorRef, error) {
	name := fmt.Sprintf("session:%d", uid)
	ref, err := s.engine.GetOrCreate(name, func() actor.Actor {
		return s.newSessionFn(uid)
	})
	if err != nil {
		return nil, fmt.Errorf("session actor unavailable: %w", err)
	}
	return ref, nil
}

func (s *UserService) AskManager(cmd any) (user.Result, error) {
	ref, err := s.ManagerRef()
	if err != nil {
		return user.Result{}, err
	}
	raw, err := ref.Ask(cmd, askTimeout)
	if err != nil {
		return user.Result{}, err
	}
	r, ok := raw.(user.Result)
	if !ok {
		return user.Result{}, fmt.Errorf("unexpected result type")
	}
	return r, nil
}

func (s *UserService) AskSession(uid uint64, cmd any) (user.Result, error) {
	ref, err := s.SessionRef(uid)
	if err != nil {
		return user.Result{}, err
	}
	raw, err := ref.Ask(cmd, askTimeout)
	if err != nil {
		return user.Result{}, err
	}
	r, ok := raw.(user.Result)
	if !ok {
		return user.Result{}, fmt.Errorf("unexpected result type")
	}
	return r, nil
}

func (s *UserService) TellSession(uid uint64, cmd any) error {
	ref, err := s.SessionRef(uid)
	if err != nil {
		return err
	}
	return ref.Tell(cmd)
}
