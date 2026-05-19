package service

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/call"
)

const callAskTimeout = 5 * time.Second

type CallService struct {
	engine *actor.Engine
}

func NewCallService(engine *actor.Engine) *CallService {
	return &CallService{engine: engine}
}

func (s *CallService) ManagerRef() (*actor.ActorRef, error) {
	ref, ok := s.engine.Lookup("call-manager")
	if !ok {
		return nil, fmt.Errorf("call-manager unavailable")
	}
	return ref, nil
}

func (s *CallService) AskManager(cmd any) (call.Result, error) {
	ref, err := s.ManagerRef()
	if err != nil {
		return call.Result{}, err
	}
	raw, err := ref.Ask(cmd, callAskTimeout)
	if err != nil {
		return call.Result{}, err
	}
	r, ok := raw.(call.Result)
	if !ok {
		return call.Result{}, fmt.Errorf("unexpected result type")
	}
	return r, nil
}
