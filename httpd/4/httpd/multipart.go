package httpd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"
)

//multipart/form-data example:
//POST / HTTP/1.1
//[[ Less interesting headers ... ]]
//Content-Type: multipart/form-data; boundary=---------------------------735323031399963166993862150
//Content-Length: 834
//
//-----------------------------735323031399963166993862150
//Content-Disposition: form-data; name="text1"
//
//text default
//-----------------------------735323031399963166993862150
//Content-Disposition: form-data; name="text2"
//
//aωb
//-----------------------------735323031399963166993862150
//Content-Disposition: form-data; name="file1"; filename="a.txt"
//Content-Type: text/plain
//
//Content of a.txt.
//
//-----------------------------735323031399963166993862150
//Content-Disposition: form-data; name="file2"; filename="a.html"
//Content-Type: text/html
//
//<!DOCTYPE html><title>Content of a.html.</title>
//
//-----------------------------735323031399963166993862150
//Content-Disposition: form-data; name="file3"; filename="binary"
//Content-Type: application/octet-stream
//
//aωb
//-----------------------------735323031399963166993862150--

const bufSize = 4096

type MultipartReader struct {
	bufr                 *bufio.Reader
	occurEofErr          bool
	crlfDashBoundaryDash []byte
	crlfDashBoundary     []byte
	dashBoundary         []byte
	dashBoundaryDash     []byte
	curPart              *Part
	crlf                 [2]byte
}

func NewMultipartReader(r io.Reader, boundary string) *MultipartReader {
	b := []byte("\r\n--" + boundary + "--")
	return &MultipartReader{
		bufr:                 bufio.NewReaderSize(r, bufSize),
		crlfDashBoundaryDash: b,
		crlfDashBoundary:     b[:len(b)-2],
		dashBoundary:         b[2 : len(b)-2],
		dashBoundaryDash:     b[2:],
	}
}

func (mr *MultipartReader) NextPart() (p *Part, err error) {
	if mr.curPart != nil {
		if err = mr.curPart.Close(); err != nil {
			return
		}
		if err = mr.discardCRLF(); err != nil {
			return
		}
	}
	line, err := mr.readLine()
	if err != nil {
		return
	}
	if bytes.Equal(line, mr.dashBoundaryDash) {
		return nil, io.EOF
	}
	if !bytes.Equal(line, mr.dashBoundary) {
		err = fmt.Errorf("want delimiter %s, but got %s", mr.dashBoundary, line)
		return
	}
	p = new(Part)
	p.mr = mr
	if err = p.readHeader(); err != nil {
		return
	}
	mr.curPart = p
	return
}

func (mr *MultipartReader) discardCRLF() (err error) {
	if _, err = io.ReadFull(mr.bufr, mr.crlf[:]); err == nil {
		if mr.crlf[0] != '\r' && mr.crlf[1] != '\n' {
			err = fmt.Errorf("expect crlf, but got %s", mr.crlf)
		}
	}
	return
}

func (mr *MultipartReader) readLine() ([]byte, error) {
	return readLine(mr.bufr)
}

type Part struct {
	Header           Header
	mr               *MultipartReader
	formName         string
	fileName         string
	closed           bool
	substituteReader io.Reader
	parsed           bool
}

func (p *Part) readHeader() (err error) {
	p.Header, err = readHeader(p.mr.bufr)
	return err
}

func (p *Part) Read(buf []byte) (n int, err error) {
	if p.closed {
		return 0, io.EOF
	}
	if p.substituteReader != nil {
		return p.substituteReader.Read(buf)
	}
	bufr := p.mr.bufr
	var peek []byte
	if p.mr.occurEofErr {
		peek, _ = bufr.Peek(bufr.Buffered())
	} else {
		peek, err = bufr.Peek(bufSize)
		if err == io.EOF {
			p.mr.occurEofErr = true
			return p.Read(buf)
		}
		if err != nil {
			return 0, err
		}
	}
	index := bytes.Index(peek, p.mr.crlfDashBoundary)
	if index != -1 || (index == -1 && p.mr.occurEofErr) {
		p.substituteReader = io.LimitReader(bufr, int64(index))
		return p.substituteReader.Read(buf)
	}
	maxRead := bufSize - len(p.mr.crlfDashBoundary) + 1
	if maxRead > len(buf) {
		maxRead = len(buf)
	}
	return bufr.Read(buf[:maxRead])
}

func (p *Part) FormName() string {
	if !p.parsed {
		p.parseFormData()
	}
	return p.formName
}

func (p *Part) FileName() string {
	if !p.parsed {
		p.parseFormData()
	}
	return p.fileName
}

func (p *Part) parseFormData() {
	p.parsed = true
	cd := p.Header.Get("Content-Disposition")
	ss := strings.Split(cd, ";")
	if len(ss) == 1 || strings.ToLower(ss[0]) != "form-data" {
		return
	}
	for _, s := range ss {
		key, value := getKV(s)
		switch key {
		case "name":
			p.formName = value
		case "filename":
			p.fileName = value
		}
	}
}

func getKV(s string) (key string, value string) {
	ss := strings.Split(s, "=")
	if len(ss) != 2 {
		return
	}
	return strings.TrimSpace(ss[0]), strings.Trim(ss[1], `"`)
}

func (p *Part) Close() error {
	if p.closed {
		return nil
	}
	_, err := io.Copy(ioutil.Discard, p)
	return err
}
