package service

import (
	"fmt"

	"qim/internal/actor"
	"qim/internal/domain/message"
)

const NumMsgReaders = 4

type MsgService struct {
	engine *actor.Engine
}

func NewMsgService(engine *actor.Engine) *MsgService {
	return &MsgService{engine: engine}
}

func readerName(convID uint64) string {
	return fmt.Sprintf("msg-reader-%d", convID%NumMsgReaders)
}

func (s *MsgService) ReaderRef(convID uint64) (*actor.ActorRef, error) {
	name := readerName(convID)
	ref, ok := s.engine.Lookup(name)
	if !ok {
		return nil, fmt.Errorf("%s unavailable", name)
	}
	return ref, nil
}

func (s *MsgService) Ask(cmd any) (message.Result, error) {
	var ref *actor.ActorRef
	var err error
	switch c := cmd.(type) {
	case message.ListMessagesCmd:
		ref, err = s.ReaderRef(c.ConversationID)
	case message.SearchMessagesCmd:
		ref, err = s.ReaderRef(c.ConversationID)
	default:
		ref, err = s.ReaderRef(0)
	}
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
	ref, err := s.ReaderRef(0)
	if err != nil {
		return err
	}
	return ref.Tell(cmd)
}
