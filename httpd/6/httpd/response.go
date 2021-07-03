package httpd

type response struct {
	c *conn
	wroteHeader bool
	header Header
}

type ResponseWriter interface {
	Write([]byte) (n int, err error)
	Header() Header
	WriteHeader(statusCode int)
}

func setupResponse(c *conn) *response {
	return &response{
		c: c,
		header: make(Header),
	}
}

func (w *response) Write(p []byte) (int, error) {
	return w.c.bufw.Write(p)
}

func (w *response) Header() Header{
	return w.header
}

func (w *response) WriteHeader(statusCode int){
	if w.wroteHeader{
		return
	}

}