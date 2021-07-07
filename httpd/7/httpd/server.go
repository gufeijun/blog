package httpd

import "net"

type Handler interface {
	ServeHTTP(w ResponseWriter, r *Request)
}

type Server struct {
	Addr    string
	Handler Handler
}

func (s *Server) ListenAndServe() error {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}
	for {
		rwc, err := l.Accept()
		if err != nil {
			continue
		}
		conn := newConn(rwc, s)
		go conn.serve()
	}
}

type HandlerFunc func(ResponseWriter, *Request)

type ServeMux struct {
	m map[string]HandlerFunc
}

func NewServeMux() *ServeMux {
	return &ServeMux{
		m: make(map[string]HandlerFunc),
	}
}

func (sm *ServeMux) ServeHTTP(w ResponseWriter, r *Request) {
	handler, ok := sm.m[r.URL.Path]
	if !ok {
		if len(r.URL.Path) > 1 && r.URL.Path[len(r.URL.Path)-1] == '/' {
			handler, ok = sm.m[r.URL.Path[:len(r.URL.Path)-1]]
		}
		if !ok {
			w.WriteHeader(StatusNotFound)
			return
		}
	}
	handler(w, r)
}

var defaultServeMux ServeMux

var DefaultServeMux = &defaultServeMux

func (sm *ServeMux) HandleFunc(pattern string, cb HandlerFunc) {
	if sm.m == nil {
		sm.m = make(map[string]HandlerFunc)
	}
	sm.m[pattern] = cb
}

func (sm *ServeMux) Handle(pattern string, handler Handler) {
	if sm.m == nil {
		sm.m = make(map[string]HandlerFunc)
	}
	sm.m[pattern] = handler.ServeHTTP
}

func HandleFunc(pattern string, cb HandlerFunc) {
	DefaultServeMux.HandleFunc(pattern, cb)
}

func Handle(pattern string, handler Handler) {
	DefaultServeMux.Handle(pattern, handler)
}

func ListenAndServe(addr string, handler Handler) error {
	if handler == nil {
		handler = DefaultServeMux
	}
	svr := &Server{
		Addr:    addr,
		Handler: handler,
	}
	return svr.ListenAndServe()
}
