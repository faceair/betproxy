package betproxy

import (
	"log"
	"net"
	"net/http"

	"github.com/faceair/betproxy/mitm"
)

// NewService create a Service instance
// The address is that the proxy server listen on, and the tlsCfg will be used to sign the https website
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

// Client each request is handled by the client
type Client interface {
	Do(req *http.Request) (*http.Response, error)
}

// Service is the proxy server
type Service struct {
	tlsCfg *mitm.Config
	server *TCPServer
	client Client
}

// Listen proxy server start accept connection
func (s *Service) Listen() error {
	if s.client == nil {
		panic("must set proxy client")
	}

	defer s.Close()
	return s.server.Serve(s.OnAcceptHandler)
}

// SetClient as name
func (s *Service) SetClient(client Client) {
	s.client = client
}

// OnAcceptHandler each connection is handled by this method
func (s *Service) OnAcceptHandler(conn net.Conn) {
	session := &Session{service: s, conn: conn}
	defer session.Close()

	err := session.handleLoop()
	if err != nil {
		log.Printf("handle session: %s", err.Error())
	}
}

// Close proxy server
func (s *Service) Close() (err error) {
	return s.server.Close()
}
