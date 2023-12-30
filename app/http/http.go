package http

import (
	"fmt"
	"net"
	"os"
	"strings"
)

func Serve(addr string) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Println("Failed to bind to port 4221")
		os.Exit(1)
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
			os.Exit(1)
		}

		handle(conn)
	}
}

func handleRequest(request Request) Response {
	response := NewResponse()
	response.Protocol = request.Protocol
	response.Status = OK
	response.AddHeader("Content-Type", "text/plain")

	if request.Path == "/" {
		return response
	}

	echoSplit := strings.Split(request.Path, "/echo/")
	if len(echoSplit) > 1 {
		response.SetBody(echoSplit[1])
		return response
	}

	isUserAgentQuery := strings.HasPrefix(request.Path, "/user-agent")
	if isUserAgentQuery {
		response.SetBody(request.Headers["User-Agent"])
		return response
	}

	response.Status = NotFound
	return response
}

func handle(conn net.Conn) {
	defer conn.Close()

	request := readRequest(conn)
	response := handleRequest(request)

	writeResponse(conn, response)
}

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

type Request struct {
	Method   string
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
	r.Method = split[0]
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
