package service

import (
	"fmt"
	"time"

	"qim/internal/actor"
	"qim/internal/domain/conversation"
)

const askTimeout = 5 * time.Second

type ConvService struct {
	engine    *actor.Engine
	newConvFn func(convID uint64) actor.Actor
}

func NewConvService(engine *actor.Engine, newConvFn func(convID uint64) actor.Actor) *ConvService {
	return &ConvService{engine: engine, newConvFn: newConvFn}
}

func (s *ConvService) ManagerRef() (*actor.ActorRef, error) {
	ref, ok := s.engine.Lookup("conv-manager")
	if !ok {
		return nil, fmt.Errorf("conv-manager unavailable")
	}
	return ref, nil
}

func (s *ConvService) ConvRef(convID uint64) (*actor.ActorRef, error) {
	name := fmt.Sprintf("conv:%d", convID)
	ref, err := s.engine.GetOrCreate(name, func() actor.Actor {
		return s.newConvFn(convID)
	})
	if err != nil {
		return nil, fmt.Errorf("conversation actor unavailable: %w", err)
	}
	return ref, nil
}

func (s *ConvService) AskManager(cmd any) (conversation.Result, error) {
	ref, err := s.ManagerRef()
	if err != nil {
		return conversation.Result{}, err
	}
	raw, err := ref.Ask(cmd, askTimeout)
	if err != nil {
		return conversation.Result{}, err
	}
	r, ok := raw.(conversation.Result)
	if !ok {
		return conversation.Result{}, fmt.Errorf("unexpected result type")
	}
	return r, nil
}

func (s *ConvService) AskConv(convID uint64, cmd any) (conversation.Result, error) {
	ref, err := s.ConvRef(convID)
	if err != nil {
		return conversation.Result{}, err
	}
	raw, err := ref.Ask(cmd, askTimeout)
	if err != nil {
		return conversation.Result{}, err
	}
	r, ok := raw.(conversation.Result)
	if !ok {
		return conversation.Result{}, fmt.Errorf("unexpected result type")
	}
	return r, nil
}

func (s *ConvService) TellConv(convID uint64, cmd any) error {
	ref, err := s.ConvRef(convID)
	if err != nil {
		return err
	}
	return ref.Tell(cmd)
}
