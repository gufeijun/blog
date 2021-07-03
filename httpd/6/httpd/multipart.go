package httpd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

//https://datatracker.ietf.org/doc/html/rfc2046#section-5.1.1

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

func (mr *MultipartReader) ReadForm() (mf *MultipartForm, err error) {
	mf = &MultipartForm{
		Value: make(map[string]string),
		File:  make(map[string]*FileHeader),
	}
	var part *Part
	var nonFileMaxMemory int64 = 10 << 20 //非文件部分在内存中存取的最大量10MB,超出返回错误
	var fileMaxMemory int64 = 30 << 20    //文件在内存中存取的最大量30MB,超出部分存储到硬盘
	for {
		part, err = mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			return
		}
		if part.FormName() == "" {
			continue
		}
		var buff bytes.Buffer
		var n int64
		//non-file part
		if part.FileName()== "" {
			n, err = io.CopyN(&buff, part, nonFileMaxMemory+1)
			if err != nil && err != io.EOF {
				return
			}
			nonFileMaxMemory -= n
			if nonFileMaxMemory < 0 {
				return nil, errors.New("multipart: message too large")
			}
			mf.Value[part.FormName()] = buff.String()
			continue
		}
		//file part
		n, err = io.CopyN(&buff, part, fileMaxMemory+1)
		if err != nil && err != io.EOF {
			return
		}
		fh := &FileHeader{
			Filename: part.FileName(),
			Header:   part.Header,
		}
		//未达到了内存限制
		if fileMaxMemory >= n {
			fileMaxMemory -= n
			fh.Size = int(n)
			fh.content = buff.Bytes()
			mf.File[part.FormName()] = fh
			continue
		}
		//达到内存限制，将数据存入硬盘
		var file *os.File
		file, err = os.CreateTemp("", "multipart-")
		if err != nil {
			return
		}
		n, err = io.Copy(file, io.MultiReader(&buff, part))
		if cerr := file.Close(); cerr != nil {
			err = cerr
		}
		if err != nil {
			os.Remove(file.Name())
			return
		}
		fh.Size = int(n)
		fh.tmpFile = file.Name()
		mf_, ok := mf.File[part.FormName()]
		if ok {
			os.Remove(mf_.tmpFile)
		}
		mf.File[part.FormName()] = fh
	}
	return mf, nil
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
	p.closed = true
	return err
}

type MultipartForm struct {
	Value map[string]string
	File  map[string]*FileHeader
}

func (mf *MultipartForm) RemoveAll() {
	for _, fh := range mf.File {
		if fh == nil || fh.tmpFile == "" {
			continue
		}
		os.Remove(fh.tmpFile)
	}
}

type FileHeader struct {
	Filename string
	Header   Header
	Size     int
	content  []byte
	tmpFile  string
}

func (fh *FileHeader) inDisk() bool {
	return fh.tmpFile != ""
}

func (fh *FileHeader) Open() (io.ReadCloser, error) {
	if fh.inDisk() {
		return os.Open(fh.tmpFile)
	}
	b := bytes.NewReader(fh.content)
	return ioutil.NopCloser(b), nil
}

func (fh *FileHeader) Save(dest string)(err error){
	rc,err:=fh.Open()
	if err!=nil{
		return
	}
	defer rc.Close()
	file,err:=os.Create(dest)
	if err!=nil{
		return
	}
	defer file.Close()
	_,err = io.Copy(file,rc)
	if err!=nil{
		os.Remove(dest)
	}
	return
}