package httpd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"strconv"
	"strings"
)

type Request struct {
	Method     string
	URL        *url.URL
	Proto      string
	Header     Header
	Body       io.Reader
	RemoteAddr string
	RequestURI string //字符串形式的url

	//私有字段
	conn        *conn
	cookies     map[string]string
	queryString map[string]string
}

func readRequest(c *conn) (r *Request, err error) {
	r = new(Request)
	r.conn = c
	r.RemoteAddr = c.rwc.RemoteAddr().String()
	//读出第一行,如：Get /index HTTP/1.1
	line, err := readLine(c.bufr)
	if err != nil {
		return
	}
	_, err = fmt.Sscanf(string(line), "%s%s%s", &r.Method, &r.RequestURI, &r.Proto)
	if err != nil {
		return
	}
	r.URL, err = url.ParseRequestURI(r.RequestURI)
	if err != nil {
		return
	}
	r.parseQuery()
	//读header
	r.Header, err = readHeader(c.bufr)
	if err != nil {
		return
	}
	const noLimit = (1 << 63) - 1
	r.conn.lr.N = noLimit
	//设置body
	r.setupBody()
	return r, nil
}

//读取一整行
func readLine(bufr *bufio.Reader) ([]byte, error) {
	p, isPrefix, err := bufr.ReadLine()
	if err != nil {
		return nil, err
	}
	var l []byte
	for isPrefix {
		l, isPrefix, err = bufr.ReadLine()
		if err != nil {
			return nil, err
		}
		p = append(p, l...)
	}
	return p, nil
}

func (r *Request) parseQuery() {
	r.queryString = parseQuery(r.URL.RawQuery)
}

func parseQuery(RawQuery string) map[string]string {
	parts := strings.Split(RawQuery, "&")
	queries := make(map[string]string, len(parts))
	for _, part := range parts {
		index := strings.IndexByte(part, '=')
		if index == -1 || index == len(part)-1 {
			continue
		}
		queries[strings.TrimSpace(part[:index])] = strings.TrimSpace(part[index+1:])
	}
	return queries
}

func readHeader(bufr *bufio.Reader) (Header, error) {
	header := make(Header)
	for {
		line, err := readLine(bufr)
		if err != nil {
			return nil, err
		}
		//如果读到/r/n/r/n，代表报文首部的结束
		if len(line) == 0 {
			break
		}
		i := bytes.IndexByte(line, ':')
		if i == -1 {
			return nil, errors.New("unsupported protocol")
		}
		if i == len(line)-1 {
			continue
		}
		k, v := string(line[:i]), strings.TrimSpace(string(line[i+1:]))
		header[k] = append(header[k], v)
	}
	return header, nil
}

func (r *Request) parseCookies() {
	if r.cookies != nil {
		return
	}
	r.cookies = make(map[string]string)
	rawCookies, ok := r.Header["Cookie"]
	if !ok {
		return
	}
	for _, line := range rawCookies {
		kvs := strings.Split(strings.TrimSpace(line), ";")
		if len(kvs) == 1 && kvs[0] == "" {
			continue
		}
		for i := 0; i < len(kvs); i++ {
			index := strings.IndexByte(kvs[i], '=')
			if index == -1 {
				continue
			}
			r.cookies[strings.TrimSpace(kvs[i][:index])] = strings.TrimSpace(kvs[i][index+1:])
		}
	}
	return
}

func (r *Request) Cookie(name string) string {
	if r.cookies == nil {
		r.parseCookies()
	}
	return r.cookies[name]
}

func (r *Request) Query(name string) string {
	return r.queryString[name]
}

type eofReader struct{}

func (er *eofReader) Read([]byte) (n int, err error) { return 0, io.EOF }

func (r *Request) chunked() bool {
	te := r.Header.Get("Transfer-Encoding")
	return te == "chunked"
}

func (r *Request) setupBody() {
	if r.Method != "POST" && r.Method != "PUT" {
		r.Body = &eofReader{} //POST和PUT以外的方法不允许设置报文主体
	}else if r.chunked(){
		r.Body = &chunkReader{bufr: r.conn.bufr}
	} else if cl := r.Header.Get("Content-Length"); cl != "" {
		//如果设置了Content-Length
		contentLength, err := strconv.ParseInt(cl, 10, 64)
		if err != nil {
			r.Body = &eofReader{}
			return
		}
		r.Body = io.LimitReader(r.conn.bufr, contentLength)
	}else{
		r.Body = &eofReader{}
	}
}

func (r *Request) finishRequest() error{
	_,err := io.Copy(ioutil.Discard,r.Body)
	return err
}
