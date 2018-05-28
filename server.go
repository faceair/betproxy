package betproxy

import (
	"crypto/x509"
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/faceair/betproxy/mitm"
)

func NewService(port int, ca *x509.Certificate, privateKey interface{}) (*Service, error) {
	server, err := NewTCPServer(port)
	if err != nil {
		return nil, err
	}
	tlsCfg, err := mitm.NewConfig(ca, privateKey)
	if err != nil {
		return nil, err
	}
	service := &Service{
		closing: make(chan struct{}),
		tlsCfg:  tlsCfg,
		client:  &http.Client{},
		server:  server,
	}
	return service, nil
}

type Service struct {
	sync.Once
	closing chan struct{}
	tlsCfg  *mitm.Config
	client  *http.Client
	server  *TCPServer
}

func (s *Service) Listen() error {
	defer s.Close()
	return s.server.Serve(s.OnAcceptHandler)
}

func (s *Service) OnAcceptHandler(conn net.Conn) {
	session := &Session{service: s, conn: conn}
	defer session.Close()

	err := session.handleLoop()
	if err != nil {
		log.Printf("handle session: %s", err.Error())
	}
}

func (s *Service) Close() (err error) {
	s.Once.Do(func() {
		close(s.closing)
		err = s.server.Close()
	})
	return
}
