package betproxy

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/faceair/betproxy/mitm"
)

func NewFakeConn() *FakeConn {
	serverRead, clientWrite := io.Pipe()
	clientRead, serverWrite := io.Pipe()

	return &FakeConn{
		Server: &End{
			Reader: serverRead,
			Writer: serverWrite,
		},
		Client: &End{
			Reader: clientRead,
			Writer: clientWrite,
		},
	}
}

type FakeConn struct {
	Server *End
	Client *End
}

type End struct {
	Reader *io.PipeReader
	Writer *io.PipeWriter
}

func (e End) Read(data []byte) (n int, err error)  { return e.Reader.Read(data) }
func (e End) Write(data []byte) (n int, err error) { return e.Writer.Write(data) }
func (e End) LocalAddr() net.Addr                  { return nil }
func (e End) RemoteAddr() net.Addr                 { return nil }
func (e End) SetDeadline(t time.Time) error        { return nil }
func (e End) SetReadDeadline(t time.Time) error    { return nil }
func (e End) SetWriteDeadline(t time.Time) error   { return nil }
func (e End) Close() (err error) {
	if err = e.Writer.Close(); err == nil {
		err = e.Reader.Close()
	}
	return
}

type HTTPBinResp struct {
	Args struct {
	} `json:"args"`
	Headers struct {
		AcceptEncoding string `json:"Accept-Encoding"`
		Connection     string `json:"Connection"`
		Host           string `json:"Host"`
		UserAgent      string `json:"User-Agent"`
	} `json:"headers"`
	Origin string `json:"origin"`
	URL    string `json:"url"`
}

func ReadResp(conn net.Conn) (*HTTPBinResp, error) {
	res, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		return nil, err
	}
	resp := new(HTTPBinResp)
	err = json.NewDecoder(res.Body).Decode(resp)
	return resp, err
}

func Test_SessionHandleHTTPOK(t *testing.T) {
	conn := NewFakeConn()
	session := &Session{
		service: &Service{
			client: &http.Client{},
		},
		conn: conn.Server,
	}

	go session.handleLoop()

	_, err := conn.Client.Write([]byte("GET /get HTTP/1.1\nHost: httpbin.org\r\n\r\n"))
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	resp, err := ReadResp(conn.Client)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	if resp.URL != "http://httpbin.org/get" {
		t.Error("response url not equal")
	}
}

func Test_SessionHandleHTTPChunked(t *testing.T) {
	conn := NewFakeConn()
	session := &Session{
		service: &Service{
			client: &http.Client{},
		},
		conn: conn.Server,
	}

	go session.handleLoop()

	_, err := conn.Client.Write([]byte("GET /stream/1 HTTP/1.1\nHost: httpbin.org\r\n\r\n"))
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	resp, err := ReadResp(conn.Client)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	if resp.URL != "http://httpbin.org/stream/1" {
		t.Error("response url not equal")
	}
}

func Test_SessionHandleHTTPFailed(t *testing.T) {
	conn := NewFakeConn()
	session := &Session{
		service: &Service{
			client: &http.Client{},
		},
		conn: conn.Server,
	}

	go session.handleLoop()

	_, err := conn.Client.Write([]byte("GET /get HTTP/1.1\nHost: not.exist.domain\r\n\r\n"))
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	res, err := http.ReadResponse(bufio.NewReader(conn.Client), nil)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	if res.StatusCode != 500 {
		t.Errorf("res.StatusCode must be 500, but got %d", res.StatusCode)
	}

	_, err = conn.Client.Write([]byte("| /get HTTP/1.1\nHost: httpbin.org\r\n\r\n"))
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	res, err = http.ReadResponse(bufio.NewReader(conn.Client), nil)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	if res.StatusCode != 400 {
		t.Errorf("res.StatusCode must be 400, but got %d", res.StatusCode)
	}
}

func Test_SessionHandleHTTPSOK(t *testing.T) {
	cacert, cakey, err := mitm.NewAuthority("betproxy", "faceair", 10*365*24*time.Hour)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
	tlsCfg, err := mitm.NewConfig(cacert, cakey)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
	tlsCfg.SkipTLSVerify(true)

	conn := NewFakeConn()
	session := &Session{
		service: &Service{
			client: &http.Client{},
			tlsCfg: tlsCfg,
		},
		conn: conn.Server,
	}

	go session.handleLoop()

	_, err = conn.Client.Write([]byte("CONNECT /get HTTP/1.1\nHost: httpbin.org\r\n\r\n"))
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	res, err := http.ReadResponse(bufio.NewReader(conn.Client), nil)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	if res.StatusCode != 200 {
		t.Errorf("res.StatusCode must be 200, but got %d", res.StatusCode)
	}

	tlsConn := tls.Client(conn.Client, tlsCfg.TLSForHost("httpbin.org"))

	err = tlsConn.Handshake()
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	_, err = tlsConn.Write([]byte("GET /get HTTP/1.1\nHost: httpbin.org\r\n\r\n"))
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	res, err = http.ReadResponse(bufio.NewReader(tlsConn), nil)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	resp := new(HTTPBinResp)
	err = json.NewDecoder(res.Body).Decode(resp)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	if resp.URL != "https://httpbin.org/get" {
		t.Error("response url not equal")
	}
}
