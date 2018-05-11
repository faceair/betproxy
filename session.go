package betproxy

import (
	"bufio"
	"net"
	"net/http"
	"sync"
)

func (s *Service) newSession(conn net.Conn) *Session {
	return &Session{
		closing: make(chan struct{}),
		conn:    conn,
	}
}

type Session struct {
	sync.Once
	closing chan struct{}
	conn    net.Conn
	service *Service
}

func (s *Session) handleLoop() (err error) {
	reader := bufio.NewReaderSize(s.conn, 1024*4)

	for {
		r, err := http.ReadRequest(reader)
		if err != nil {
			return err
		}

		req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
		if err != nil {
			return HTTPError(s.conn, http.StatusInternalServerError, err.Error(), req)
		}
		for key, values := range req.Header {
			for _, value := range values {
				switch key {
				case "Connection":
				case "Prxoy-Authenticate":
				case "Proxy-Connection":
				case "Trailer":
				case "Transfer-Encoding":
				case "Upgrade":
				default:
					req.Header.Add(key, value)
				}
			}
		}

		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}

		res, err := client.Do(req)
		if err != nil {
			return HTTPError(s.conn, http.StatusInternalServerError, err.Error(), req)
		}

		w := NewResponse(res.StatusCode, res.Body, req)
		for key, values := range res.Header {
			for _, value := range values {
				w.Header.Add(key, value)
			}
		}
		w.ContentLength = res.ContentLength
		w.TransferEncoding = res.TransferEncoding
		err = w.Write(s.conn)
		if err != nil {
			return err
		}

		res.Body.Close()
	}
}
