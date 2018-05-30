package betproxy

import (
	"net"
	"time"
)

func NewTCPServer(address string) (*TCPServer, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return &TCPServer{
		listener: listener,
	}, nil
}

type TCPServer struct {
	listener net.Listener
}

func (s *TCPServer) Serve(onAcceptHandler func(net.Conn)) error {
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

		go onAcceptHandler(conn)
	}
}

func (s *TCPServer) Close() (err error) {
	return s.listener.Close()
}
