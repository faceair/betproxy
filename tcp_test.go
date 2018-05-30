package betproxy

import (
	"net"
	"strings"
	"testing"
	"time"
)

func Test_NewTCPServer(t *testing.T) {
	s1, err := NewTCPServer("127.0.0.1:3128")
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
	defer s1.Close()

	_, err = NewTCPServer("127.0.0.1:3128")
	if !strings.Contains(err.Error(), "address already in use") {
		t.Error("err must not nil, but got nil")
	}

	s2, err := NewTCPServer(":0")
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
	defer s2.Close()
}

func Test_TCPServerServe(t *testing.T) {
	server, err := NewTCPServer("127.0.0.1:3128")
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
	defer server.Close()

	count := 0
	go func() {
		server.Serve(func(conn net.Conn) {
			count++
		})
	}()

	_, err = net.Dial("tcp", "127.0.0.1:3128")
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	time.Sleep(time.Millisecond * 100)

	if count != 1 {
		t.Error("onAcceptHandler must be called")
	}
}

type FakeTemporaryErrListener struct {
}

func (l *FakeTemporaryErrListener) Accept() (net.Conn, error) {
	return nil, &TemporaryErr{}
}

func (l *FakeTemporaryErrListener) Close() error {
	return nil
}

func (l *FakeTemporaryErrListener) Addr() net.Addr {
	return nil
}

type TemporaryErr struct {
}

func (e *TemporaryErr) Error() string {
	return "Fake Temporary Err"
}

func (e *TemporaryErr) Timeout() bool {
	return false
}

func (e *TemporaryErr) Temporary() bool {
	return true
}
func Test_TCPServerTemporaryErr(t *testing.T) {
	server := &TCPServer{
		closing:  make(chan struct{}),
		listener: &FakeTemporaryErrListener{},
	}
	defer server.Close()

	count := 0
	go func() {
		server.Serve(func(conn net.Conn) {
			count++
		})
	}()

	time.Sleep(time.Millisecond * 100)

	if count != 0 {
		t.Error("onAcceptHandler must not be called")
	}
}

func Test_TCPServerClose(t *testing.T) {
	server, err := NewTCPServer(":0")
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	err = server.Close()
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	err = server.Close()
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
}
