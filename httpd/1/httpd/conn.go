package httpd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
)

type conn struct{
	svr  *Server
	rwc  net.Conn
	lr   *io.LimitedReader
	bufr *bufio.Reader
	bufw *bufio.Writer
}

func newConn(rwc net.Conn,svr *Server)*conn{
	lr:=&io.LimitedReader{R: rwc, N: 1<<20}
	return &conn{
		svr: svr,
		rwc: rwc,
		bufw: bufio.NewWriterSize(rwc,4<<10),
		lr:lr,
		bufr: bufio.NewReaderSize(lr,4<<10),
	}
}

func (c *conn) serve(){
	defer func() {
		if err:=recover();err!=nil{
			log.Printf("panic recoverred,err:%v\n",err)
		}
		c.close()
	}()
	//http1.1支持keep-alive长连接，所以一个连接中可能读出
	//多个请求，因此实用for循环读取
	for{
		req,err:=c.readRequest()
		if err!=nil{
			handleErr(err,c)
			return
		}
		res:=c.setupResponse()
		c.svr.Handler.ServeHTTP(res,req)
		//将缓存中的剩余的数据发送到rwc中
		if err=c.bufw.Flush();err!=nil{
			return
		}
	}
}

func (c *conn) readRequest()(*Request,error){
	return readRequest(c)
}

func (c *conn) setupResponse()*response{
	return nil
}

func (c *conn) close(){}

func handleErr(err error,c *conn){fmt.Println(err)}