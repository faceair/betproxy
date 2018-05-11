package betproxy

import (
	"fmt"
	"net"
	"sync"
	"time"
)

func NewServer(port int, closing chan struct{}) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	server := &Server{
		closing:  closing,
		listener: listener,
	}
	return server, nil
}

type Server struct {
	sync.Once
	closing  chan struct{}
	listener net.Listener
}

func (s *Server) Serve(acceptHandler func(c net.Conn)) error {
	defer s.Close()

	var tempDelay time.Duration
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Temporary() {
				if tempDelay == 0 {
					tempDelay = 5 * time.Millisecond
				} else {
					tempDelay *= 2
				}
				if max := 1 * time.Second; tempDelay > max {
					tempDelay = max
				}

				time.Sleep(tempDelay)
				continue
			}
			return err
		}
		tempDelay = 0

		go acceptHandler(conn)
	}
}

func (s *Server) Close() (err error) {
	s.Once.Do(func() {
		close(s.closing)
		err = s.listener.Close()
	})
	return
}
