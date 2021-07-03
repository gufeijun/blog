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
	conn          *conn
	cookies       map[string]string
	queryString   map[string]string
	postForm      map[string]string
	multipartForm *MultipartForm

	contentType    string
	boundary       string
	haveParsedForm bool
	parseFormErr   error
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
	r.parseContentType()
	return r, nil
}

//读取一整行
func readLine(bufr *bufio.Reader) ([]byte, error) {
	p, isPrefix, err := bufr.ReadLine()
	if err != nil {
		return p, err
	}
	var l []byte
	for isPrefix {
		l, isPrefix, err = bufr.ReadLine()
		if err != nil {
			break
		}
		p = append(p, l...)
	}
	return p, err
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
	} else if r.chunked() {
		r.Body = &chunkReader{bufr: r.conn.bufr}
	} else if cl := r.Header.Get("Content-Length"); cl != "" {
		//如果设置了Content-Length
		contentLength, err := strconv.ParseInt(cl, 10, 64)
		if err != nil {
			r.Body = &eofReader{}
			return
		}
		r.Body = io.LimitReader(r.conn.bufr, contentLength)
	} else {
		r.Body = &eofReader{}
	}
}

func (r *Request) finishRequest() (err error) {
	//将缓存中的剩余的数据发送到rwc中
	if err = r.conn.bufw.Flush(); err != nil {
		return
	}
	_, err = io.Copy(ioutil.Discard, r.Body)
	if r.multipartForm!=nil{
		r.multipartForm.RemoveAll()
	}
	return err
}

func (r *Request) parseContentType() {
	ct := r.Header.Get("Content-Type")
	//Content-Type: multipart/form-data; boundary=------974767299852498929531610575
	//Content-Type: application/x-www-form-urlencoded
	index := strings.IndexByte(ct, ';')
	if index == -1 {
		r.contentType = ct
		return
	}
	if index == len(ct)-1 {
		return
	}
	ss := strings.Split(ct[index+1:], "=")
	if len(ss) < 2 || strings.TrimSpace(ss[0]) != "boundary" {
		return
	}
	r.contentType, r.boundary = ct[:index], strings.Trim(ss[1],`"`)
	return
}

func (r *Request) MultipartReader()(*MultipartReader,error){
	if r.boundary==""{
		return nil,errors.New("no boundary detected")
	}
	return NewMultipartReader(r.Body,r.boundary),nil
}

func (r *Request) PostForm(name string) string {
	if !r.haveParsedForm {
		r.parseFormErr = r.parseForm()
	}
	if r.parseFormErr != nil || r.postForm == nil {
		return ""
	}
	return r.postForm[name]
}

func (r *Request) MultipartForm() (*MultipartForm, error) {
	if !r.haveParsedForm {
		if err := r.parseForm(); err != nil {
			r.parseFormErr = err
			return nil, err
		}
	}
	return r.multipartForm, r.parseFormErr
}

func (r *Request) parseForm() error {
	if r.Method != "POST" && r.Method != "PUT" {
		return errors.New("missing form body")
	}
	r.haveParsedForm = true
	switch r.contentType {
	case "application/x-www-form-urlencoded":
		return r.parsePostForm()
	case "multipart/form-data":
		return r.parseMultipartForm()
	default:
		return errors.New("unsupported form type")
	}
}

func (r *Request) parsePostForm() error {
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return err
	}
	r.postForm = parseQuery(string(data))
	return nil
}

func (r *Request) parseMultipartForm() (err error) {
	mr,err := r.MultipartReader()
	if err!=nil{
		return
	}
	r.multipartForm,err = mr.ReadForm()
	r.postForm = r.multipartForm.Value
	return
}

func (r *Request) FormFile(key string)(fh* FileHeader,err error){
	mf,err := r.MultipartForm()
	if err!=nil{
		return
	}
	fh,ok:=mf.File[key]
	if !ok{
		return nil,errors.New("http: missing multipart file")
	}
	return
}