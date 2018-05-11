package betproxy

import (
	"crypto/x509"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/faceair/betproxy/mitm"
)

func NewServer(port int, ca *x509.Certificate, privateKey interface{}) (*Server, error) {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return nil, err
	}
	tlsCfg, err := mitm.NewConfig(ca, privateKey)
	if err != nil {
		return nil, err
	}
	server := &Server{
		closing:  make(chan struct{}),
		listener: listener,
		tlsCfg:   tlsCfg,
	}
	return server, nil
}

type Server struct {
	sync.Once
	tlsCfg   *mitm.Config
	closing  chan struct{}
	listener net.Listener
}

func (s *Server) Serve() error {
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

		go s.OnAcceptHandler(conn)
	}
}

func (s *Server) OnAcceptHandler(conn net.Conn) {
	(&Session{server: s, conn: conn}).handleLoop()
}

func (s *Server) Close() (err error) {
	s.Once.Do(func() {
		close(s.closing)
		err = s.listener.Close()
	})
	return
}
