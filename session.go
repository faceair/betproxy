package betproxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

var ErrInvalidProtocol = errors.New("invalid protocol")

type Session struct {
	sync.Once
	service *Service
	reader  *bufio.Reader
	writer  *bufio.Writer
	conn    net.Conn
	secure  bool
}

func (s *Session) handleLoop() (err error) {
	s.reader = bufio.NewReader(s.conn)
	s.writer = bufio.NewWriter(s.conn)

	for {
		r, err := http.ReadRequest(s.reader)
		if err != nil {
			if err == io.EOF {
				return nil
			}
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

	tlsconn := tls.Server(&peekedConn{s.conn, io.MultiReader(bytes.NewReader(b), bytes.NewReader(buf), s.conn)}, s.service.tlsCfg.TLSForHost(r.Host))
	if err := tlsconn.Handshake(); err != nil {
		return err
	}
	s.secure = true
	s.reader.Reset(tlsconn)
	s.writer.Reset(tlsconn)
	return nil
}

func (s *Session) handleHTTP(r *http.Request) (err error) {
	start := time.Now()

	var w, res *http.Response
	defer func() {
		if err = w.Write(s.writer); err == nil {
			err = s.writer.Flush()
		}
		log.Printf("%s %d %d %s", r.URL.String(), w.ContentLength, w.StatusCode, time.Since(start))
		if res != nil {
			res.Body.Close()
		}
	}()

	r.URL.Scheme = "http"
	if s.secure {
		r.URL.Scheme = "https"
	}
	if r.URL.Host == "" {
		r.URL.Host = r.Host
	}

	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	if err != nil {
		w = HTTPError(http.StatusBadRequest, err.Error(), req)
		return
	}
	for key, values := range r.Header {
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

	res, err = s.service.client.Do(req)
	if err != nil {
		w = HTTPError(http.StatusInternalServerError, err.Error(), req)
		return
	}

	w = NewResponse(res.StatusCode, res.Header, res.Body, req)
	w.ContentLength = res.ContentLength
	if w.ContentLength == -1 {
		w.TransferEncoding = []string{"chunked"}
	}
	return
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
