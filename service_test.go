package betproxy

import (
	"net"
	"net/http"
	"testing"
)

func Test_NewService(t *testing.T) {
	_, err := NewService("hi", nil)
	if err == nil {
		t.Error("must error, but got nil")
	}

	service, err := NewService(":3128", nil)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
	defer service.Close()
}

func Test_ServiceOnAccept(t *testing.T) {
	service, err := NewService(":3128", nil)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
	service.client = &http.Client{}
	defer service.Close()

	conn := NewFakeConn()

	go service.OnAcceptHandler(conn.Server)

	_, err = conn.Client.Write([]byte("GET /get HTTP/1.1\nHost: httpbin.org\r\n\r\n"))
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

func Test_ServiceListen(t *testing.T) {
	service, err := NewService(":3128", nil)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
	service.client = &http.Client{}
	defer service.Close()

	go service.Listen()

	conn, err := net.Dial("tcp", "127.0.0.1:3128")
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	_, err = conn.Write([]byte("GET /get HTTP/1.1\nHost: httpbin.org\r\n\r\n"))
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	resp, err := ReadResp(conn)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}

	if resp.URL != "http://httpbin.org/get" {
		t.Error("response url not equal")
	}
}

func Test_ServiceSetClient(t *testing.T) {
	service, err := NewService(":3128", nil)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
	defer service.Close()

	client := &http.Client{}
	service.SetClient(client)
	if service.client != client {
		t.Errorf("set client failed")
	}
}

func Test_ServiceClose(t *testing.T) {
	service, err := NewService(":3128", nil)
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
	err = service.Close()
	if err != nil {
		t.Errorf("err must be nil, but got %s", err.Error())
	}
	err = service.Close()
	if err == nil {
		t.Error("must error, but got nil")
	}
}
