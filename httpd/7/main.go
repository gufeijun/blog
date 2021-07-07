package main

import (
	"example/httpd"
	"io"
)

func main() {
	httpd.HandleFunc("/foo1", func(w httpd.ResponseWriter, r *httpd.Request) {
		io.WriteString(w, "/foo1")
	})
	httpd.HandleFunc("/foo2", func(w httpd.ResponseWriter, r *httpd.Request) {
		io.WriteString(w, "/foo2")
	})
	httpd.HandleFunc("/foo1/bar1", func(w httpd.ResponseWriter, r *httpd.Request) {
		io.WriteString(w, "/foo1/bar1")
	})
	httpd.ListenAndServe("127.0.0.1:80", nil)
}
