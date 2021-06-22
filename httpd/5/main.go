package main

import (
	"example/httpd"
	"fmt"
	"io"
	"io/ioutil"
)

type myHandler struct{}

//测试FormFile
func handleTest1(w httpd.ResponseWriter, r *httpd.Request) (err error) {
	fh, err := r.FormFile("file1")
	if err != nil {
		return
	}
	rc, err := fh.Open()
	if err != nil {
		return
	}
	defer rc.Close()
	buf, err := ioutil.ReadAll(rc)
	if err != nil {
		return
	}
	fmt.Printf("%s\n", buf)
	return
}

//测试Save
func handleTest2(w httpd.ResponseWriter, r *httpd.Request) (err error) {
	mr, err := r.MultipartForm()
	if err != nil {
		return
	}
	for _, fh := range mr.File {
		err = fh.Save(fh.Filename)
	}
	return err
}

//测试PostForm
func handleTest3(w httpd.ResponseWriter, r *httpd.Request) (err error) {
	value1 := r.PostForm("foo1")
	value2 := r.PostForm("foo2")
	fmt.Printf("foo1=%s,foo2=%s\n", value1, value2)
	return nil
}

func (*myHandler) ServeHTTP(w httpd.ResponseWriter, r *httpd.Request) {
	var err error
	switch r.URL.Path {
	case "/test1":
		err = handleTest1(w, r)
	case "/test2":
		err = handleTest2(w, r)
	case "/test3":
		err = handleTest3(w, r)
	}
	if err != nil {
		fmt.Println(err)
	}
	io.WriteString(w, "HTTP/1.1 200 OK\r\n")
	io.WriteString(w, fmt.Sprintf("Content-Length: %d\r\n", 0))
	io.WriteString(w, "\r\n")
}

func main() {
	svr := &httpd.Server{
		Addr:    "127.0.0.1:80",
		Handler: new(myHandler),
	}
	panic(svr.ListenAndServe())
}
