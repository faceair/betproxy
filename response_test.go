package betproxy

import (
	"net/http"
	"testing"
)

func Test_NewResponse(t *testing.T) {
	r := NewResponse(200, nil, nil, nil)
	if r.StatusCode != 200 {
		t.Errorf("StatusCode must be 200, but got %d", r.StatusCode)
	}
	if r.Request != nil {
		t.Errorf("Body must be nil, but got %v", r.Body)
	}

	req := &http.Request{}
	r = NewResponse(400, nil, nil, req)
	if r.StatusCode != 400 {
		t.Errorf("StatusCode must be 200, but got %d", r.StatusCode)
	}
	if r.Request != req {
		t.Error("Request not equal to req")
	}
}

func Test_NewHTTPError(t *testing.T) {
	r := HTTPError(400, "error", nil)
	if r.StatusCode != 400 {
		t.Errorf("StatusCode must be 200, but got %d", r.StatusCode)
	}
	if r.Header.Get("Via") != "betproxy" {
		t.Error("Via not equal to betproxy")
	}
}
