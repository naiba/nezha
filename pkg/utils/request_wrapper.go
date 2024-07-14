package utils

import (
	"bytes"
	"io"
	"net"
	"net/http"

	"github.com/gin-gonic/gin"
)

var _ io.ReadWriteCloser = &RequestWrapper{}

type RequestWrapper struct {
	req    *http.Request
	reader *bytes.Buffer
	writer net.Conn
}

func NewRequestWrapper(req *http.Request, writer gin.ResponseWriter) (*RequestWrapper, error) {
	conn, _, err := writer.Hijack()
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer(nil)
	if err = req.Write(buf); err != nil {
		return nil, err
	}
	return &RequestWrapper{
		req:    req,
		reader: buf,
		writer: conn,
	}, nil
}

func (rw *RequestWrapper) Read(p []byte) (int, error) {
	count, err := rw.reader.Read(p)
	if err == nil {
		return count, nil
	}
	if err != io.EOF {
		return count, err
	}
	// request 数据读完之后等待客户端断开连接或 grpc 超时
	return rw.writer.Read(p)
}

func (rw *RequestWrapper) Write(p []byte) (int, error) {
	return rw.writer.Write(p)
}

func (rw *RequestWrapper) Close() error {
	rw.req.Body.Close()
	rw.writer.Close()
	return nil
}
