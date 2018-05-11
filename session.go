package betproxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
)

var ErrInvalidProtocol = errors.New("invalid protocol")

func (s *Service) newSession(conn net.Conn) *Session {
	return &Session{conn: conn}
}

type Session struct {
	sync.Once
	conn    net.Conn
	service *Service
}

func (s *Session) handleLoop() (err error) {
	defer s.Close()

	reader := bufio.NewReaderSize(s.conn, 1024*4)

	for {
		r, err := http.ReadRequest(reader)
		if err != nil {
			return err
		}

		switch r.Method {
		case "CONNECT":
			_, err = fmt.Fprintf(s.conn, "HTTP/%d.%d %03d %s\r\n\r\n", r.ProtoMajor, r.ProtoMinor, http.StatusOK, "Connection established")
			if err != nil {
				return err
			}
			err = s.handleTLS(reader)
			if err != nil {
				return err
			}
		default:
			s.handleHTTP(r)
		}

	}
}

func (s *Session) handleTLS(reader *bufio.Reader) error {
	b := make([]byte, 1)
	if _, err := reader.Read(b); err != nil {
		return err
	}
	buf := make([]byte, reader.Buffered())
	if _, err := reader.Read(buf); err != nil {
		return err
	}

	// 22 is the TLS handshake
	// https://tools.ietf.org/html/rfc5246#section-6.2.1
	if b[0] != 22 {
		return ErrInvalidProtocol
	}

	tlsconn := tls.Server(&peekedConn{s.conn, io.MultiReader(bytes.NewReader(b), bytes.NewReader(buf), s.conn)}, nil)
	if err := tlsconn.Handshake(); err != nil {
		return err
	}
	reader.Reset(tlsconn)
	return nil
}

func (s *Session) handleHTTP(r *http.Request) error {
	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		return HTTPError(s.conn, http.StatusBadRequest, err.Error(), req)
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

	defer res.Body.Close()
	return w.Write(s.conn)
}

func (s *Session) Close() error {
	return s.conn.Close()
}

type peekedConn struct {
	net.Conn
	r io.Reader
}

func (c *peekedConn) Read(buf []byte) (int, error) {
	return c.r.Read(buf)
}
