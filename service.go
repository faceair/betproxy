package betproxy

import (
	"log"
	"net"
	"net/http"

	"github.com/faceair/betproxy/mitm"
)

func NewService(address string, tlsCfg *mitm.Config) (*Service, error) {
	server, err := NewTCPServer(address)
	if err != nil {
		return nil, err
	}
	service := &Service{
		tlsCfg: tlsCfg,
		server: server,
	}
	return service, nil
}

type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

type Service struct {
	tlsCfg *mitm.Config
	server *TCPServer
	client Client
}

func (s *Service) Listen() error {
	if s.client == nil {
		panic("must set proxy client")
	}

	defer s.Close()
	return s.server.Serve(s.OnAcceptHandler)
}

func (s *Service) SetClient(client Client) {
	s.client = client
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
