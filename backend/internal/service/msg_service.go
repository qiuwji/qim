package service

import (
	"fmt"

	"qim/internal/actor"
	"qim/internal/domain/message"
)

type MsgService struct {
	engine *actor.Engine
}

func NewMsgService(engine *actor.Engine) *MsgService {
	return &MsgService{engine: engine}
}

func (s *MsgService) Ref() (*actor.ActorRef, error) {
	ref, ok := s.engine.Lookup("msg-store")
	if !ok {
		return nil, fmt.Errorf("msg-store unavailable")
	}
	return ref, nil
}

func (s *MsgService) Ask(cmd any) (message.Result, error) {
	ref, err := s.Ref()
	if err != nil {
		return message.Result{}, err
	}
	raw, err := ref.Ask(cmd, askTimeout)
	if err != nil {
		return message.Result{}, err
	}
	r, ok := raw.(message.Result)
	if !ok {
		return message.Result{}, fmt.Errorf("unexpected result type")
	}
	return r, nil
}

func (s *MsgService) Tell(cmd any) error {
	ref, err := s.Ref()
	if err != nil {
		return err
	}
	return ref.Tell(cmd)
}
