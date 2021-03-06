package betproxy

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/gzip"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

// Session parse the protocol on the connection and call Client.Do handle every http request
type Session struct {
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
		r.RemoteAddr = s.conn.RemoteAddr().String()
		r.RequestURI = ""

		switch r.Method {
		case "CONNECT":
			if _, err = fmt.Fprintf(s.conn, "%s 200 Connection established\r\n\r\n", r.Proto); err != nil {
				return err
			}
			if err = s.handleTLS(r); err != nil {
				return err
			}
		default:
			start := time.Now()

			w := s.handleHTTP(r)
			if err = w.Write(s.writer); err != nil {
				return err
			}
			if err = s.writer.Flush(); err != nil {
				return err
			}
			if err = w.Body.Close(); err != nil {
				return err
			}

			log.Printf("%s %s %db %d %s", r.RemoteAddr, r.URL.String(), w.ContentLength, w.StatusCode, time.Since(start))
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
		return errors.New("invalid protocol")
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

func (s *Session) handleHTTP(r *http.Request) *http.Response {
	var err error

	r.URL.Scheme = "http"
	if s.secure {
		r.URL.Scheme = "https"
	}
	if r.URL.Host == "" {
		r.URL.Host = r.Host
	}

	switch r.Header.Get("Content-Encoding") {
	case "gzip":
		r.Body, err = gzip.NewReader(r.Body)
		if err != nil {
			return HTTPError(http.StatusBadRequest, err.Error(), r)
		}
		r.Header.Set("Content-Encoding", "identity")
	case "deflate":
		r.Body = flate.NewReader(r.Body)
		r.Header.Set("Content-Encoding", "identity")
	}

	for key := range r.Header {
		switch key {
		case "Connection", "Prxoy-Authenticate", "Proxy-Connection", "Trailer", "Transfer-Encoding", "Upgrade":
			r.Header.Del(key)
		}
	}

	res, err := s.service.client.Do(r)
	if err != nil {
		return HTTPError(http.StatusInternalServerError, err.Error(), r)
	}
	if res.ContentLength == -1 {
		res.TransferEncoding = []string{"chunked"}
	}
	return res
}

// Close connection
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
