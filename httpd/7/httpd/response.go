package httpd

import (
	"bufio"
	"fmt"
)

type response struct {
	c *conn

	//是否已经调用过WriteHeader
	wroteHeader bool
	header      Header

	//WriteHeader传入的状态码，默认为200
	statusCode int
	//如果handler已经结束并且Write的长度未超过最大写缓存量，
	//我们给头部自动设置Content-Length
	//如果handler未结束且Write的长度超过了最大写缓存量，
	//我们使用chunk编码传输数据
	handlerDone bool

	//it's a wrapper of chunkWriter
	bufw *bufio.Writer
	cw   *chunkWriter

	req *Request

	//是否在本次http请求结束后关闭tcp连接
	closeAfterReply bool

	chunking bool
}

type ResponseWriter interface {
	Write([]byte) (n int, err error)
	Header() Header
	WriteHeader(statusCode int)
}

func setupResponse(c *conn, req *Request) *response {
	resp := &response{
		c:          c,
		header:     make(Header),
		statusCode: 200,
		req:        req,
	}
	cw := &chunkWriter{resp: resp}
	resp.cw = cw
	resp.bufw = bufio.NewWriterSize(cw, 4096)
	var (
		protoMinor int
		protoMajor int
	)
	fmt.Sscanf(req.Proto, "HTTP/%d.%d", &protoMinor, &protoMajor)
	if protoMajor < 1 || protoMinor == 1 && protoMajor == 0 || req.Header.Get("Connection") == "close" {
		resp.closeAfterReply = true
	}
	return resp
}

//写入流的顺序：response => (*response).bufw => chunkWriter
// =>  (*response).(*conn).bufw => net.Conn
func (w *response) Write(p []byte) (int, error) {
	n, err := w.bufw.Write(p)
	if err != nil {
		w.closeAfterReply = true
	}
	return n, err
}

func (w *response) Header() Header {
	return w.header
}

func (w *response) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.statusCode = statusCode
	w.wroteHeader = true
}
