package main

import (
	"example/httpd"
	"fmt"
	"io"
)

type myHandler struct{}

func (*myHandler) ServeHTTP(w httpd.ResponseWriter, r *httpd.Request) {
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
