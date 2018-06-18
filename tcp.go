package betproxy

import (
	"net"
	"time"
)

// NewTCPServer create new tcp server
func NewTCPServer(address string) (*TCPServer, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	return &TCPServer{
		listener: listener,
	}, nil
}

// TCPServer as name
type TCPServer struct {
	listener net.Listener
}

// Serve start accetp new connection and call onAcceptHandler
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

// Close server
func (s *TCPServer) Close() (err error) {
	return s.listener.Close()
}
