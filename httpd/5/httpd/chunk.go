package httpd

import (
	"bufio"
	"errors"
	"io"
)

type chunkReader struct {
	//当前正在处理的块中还剩多少字节未读
	n    int
	bufr *bufio.Reader
	//利用done来记录报文主体是否读取完毕
	done bool
	crlf [2]byte //用来读取\r\n
}

func (cw *chunkReader) Read(p []byte) (n int, err error) {
	if cw.done {
		return 0, io.EOF
	}
	if cw.n == 0 {
		cw.n, err = cw.getChunkSize()
		if err != nil {
			return
		}
	}
	if cw.n == 0 {
		cw.done = true
		err = cw.discardCRLF()
		return
	}

	//如果当前块剩余的数据大于等于p的长度
	if len(p) <= cw.n {
		n, err = cw.bufr.Read(p)
		cw.n -= n
		return n, err
	}
	//如果当前块剩余的数据不够p的长度
	n, _ = io.ReadFull(cw.bufr, p[:cw.n])
	cw.n = 0
	//将\r\n从流中消费掉
	if err = cw.discardCRLF(); err != nil {
		return
	}
	return
}

func (cw *chunkReader) discardCRLF() (err error) {
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
