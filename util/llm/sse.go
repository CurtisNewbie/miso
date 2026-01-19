package llm

import (
	"strings"

	"github.com/curtisnewbie/miso/util/json"
	"github.com/tmaxmax/go-sse"
)

const (
	sseMessage = "message"
	sseAction  = "action"
	sseData    = "data"
)

func NewSSEPiper[D any]() *SSEPiper[D] {
	return &SSEPiper[D]{}
}

type SSEAction struct {
	Name  string
	Param string
}

func (s SSEAction) UnmarshalParam(p any) error {
	return json.SParseJson(s.Param, p)
}

type SSEPiper[D any] struct {
	onMessage func(delta string, accumulated string) error
	onAction  func(s SSEAction) error
	onData    func(d D) error
}

func (s *SSEPiper[D]) PushMsg(inb interface {
	WriteSSE(name string, message any)
}, m string) {
	inb.WriteSSE(sseMessage, m)
}

func (s *SSEPiper[D]) PushAction(inb interface {
	WriteSSE(name string, message any)
}, m SSEAction) {
	inb.WriteSSE(sseAction, m)
}

func (s *SSEPiper[D]) PushData(inb interface {
	WriteSSE(name string, message any)
}, m D) {
	inb.WriteSSE(sseData, m)
}

func (s *SSEPiper[D]) OnMessage(f func(delta string, accumulated string) error) {
	s.onMessage = f
}

func (s *SSEPiper[D]) OnAction(f func(s SSEAction) error) {
	s.onAction = f
}

func (s *SSEPiper[D]) OnData(f func(d D) error) {
	s.onData = f
}

func (s *SSEPiper[D]) Listen() func(e sse.Event) (stop bool, err error) {
	if s.onMessage == nil {
		s.onMessage = func(delta, accumulated string) error { return nil }
	}
	if s.onAction == nil {
		s.onAction = func(s SSEAction) error { return nil }
	}
	if s.onData == nil {
		s.onData = func(d D) error { return nil }
	}
	accu := &strings.Builder{}
	return func(e sse.Event) (stop bool, err error) {
		switch e.Type {
		case sseMessage:
			accu.WriteString(e.Data)
			if err := s.onMessage(e.Data, accu.String()); err != nil {
				return false, err
			}
		case sseAction:
			a, err := json.SParseJsonAs[SSEAction](e.Data)
			if err != nil {
				return false, err
			}
			if err := s.onAction(a); err != nil {
				return false, err
			}
		case sseData:
			a, err := json.SParseJsonAs[D](e.Data)
			if err != nil {
				return false, err
			}
			if err := s.onData(a); err != nil {
				return false, err
			}
		}
		return false, nil
	}
}
