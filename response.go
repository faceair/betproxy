package betproxy

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
)

func NewResponse(code int, header http.Header, body io.Reader, req *http.Request) *http.Response {
	if body == nil {
		body = &bytes.Buffer{}
	}
	if header == nil {
		header = http.Header{}
	}

	rc, ok := body.(io.ReadCloser)
	if !ok {
		rc = ioutil.NopCloser(body)
	}

	res := &http.Response{
		StatusCode: code,
		Status:     fmt.Sprintf("%d %s", code, http.StatusText(code)),
		Proto:      "HTTP/1.1",
		ProtoMajor: 1,
		ProtoMinor: 1,
		Header:     header,
		Body:       rc,
		Request:    req,
	}

	if req != nil {
		res.Close = req.Close
		res.Proto = req.Proto
		res.ProtoMajor = req.ProtoMajor
		res.ProtoMinor = req.ProtoMinor
	}

	return res
}

func HTTPError(code int, err string, req *http.Request) *http.Response {
	res := NewResponse(code, http.Header{
		"Content-Type": []string{"text/plain; charset=utf-8"},
		"Via":          []string{"betproxy"},
	}, strings.NewReader(err), req)
	res.ContentLength = int64(len(err))
	return res
}
