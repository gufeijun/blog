package main

import (
	"example/httpd"
	"fmt"
	"io"
	"io/ioutil"
)

type myHandler struct{}

func (*myHandler) ServeHTTP(w httpd.ResponseWriter, r *httpd.Request) {
	buf, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return
	}
	const prefix = "your message:"
	io.WriteString(w, "HTTP/1.1 200 OK\r\n")
	io.WriteString(w, fmt.Sprintf("Content-Length: %d\r\n", len(buf)+len(prefix)))
	io.WriteString(w, "\r\n")
	io.WriteString(w, prefix)
	w.Write(buf)
}

func main() {
	svr := &httpd.Server{
		Addr:    "127.0.0.1:80",
		Handler: new(myHandler),
	}
	panic(svr.ListenAndServe())
}
