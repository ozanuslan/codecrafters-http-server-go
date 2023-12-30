package http

import (
	"fmt"
	"net"
	"os"
	"strings"
)

type Request struct {
	Method   string
	Path     string
	Protocol string
	Headers  []Header
}

func (r *Request) Unmarshal(buf []byte) {
	s := string(buf)

	s = strings.Trim(s, "\x00")
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
	line = strings.Trim(line, "\r\n")
	split := strings.Split(line, ": ")
	if len(split) != 2 {
		return
	}
	r.Headers = append(r.Headers, Header{split[0], split[1]})
}

type Header struct {
	Key   string
	Value string
}

func NewHeader(key, value string) Header {
	return Header{key, value}
}

func (h Header) String() string {
	return fmt.Sprintf("%s: %s", h.Key, h.Value)
}

func (h Header) Marshal() []byte {
	return []byte(h.String())
}

type Response struct {
	Protocol string
	Status   Status
	Headers  []Header
}

func NewResponse() Response {
	return Response{
		Protocol: "HTTP/1.1",
		Status:   OK,
		Headers:  []Header{},
	}
}

func (r *Response) AddHeader(header Header) {
	r.Headers = append(r.Headers, header)
}

func (r Response) Marshal() []byte {
	return []byte(r.String())
}

func (r Response) String() string {
	return fmt.Sprintf("%s %d %s\r\n%s\r\n\r\n", r.Protocol, r.Status, StatusText(r.Status), r.Headers)
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

	request := Request{}
	request.Unmarshal(buf)

	return request
}

func handleRequest(request Request) Response {
	response := Response{}
	response.Protocol = request.Protocol
	response.Status = OK
	response.AddHeader(NewHeader("Content-Type", "text/html"))

	if request.Path != "/" {
		response.Status = NotFound
	}

	return response
}

func writeResponse(conn net.Conn, response Response) {
	_, err := conn.Write(response.Marshal())
	if err != nil {
		fmt.Println("Error writing response: ", err.Error())
		os.Exit(1)
	}
}
