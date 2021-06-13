package main

import (
	"example/httpd"
	"fmt"
	"io"
	"log"
	"os"
)

type myHandler struct{}

func (*myHandler) ServeHTTP(w httpd.ResponseWriter, r *httpd.Request) {
	mr, err := r.MultipartReader()
	if err != nil {
		log.Println(err)
		return
	}
	var part *httpd.Part
label:
	for {
		part, err = mr.NextPart()
		if err != nil {
			break
		}
		switch part.FileName() {
		case "":
			fmt.Printf("FormName=%s, FormData:\n", part.FormName())
			if _, err = io.Copy(os.Stdout, part); err != nil {
				break label
			}
			fmt.Println()
		default:
			fmt.Printf("FormName=%s, FileName=%s\n", part.FormName(), part.FileName())
			var file *os.File
			if file, err = os.Create(part.FileName()); err != nil {
				break label
			}
			if _, err = io.Copy(file, part); err != nil {
				file.Close()
				break label
			}
			file.Close()
		}
	}
	if err != io.EOF {
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