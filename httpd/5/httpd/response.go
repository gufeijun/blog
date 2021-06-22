package httpd

type response struct {
	c *conn
}

type ResponseWriter interface {
	Write([]byte) (n int, err error)
}

func setupResponse(c *conn) *response { return &response{c: c} }

func (w *response) Write(p []byte) (int, error) {
	return w.c.bufw.Write(p)
}
