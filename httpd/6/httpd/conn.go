package httpd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
)

type conn struct {
	svr  *Server
	rwc  net.Conn
	lr   *io.LimitedReader
	bufr *bufio.Reader
	bufw *bufio.Writer
}

func newConn(rwc net.Conn, svr *Server) *conn {
	lr := &io.LimitedReader{R: rwc, N: 1 << 20}
	return &conn{
		svr:  svr,
		rwc:  rwc,
		bufw: bufio.NewWriterSize(rwc, 4<<10),
		lr:   lr,
		bufr: bufio.NewReaderSize(lr, 4<<10),
	}
}

func (c *conn) serve() {
	defer func() {
		if err := recover(); err != nil {
			log.Printf("panic recoverred,err:%v\n", err)
		}
		c.close()
	}()
	//http1.1支持keep-alive长连接，所以一个连接中可能读出
	//多个请求，因此实用for循环读取
	for {
		req, err := c.readRequest()
		if err != nil {
			handleErr(err, c)
			return
		}
		resp := c.setupResponse(req)
		c.svr.Handler.ServeHTTP(resp, req)
		if err = req.finishRequest(resp); err != nil {
			return
		}
		if resp.closeAfterReply{
			return
		}
	}
}

func (c *conn) readRequest() (*Request, error) {
	return readRequest(c)
}

func (c *conn) setupResponse(req *Request) *response {
	return setupResponse(c,req)
}

func (c *conn) close() {
	c.rwc.Close()
}

func handleErr(err error, c *conn) {
	fmt.Println("handleErr:err=", err)
}
