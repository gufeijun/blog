package httpd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

type chunkReader struct {
	//当前正在处理的块中还剩多少字节未读
	n    int
	bufr *bufio.Reader
	//利用done来记录报文主体是否读取完毕
	done            bool
	crlf            [2]byte //用来读取\r\n
	haveDiscardCRLF bool
}

func (cw *chunkReader) Read(p []byte) (n int, err error) {
	if cw.done {
		return 0, io.EOF
	}
	var nn int
	lenP := len(p)
	for n < lenP {
		//如果当前块剩余的数据大于等于p的长度
		if len(p) <= cw.n {
			nn, err = cw.bufr.Read(p)
			cw.n -= nn
			return nn, err
		}
		//如果当前块剩余的数据不够p的长度
		_, err = io.ReadFull(cw.bufr, p[:cw.n])
		if err != nil {
			return
		}
		n += cw.n
		p = p[cw.n:]
		//将\r\n从流中消费掉
		if err = cw.discardCRLF(); err != nil {
			return
		}
		//获取当前块中chunk data的长度
		cw.n, err = cw.getChunkSize()
		if err != nil {
			return
		}
		if cw.n == 0 {
			cw.done = true
			err = cw.discardCRLF()
			return
		}
	}
	return
}

func (cw *chunkReader) discardCRLF() (err error) {
	//第一次读chunkSize之前不需要舍弃\r\n
	if !cw.haveDiscardCRLF {
		cw.haveDiscardCRLF = true
		return
	}
	if _, err = io.ReadFull(cw.bufr, cw.crlf[:]); err == nil {
		if cw.crlf[0] != '\r' || cw.crlf[1] != '\n' {
			return errors.New("unsupported encoding format of chunk")
		}
	}
	return
}

func (cw *chunkReader) getChunkSize() (chunkSize int, err error) {
	line, err := readLine(cw.bufr)
	if err != nil {
		return
	}
	//将16进制换算成10进制
	for i := 0; i < len(line); i++ {
		switch {
		case 'a' <= line[i] && line[i] <= 'f':
			chunkSize = chunkSize*16 + int(line[i]-'a') + 10
		case 'A' <= line[i] && line[i] <= 'F':
			chunkSize = chunkSize*16 + int(line[i]-'A') + 10
		case '0' <= line[i] && line[i] <= '9':
			chunkSize = chunkSize*16 + int(line[i]-'0')
		default:
			return 0, errors.New("illegal hex number")
		}
	}
	return
}

type chunkWriter struct {
	resp  *response
	wrote bool
}

//response的bufw是chunkWriter的封装，对response的写实际上是对bufw的写
//因此只有在handler结束后调用bufw.Flush，或者在Handler结束前累计写入超过4096B的数据，
//才会触发chunkWriter的Write方法。我们通过handlerDone来区分这两种情况。
func (cw *chunkWriter) Write(p []byte) (n int, err error) {
	if !cw.wrote {
		cw.finalizeHeader(p)
		if err = cw.writeHeader(); err != nil {
			return
		}
		cw.wrote = true
	}
	bufw := cw.resp.c.bufw
	//当Write数据超过缓存容量时，利用chunk编码传输
	if cw.resp.chunking {
		_,err = fmt.Fprintf(bufw,"%x\r\n",len(p))
		if err!=nil{
			return
		}
	}
	n,err = bufw.Write(p)
	if err == nil && cw.resp.chunking{
		_,err=bufw.WriteString("\r\n")
	}
	return n,err
}

func (cw *chunkWriter) finalizeHeader(p []byte) {
	header := cw.resp.header
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", http.DetectContentType(p))
	}
	if header.Get("Content-Length") == "" && header.Get("Transfer-Encoding") == "" {
		if cw.resp.handlerDone {
			buffered := cw.resp.bufw.Buffered()
			header.Set("Content-Length", strconv.Itoa(buffered))
		} else {
			cw.resp.chunking = true
			header.Set("Transfer-Encoding", "chunked")
		}
	}
}

func (cw *chunkWriter) writeHeader() (err error) {
	codeString := strconv.Itoa(cw.resp.statusCode)
	statusLine := cw.resp.req.Proto + " " + codeString + " " + statusText[cw.resp.statusCode] + "\r\n"
	bufw := cw.resp.c.bufw
	_, err = bufw.WriteString(statusLine)
	if err != nil {
		return
	}
	for key, value := range cw.resp.header {
		_, err = bufw.WriteString(key + ": " + value[0] + "\r\n")
		if err != nil {
			return
		}
	}
	_, err = bufw.WriteString("\r\n")
	return
}