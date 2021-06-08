package main

import (
	"example/httpd"
	"fmt"
)

type myHandler struct {}

func (*myHandler) ServeHTTP(w httpd.ResponseWriter,r *httpd.Request){
	fmt.Println("hello world")
}

func main(){
	svr:=httpd.Server{
		Addr: "127.0.0.1:8080",
		Handler:new(myHandler),
	}
	panic(svr.ListenAndServe())
}
