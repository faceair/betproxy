package betproxy

import (
	"log"
	"net"
	"net/http"
	"sync"

	"github.com/faceair/betproxy/mitm"
)

func NewService(address string, tlsCfg *mitm.Config) (*Service, error) {
	server, err := NewTCPServer(address)
	if err != nil {
		return nil, err
	}
	service := &Service{
		tlsCfg: tlsCfg,
		client: &http.Client{},
		server: server,
	}
	return service, nil
}

type Service struct {
	sync.Once
	tlsCfg *mitm.Config
	client *http.Client
	server *TCPServer
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
	return s.server.Close()
}
