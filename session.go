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

type Session struct {
	sync.Once
	server *Server
	reader *bufio.Reader
	writer *bufio.Writer
	conn   net.Conn
	secure bool
}

func (s *Session) handleLoop() (err error) {
	defer s.Close()

	s.reader = bufio.NewReader(s.conn)
	s.writer = bufio.NewWriter(s.conn)

	for {
		r, err := http.ReadRequest(s.reader)
		if err != nil {
			return err
		}

		switch r.Method {
		case "CONNECT":
			if _, err = fmt.Fprintf(s.conn, "%s 200 Connection established\r\n\r\n", r.Proto); err != nil {
				return err
			}
			if err = s.handleTLS(r); err != nil {
				return err
			}
		default:
			if err = s.handleHTTP(r); err != nil {
				return err
			}
		}
	}
}

func (s *Session) handleTLS(r *http.Request) error {
	b := make([]byte, 1)
	if _, err := s.reader.Read(b); err != nil {
		return err
	}
	buf := make([]byte, s.reader.Buffered())
	if _, err := s.reader.Read(buf); err != nil {
		return err
	}

	// 22 is the TLS handshake
	// https://tools.ietf.org/html/rfc5246#section-6.2.1
	if b[0] != 22 {
		return ErrInvalidProtocol
	}

	tlsconn := tls.Server(&peekedConn{s.conn, io.MultiReader(bytes.NewReader(b), bytes.NewReader(buf), s.conn)}, s.server.tlsCfg.TLSForHost(r.Host))
	if err := tlsconn.Handshake(); err != nil {
		return err
	}
	s.secure = true
	s.reader.Reset(tlsconn)
	s.writer.Reset(tlsconn)
	return nil
}

func (s *Session) handleHTTP(r *http.Request) error {
	defer s.writer.Flush()

	r.URL.Scheme = "http"
	if s.secure {
		r.URL.Scheme = "https"
	}
	if r.URL.Host == "" {
		r.URL.Host = r.Host
	}

	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		return HTTPError(s.writer, http.StatusBadRequest, err.Error(), req)
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
		return HTTPError(s.writer, http.StatusInternalServerError, err.Error(), req)
	}

	w := NewResponse(res.StatusCode, res.Body, req)
	for key, values := range res.Header {
		for _, value := range values {
			w.Header.Add(key, value)
		}
	}
	w.ContentLength = res.ContentLength
	if res.ContentLength == -1 && res.TransferEncoding == nil {
		w.TransferEncoding = []string{"chunked"}
	} else {
		w.TransferEncoding = res.TransferEncoding
	}

	defer res.Body.Close()
	return w.Write(s.writer)
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
