package http

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func readRequest(conn net.Conn) Request {
	buf := make([]byte, 1024)
	_, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading request: ", err.Error())
		os.Exit(1)
	}

	request := NewRequest()
	request.Unmarshal(buf)

	return request
}

func writeResponse(conn net.Conn, response Response) {
	_, err := conn.Write(response.Marshal())
	if err != nil {
		fmt.Println("Error writing response: ", err.Error())
		os.Exit(1)
	}
}

type RequestHandler struct {
	Strict  bool
	Handler HandlerFunc
}

func NewRequestHandler(handler HandlerFunc) RequestHandler {
	return RequestHandler{
		Strict:  false,
		Handler: handler,
	}
}

type HandlerFunc func(Request) Response

type Server struct {
	Addr     string
	Handlers map[string]map[Method]RequestHandler
}

func NewServer(addr string) *Server {
	return &Server{
		Addr:     addr,
		Handlers: make(map[string]map[Method]RequestHandler),
	}
}

func (s *Server) Handle(method Method, path string, handler HandlerFunc) {
	s.registerHandler(method, path, handler, false)
}

func (s *Server) HandleStrict(method Method, path string, handler HandlerFunc) {
	s.registerHandler(method, path, handler, true)
}

func (s *Server) registerHandler(method Method, path string, handler HandlerFunc, strict bool) {
	if s.Handlers[path] == nil {
		s.Handlers[path] = make(map[Method]RequestHandler)
	}
	newHandler := NewRequestHandler(handler)
	newHandler.Strict = strict
	s.Handlers[path][method] = newHandler
}

func (s *Server) ListenAndServe() {
	l, err := net.Listen("tcp", s.Addr)
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	ch := make(chan net.Conn)
	go acceptConnections(l, ch)
	for {
		go s.handle(<-ch)
	}
}

func acceptConnections(l net.Listener, ch chan net.Conn) {
	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}
		ch <- conn
	}
}

func (s *Server) handle(conn net.Conn) {
	defer conn.Close()

	request := readRequest(conn)

	for path, method := range s.Handlers {
		isPathMatch := strings.HasPrefix(request.Path, path)
		if !isPathMatch {
			continue
		}

		handler := method[request.Method]
		if handler.Handler == nil {
			continue
		}

		if handler.Strict && request.Path != path {
			continue
		}

		writeResponse(conn, handler.Handler(request))
	}

	writeResponse(conn, s.NotFound(request))
}

func (s *Server) NotFound(request Request) Response {
	response := NewResponse()
	response.Protocol = request.Protocol
	response.Status = NotFound
	response.AddHeader("Content-Type", "text/plain")
	response.SetBody("Not Found")
	return response
}

type Request struct {
	Method   Method
	Path     string
	Protocol string
	Headers  map[string]string
}

func NewRequest() Request {
	return Request{
		Headers: make(map[string]string),
	}
}

func (r *Request) Unmarshal(buf []byte) {
	s := string(buf)

	split := strings.Split(s, "\r\n")
	fmt.Println(split)
	for i, line := range split {
		if i == 0 {
			r.parseRequestLine(line)
		} else {
			r.parseHeaderLine(line)
		}
	}
}

func (r *Request) parseRequestLine(line string) {
	split := strings.Split(line, " ")
	r.Method = Method(split[0])
	r.Path = split[1]
	r.Protocol = split[2]
}

func (r *Request) parseHeaderLine(line string) {
	split := strings.Split(line, ": ")
	if len(split) < 2 {
		return
	}
	if split[0] == "" {
		return
	}
	r.Headers[split[0]] = split[1]
}

type Response struct {
	Protocol string
	Status   Status
	Headers  map[string]string
	Body     string
}

func NewResponse() Response {
	return Response{
		Protocol: "HTTP/1.1",
		Status:   OK,
		Headers:  make(map[string]string),
		Body:     "",
	}
}

func (r *Response) AddHeader(key string, value string) {
	r.Headers[key] = value
}

func (r *Response) RemoveHeader(key string) {
	delete(r.Headers, key)
}

func (r Response) ReplaceHeader(key string, value string) {
	r.RemoveHeader(key)
	r.AddHeader(key, value)
}

func (r Response) HeadersString() string {
	var s string
	for key, value := range r.Headers {
		s += fmt.Sprintf("%s: %s\r\n", key, value)
	}
	return s
}

func (r *Response) SetBody(body string) {
	r.Body = body
	r.AddHeader("Content-Length", fmt.Sprintf("%d", len(r.Body)))
}

func (r *Response) SetBodyFile(filePath string) {
	file, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Error opening file: ", err.Error())
		os.Exit(1)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		fmt.Println("Error getting file info: ", err.Error())
		os.Exit(1)
	}

	fileName := fileInfo.Name()
	fileSize := fileInfo.Size()

	fileContent := make([]byte, fileSize)
	_, err = file.Read(fileContent)
	if err != nil {
		fmt.Println("Error reading file: ", err.Error())
		os.Exit(1)
	}

	r.AddHeader("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileName))
	r.AddHeader("Content-Length", fmt.Sprintf("%d", fileSize))
	r.Body = string(fileContent)
}

func (r Response) Marshal() []byte {
	return []byte(r.String())
}

func (r Response) String() string {
	return fmt.Sprintf("%s %d %s\r\n%s\r\n%s",
		r.Protocol,
		r.Status,
		StatusText(r.Status),
		r.HeadersString(),
		r.Body)
}

type Status int

const (
	OK       Status = 200
	NotFound Status = 404
)

func StatusText(status Status) string {
	switch status {
	case OK:
		return "OK"
	case NotFound:
		return "Not Found"
	}
	return ""
}

type Method string

const (
	UNDEFINED Method = ""
	GET       Method = "GET"
	POST      Method = "POST"
)

func (m Method) String() string {
	return string(m)
}

func NewMethod(s string) Method {
	switch s {
	case "GET":
		return GET
	case "POST":
		return POST
	default:
		return UNDEFINED
	}
}
