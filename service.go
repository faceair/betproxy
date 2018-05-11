package betproxy

import (
	"net"
	"sync/atomic"
)

func NewService(port int) (*Service, error) {
	s := &Service{
		closing: make(chan struct{}),
	}
	tcp, err := NewServer(port, s.closing)
	if err != nil {
		return nil, err
	}
	s.tcp = tcp
	return s, nil
}

type Service struct {
	closing chan struct{}
	tcp     *Server
	conns   int64
}

func (s *Service) Listen() (err error) {
	return s.tcp.Serve(s.onAcceptConn)
}

func (s *Service) onAcceptConn(c net.Conn) {
	defer atomic.AddInt64(&s.conns, -1)
	atomic.AddInt64(&s.conns, 1)

	s.newSession(c).handleLoop()
}

func (s *Service) Close() error {
	return s.tcp.Close()
}
